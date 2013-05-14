package main
import (
        zmq "github.com/apanda/go-zmq"
        sproto "github.com/apanda/smpc/proto"
        "code.google.com/p/goprotobuf/proto"
        "fmt"
        "flag"
        "os"
        "os/signal"
        "sync"
        "strings"
         "runtime/pprof"
         "time"
         "runtime"
        )
var _ = time.Now
type CoordChannelMessage struct {
    Index int
    Message *sproto.Action
}
type InputPeerState struct {
    ClusterID int64
    ComputeSlaves [][]byte
    Config *Configuration
    RequestID int64
    PubSock *zmq.Socket
    CoordSock *zmq.Socket
    CoordChannel *zmq.Channels
    CoordNaggleChannel chan *CoordChannelMessage 
    PubChannel *zmq.Channels
    PubNaggleChannel chan *sproto.Action
    ChannelMap map[int64] chan *sproto.Response
    ChannelLock sync.RWMutex
}

const INITIAL_MAP_CONSTANT int = 1000
const INITIAL_CHANNEL_SIZE int = 100

func (state *InputPeerState) InitPeerState (clients int) {
    state.ComputeSlaves = make([][]byte, clients)
    state.ChannelMap = make(map[int64] chan *sproto.Response, 1000)
    state.RequestID = 1
    state.CoordNaggleChannel = make(chan *CoordChannelMessage, INITIAL_CHANNEL_SIZE)
    state.PubNaggleChannel = make(chan *sproto.Action, INITIAL_CHANNEL_SIZE)
}

const NAGGLE_SIZE int = 275
const NAGGLE_MULT time.Duration = time.Duration(2)
func ActionToCoordChannelMessage (action *sproto.Action, index int) (*CoordChannelMessage) {
    coordMessage := &CoordChannelMessage{}
    coordMessage.Index = index
    coordMessage.Message = action
    return coordMessage
}

func (state *InputPeerState) ActionToNaggledActionCoord(messageList []*sproto.Action, index int) ([][]byte) {
    msg := make([][]byte, 3)
    msg[0] = state.ComputeSlaves[index]
    msg[1] = []byte("")
    naggle := &sproto.NaggledAction{}
    naggle.Messages = messageList
    var err error
    msg[2], err = proto.Marshal(naggle)
    if err != nil {
        return nil
    }
    return msg
}

func (state *InputPeerState) ActionToNaggledActionPub(messageList []*sproto.Action) ([][]byte) {
    msg := make([][]byte, 2)
    msg[0] = []byte("CMD") 
    naggle := &sproto.NaggledAction{}
    naggle.Messages = messageList
    var err error
    msg[1], err = proto.Marshal(naggle)
    if err != nil {
        return nil
    }
    return msg
}

func (state *InputPeerState) NagglePubChannel () {
    NAGGLE_TIME := NAGGLE_MULT * time.Millisecond
    messageList := make([]*sproto.Action, NAGGLE_SIZE)
    occupied := 0
    timerRunning := false
    timerChan := make(chan bool)
    timerSend := int64(0)
    bufferSend := int64(0)
    timerFunc := func () {
        time.Sleep(NAGGLE_TIME)
        timerChan <- true
    }
    defer func() {
        fmt.Printf("\n\nMessage send stats: Sent %d using TIMER, %d using buffer\n", timerSend, bufferSend)
    }()
    send := func() {
        if occupied > 0 {
            msg := state.ActionToNaggledActionPub(messageList[:occupied])
            state.PubChannel.Out() <- msg
        }
        occupied = 0
    }
    for {
        select {
            case msg := <- state.PubNaggleChannel:
                messageList[occupied] = msg
                occupied++
                if occupied >= NAGGLE_SIZE {
                    bufferSend++
                    //fmt.Printf("Buffer over, sending\n")
                    send()
                } else if !timerRunning {
                    timerRunning = true
                    go timerFunc()
                }
            case <- timerChan:
                timerSend++
                timerRunning = false
                //fmt.Printf("Timer going off, sending\n")
                send()
        }
    }
}

func (state *InputPeerState) NaggleCoordChannel () {
    NAGGLE_TIME := NAGGLE_MULT * time.Millisecond
    messageList := make([][]*sproto.Action, len(state.ComputeSlaves))
    occupied := make([]int, len(state.ComputeSlaves))
    for i := range messageList {
        messageList[i] = make([]*sproto.Action, NAGGLE_SIZE)
        occupied[i] = 0
    }
    timerRunning := false
    timerChan := make(chan bool)
    timerFunc := func () {
        time.Sleep(NAGGLE_TIME)
        timerChan <- true
    }
    timerSend := int64(0)
    bufferSend := int64(0)
    defer func() {
        fmt.Printf("\n\nMessage send stats: Sent %d using TIMER, %d using buffer\n", timerSend, bufferSend)
    }()
    send := func() {
        for i := range messageList {
            if occupied[i] > 0 {
                msg := state.ActionToNaggledActionCoord(messageList[i][:occupied[i]], i)
                state.CoordChannel.Out() <- msg
            }
            occupied[i] = 0
        }
    }
    for {
        select {
            case msg := <- state.CoordNaggleChannel:
                idx := msg.Index
                messageList[idx][occupied[idx]] = msg.Message
                occupied[idx]++
                if occupied[idx] >= NAGGLE_SIZE {
                    bufferSend++
                    //fmt.Printf("Buffer over, sending\n")
                    send()
                } else if !timerRunning {
                    timerRunning = true
                    go timerFunc()
                }

            case <- timerChan:
                timerSend++
                timerRunning = false
                //fmt.Printf("Timr going off, sending\n")
                send()
        }
    }
}

