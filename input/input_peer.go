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
        "time"
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

func circuit (state *InputPeerState, topo *Topology, end_channel chan int) {
    val := int64(0)
    _ = val
    //topo := state.MakeTestTopology(end_channel)  
    
    nnhop := make(map[int64] string, len(topo.AdjacencyMatrix))
    elapsed := float64(0)
    iters :=12
    for it := 0; it < iters; it++ {
        t := time.Now()
        nnhop = make(map[int64] string, len(topo.AdjacencyMatrix))
        ch := make(map[int64] chan string, len(topo.AdjacencyMatrix))
        for i := range topo.AdjacencyMatrix {
            ch[i] = state.RunSingleIteration(topo, i, end_channel)
        }
        for i  := range topo.AdjacencyMatrix {
            nnhop[i] = <- ch[i]
        }
        topo.NextHop = nnhop
        elapsed += (time.Since(t).Seconds())
    }

    fmt.Printf("Two round NextHop, should be 2, 2, 2, 1 Time: %f\n", elapsed/float64(iters))
    for ind := range nnhop {
        c42 := state.GetValue(nnhop[ind], end_channel)
        val = <- c42
        fmt.Printf("%d: %d\n", ind, val)
    }
    
    end_channel <- 0
}

func main() {
    // Start up by setting up a flag for the Configuration file
    config := flag.String("config", "conf", "Configuration file")
    topoFile := flag.String("topo", "", "Topology file")
    dest := flag.Int64("dest", 0, "Destination")
    flag.Parse()
    jsonTopo := ParseJsonTopology(topoFile)  
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
                topo := jsonTopo.MakeTopology(state, end_channel)
                if *dest != 0 {
                    ch := state.SetValue(topo.NextHop[*dest], *dest, end_channel)
                    <- ch
                }
                go circuit(state, topo, end_channel)
            case status = <- end_channel: 
                os.Exit(status)
            case <- os_channel:
                os.Exit(status)
        }
    }
}

