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
        )

type InputPeerState struct {
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

const BUFFER_SIZE int = 10
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
    select {
        case err = <- state.CoordChannel.Errors():
            fmt.Println("Coordination error", err)
        case err = <- state.PubChannel.Errors():
            fmt.Println("Publishing error", err)
        // Do nothing
    }

    q <- 1
    defer func() {
        state.PubSock.Close()
        state.CoordSock.Close()
        ctx.Close()
        fmt.Println("Closed socket")
    }()
}

func circuit (state *InputPeerState, end_channel chan int) {
    c := state.SetValue("food", int64(1), end_channel)
    <- c
    c = state.SetValue("pizza", int64(1), end_channel)
    <- c
    c = state.Add("delicious", "food", "pizza", end_channel)
    <- c
    fmt.Println("Running mul")
    c = state.Mul("dd", "delicious", "food", end_channel)
    c1 := state.GetValue("delicious", end_channel)
    val := <- c1
    fmt.Printf("delicious = %d\n", val)
    c1 = state.GetValue("food", end_channel)
    val = <-c1
    fmt.Printf("food = %d\n", val)
    <- c
    c1 = state.GetValue("dd", end_channel)
    val = <- c1
    fmt.Printf("dd = %d\n", val)
    c = state.Mul("dd", "dd", "dd", end_channel)
    <- c
    c1 = state.GetValue("dd", end_channel)
    val = <- c1
    fmt.Printf("dd = %d\n", val)
    c = state.Mul("dd", "dd", "dd", end_channel)
    <- c
    c = state.Mul("dd", "dd", "dd", end_channel)
    <- c
    //c = state.Mul("dd", "dd", "dd", end_channel)
    //<- c
    c = state.Mul("dd", "dd", "dd", end_channel)
    <- c
    c1 = state.GetValue("dd", end_channel)
    val = <- c1
    fmt.Printf("dd = %d\n", val)

    c = state.SetValue("a", int64(100), end_channel)
    <- c
    c = state.SetValue("b", int64(100), end_channel)
    <- c
    c = state.SetValue("c", int64(20), end_channel)
    <- c
    c = state.SetValue("d", int64(0), end_channel)
    <- c
    c = state.Cmp("avb", "a", "b", end_channel)
    c2 := state.Cmp("cvd", "c", "d", end_channel)
    c3 := state.Cmp("avd", "a", "d", end_channel)
    <- c3
    <- c
    <- c2
    c1 = state.GetValue("avb", end_channel)
    val = <- c1
    fmt.Printf("Comparison a == b (should be 1) = %d\n", val)
    c1 = state.GetValue("cvd", end_channel)
    val = <- c1
    fmt.Printf("c == d? (should be 0) = %d\n", val)
    c1 = state.GetValue("avd", end_channel)
    val = <- c1
    fmt.Printf("a == d? (should be 0) = %d\n", val)
    end_channel <- 0
}

func main() {
    // Start up by setting up a flag for the Configuration file
    config := flag.String("config", "conf", "Configuration file")
    flag.Parse()
    os_channel := make(chan os.Signal)
    signal.Notify(os_channel)
    end_channel := make(chan int, 1)
    coordinate_channel := make(chan bool)
    state := &InputPeerState{}
    go EventLoop(config, state, end_channel, coordinate_channel)
    var status = 0
    for {
        select {
            case <- coordinate_channel:
                // Now ready to execute
                go circuit(state, end_channel)
            case status = <- end_channel: 
                os.Exit(status)
            case <- os_channel:
                os.Exit(status)
        }
    }
}