func (state *InputPeerState) GetChannelForRequest (request int64) (chan *sproto.Response) {
    state.ChannelLock.RLock()
    defer state.ChannelLock.RUnlock()
    return state.ChannelMap[request]
}

func (state *InputPeerState) SetChannelForRequest (request int64, c chan *sproto.Response) {
    state.ChannelLock.Lock()
    defer state.ChannelLock.Unlock()
    state.ChannelMap[request] = c
}

func (state *InputPeerState) DelChannelForRequest (request int64) {
    state.ChannelLock.Lock()
    defer state.ChannelLock.Unlock()
    delete(state.ChannelMap, request)
}

func (state *InputPeerState) MessageLoop () {
    for {
        select {
            case msg := <- state.CoordChannel.In():
               nresponse := &sproto.NaggledResponse{}
               err := proto.Unmarshal(msg[2], nresponse)
               if err != nil {
                   panic(fmt.Sprintf("Error unmarshalling response %v\n", err))
               }
               for idx := range nresponse.Messages {
                   go func (response *sproto.Response) {
                       c := state.GetChannelForRequest(*response.RequestCode)
                       if c == nil {
                           fmt.Printf("%d has no associated channel \n", *response.RequestCode)
                       } else {
                           c <- response
                       }
                   } (nresponse.Messages[idx])
               }
        }
    }
}

const BUFFER_SIZE int = 1000
const ZMQ_HWM int = int((^uint32(0) >> 1))
/* The main event loop */
func EventLoop (config *string, state *InputPeerState, q chan int, ready chan bool) {
    // Create the 0MQ context
    ctx, err := zmq.NewContext()
    state.Config = ParseConfig(config, q)
    state.InitPeerState(state.Config.Clients)
    if err != nil {
        fmt.Println("Error creating 0mq context: ", err)
        q <- 1
        return
    }
    // Establish the PUB-SUB connection that will be used to direct all the computation clusters
    state.PubSock, err = ctx.Socket(zmq.Pub)
    state.PubSock.SetHWM(ZMQ_HWM)
    if err != nil {
        fmt.Println("Error creating PUB socket: ", err)
        q <- 1
        return
    }
    err = state.PubSock.Bind(state.Config.PubAddress)
    if err != nil {
        fmt.Println("Error binding PUB socket: ", err)
        q <- 1
        return
    }
    // Establish coordination socket
    state.CoordSock, err = ctx.Socket(zmq.Router)
    state.CoordSock.SetHWM(ZMQ_HWM)
    if err != nil {
        fmt.Println("Error creating REP socket: ", err)
        q <- 1
        return
    }
    // Strict error checking
    //state.CoordSock.SetRouterMandatory()

    err = state.CoordSock.Bind(state.Config.ControlAddress)
    if err != nil {
        fmt.Println("Error binding coordination socket ", err)
        q <- 1
        return
    }
    state.CoordChannel = state.CoordSock.ChannelsBuffer(BUFFER_SIZE) 
    state.PubChannel = state.PubSock.ChannelsBuffer(BUFFER_SIZE)
    state.Sync(q)
    go state.NaggleCoordChannel()
    go state.NagglePubChannel()
    go state.MessageLoop() 
    ready <- true
    // Handle errors from here on out
    for {
        select {
            case err = <- state.CoordChannel.Errors():
                fmt.Println("Coordination error", err)
            case err = <- state.PubChannel.Errors():
                fmt.Println("Publishing error", err)
            // Do nothing
        }
    }

    defer func() {
        state.CoordChannel.Close()
        state.PubChannel.Close()
        state.PubSock.Close()
        state.CoordSock.Close()
        ctx.Close()
        fmt.Println("Closed socket")
    }()
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())
    // Start up by setting up a flag for the Configuration file
    config := flag.String("config", "conf", "Configuration file")
    topoFile := flag.String("topo", "", "Topology file")
    dest := flag.Int64("dest", 0, "Destination")
    cpuprof := flag.String("cpuprofile", "", "write cpu profile")
    flag.Parse()
    if *cpuprof != "" {
        f, err := os.Create(*cpuprof)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            os.Exit(1)
        }
        defer func() {
            fmt.Printf("Closing cpuprof file\n")
            f.Close()
        }()
        pprof.StartCPUProfile(f)
        defer func() {
            fmt.Printf("Stopping CPU profiling\n")
            pprof.StopCPUProfile()
        }()
    }
    os_channel := make(chan os.Signal, 2)
    signal.Notify(os_channel)
    configs := strings.Split(*config, " ")
    state := make([]*InputPeerState, len(configs))
    coordinate_channel := make([]chan bool, len(configs))
    fmt.Printf("Config files :%s\n", *config)
    end_channel := make(chan int, INITIAL_CHANNEL_SIZE)
    var status = 0
    _ = status
    for i := range configs {
        state[i] = &InputPeerState{}
        coordinate_channel[i] = make(chan bool, INITIAL_CHANNEL_SIZE)
        state[i].ClusterID = int64(i) 
        go EventLoop(&configs[i], state[i], end_channel, coordinate_channel[i])
    }
    for ch := range coordinate_channel {
        select {
            case <- coordinate_channel[ch]:
                continue
            case status = <- end_channel: 
                return
            case signal := <- os_channel:
                if (signal == os.Interrupt) ||  (signal == os.Kill) {
                    fmt.Printf("signalling %v\n", signal)
                    panic("signal")
                }
        }
    }
    fmt.Printf("All reported in, going to start\n")
    go circuit(state, topoFile, *dest, end_channel)
    for {
        select {
            case status = <- end_channel: 
                fmt.Printf("Exiting for some reason internal to us")
                return
            case signal := <- os_channel:
                if (signal == os.Interrupt) ||  (signal == os.Kill) {
                    fmt.Printf("signalling %v\n", signal)
                    panic("signal")
                }
        }
    }
}

