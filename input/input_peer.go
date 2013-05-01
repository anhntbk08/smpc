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
type InputPeerState struct {
    ClusterID int64
    ComputeSlaves [][]byte
    Config *Configuration
    RequestID int64
    PubSock *zmq.Socket
    CoordSock *zmq.Socket
    CoordChannel *zmq.Channels
    PubChannel *zmq.Channels
    ChannelMap map[int64] chan *sproto.Response
    ChannelLock sync.RWMutex
}

const INITIAL_MAP_CONSTANT int = 1000
const INITIAL_CHANNEL_SIZE int = 100

func (state *InputPeerState) InitPeerState (clients int) {
    state.ComputeSlaves = make([][]byte, clients)
    state.ChannelMap = make(map[int64] chan *sproto.Response, 1000)
    state.RequestID = 1
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
               response := &sproto.Response{}
               err := proto.Unmarshal(msg[2], response)
               if err != nil {
                   panic(fmt.Sprintf("Error unmarshalling response %v\n", err))
               }
               c := state.GetChannelForRequest(*response.RequestCode)
               if c == nil {
                   fmt.Printf("%d has no associated channel \n", *response.RequestCode)
               } else {
                   c <- response
               }
        }
    }
}

const BUFFER_SIZE int = 1000
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
    if err != nil {
        fmt.Println("Error creating REP socket: ", err)
        q <- 1
        return
    }
    // Strict error checking
    state.CoordSock.SetRouterMandatory()

    err = state.CoordSock.Bind(state.Config.ControlAddress)
    if err != nil {
        fmt.Println("Error binding coordination socket ", err)
        q <- 1
        return
    }
    state.CoordChannel = state.CoordSock.ChannelsBuffer(BUFFER_SIZE) 
    state.PubChannel = state.PubSock.ChannelsBuffer(BUFFER_SIZE)
    state.Sync(q)
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
        pprof.StartCPUProfile(f)
        defer pprof.StopCPUProfile()
    }
    os_channel := make(chan os.Signal, 2)
    signal.Notify(os_channel)
    configs := strings.Split(*config, " ")
    state := make([]*InputPeerState, len(configs))
    coordinate_channel := make([]chan bool, len(configs))
    fmt.Printf("Config files :%s\n", *config)
    end_channel := make(chan int, INITIAL_CHANNEL_SIZE)
    var status = 0
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
                os.Exit(status)
            case <- os_channel:
                panic("signal")
        }
    }
    fmt.Printf("All reported in, going to start\n")
    go circuit(state, topoFile, *dest, end_channel)
    for {
        select {
            case status = <- end_channel: 
                fmt.Printf("Exiting for some reason internal to us")
                os.Exit(status)
            case sig := <- os_channel:
                panic("signal")
        }
    }
}

