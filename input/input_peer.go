package main
import (
        "github.com/apanda/smpc/core"
        sproto "github.com/apanda/smpc/proto"
        zmq "github.com/apanda/go-zmq"
        "fmt"
        "flag"
        "os"
        "os/signal"
        "code.google.com/p/goprotobuf/proto"
        "sync/atomic"
        )

type InputPeerState struct {
    ComputeSlaves [][]byte
    Config *Configuration
    RequestID int64
    PubSock *zmq.Socket
    CoordSock *zmq.Socket
    CoordChannel *zmq.Channels
    PubChannel *zmq.Channels
}

func (state *InputPeerState) InitPeerState (clients int) {
    state.ComputeSlaves = make([][]byte, clients)
}

/* Set value */
func (state *InputPeerState) SetValue (name string, value int64, q chan int) {
    shares := core.DistributeSecret(value, int32(len(state.ComputeSlaves)))
    requestID := atomic.AddInt64(&state.RequestID, 1)
    for index, value := range state.ComputeSlaves {
        var err error
        msg := make([][]byte, 3)
        msg[0] = value
        msg[1] = []byte("")
        action := &sproto.Action{}
        t := sproto.Action_Set
        action.Action = &t
        action.Result = &name
        action.RequestCode = &requestID
        action.Value = &shares[index]
        msg[2], err = proto.Marshal(action)
        if err != nil {
            fmt.Println("Error marshaling SET message: ", err)
            q <- 1
        }
        state.CoordChannel.Out() <- msg
    }
    received := 0
    for received < len(state.ComputeSlaves) {
       fmt.Printf("Set value waiting for %d compute nodes\n", len(state.ComputeSlaves) - received)
       msg := <- state.CoordChannel.In()
       response := &sproto.Response{}
       err := proto.Unmarshal(msg[2], response)
       if err != nil {
           fmt.Println("Error marshaling SET message: ", err)
           q <- 1
       }
       if response.GetRequestCode() == requestID {
           received += 1
       } else {
           fmt.Printf("Set value saw an unexpected message")
       }
    }
    fmt.Println("Done setting")
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
    state.SetValue("food", int64(5), end_channel)
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

