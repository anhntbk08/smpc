package main
import (
        zmq "github.com/apanda/go-zmq"
        "fmt"
        "flag"
        "os"
        "os/signal"
        sproto "github.com/apanda/smpc/proto"
        "sync"
        )
type ComputePeerState struct {
    SubSock *zmq.Socket
    CoordSock *zmq.Socket
    PeerInSock *zmq.Socket
    PeerOutSocks map[int] *zmq.Socket
    SubChannel *zmq.Channels
    CoordChannel *zmq.Channels
    PeerInChannel *zmq.Channels
    PeerOutChannels map[int] *zmq.Channels
    Shares map[string] int64
    HasShare map[string] bool
    ShareLock sync.RWMutex
    Client int
}

const INITIAL_MAP_CAPACITY int = 1000

func MakeComputePeerState (client int) (*ComputePeerState) {
    state := &ComputePeerState{}
    state.Client = client
    state.Shares = make(map[string] int64, INITIAL_MAP_CAPACITY)
    state.HasShare = make(map[string] bool, INITIAL_MAP_CAPACITY)
    return state
}

const BUFFER_SIZE int = 10

func (state *ComputePeerState) SharesGet (share string) (int64, bool) {
    state.ShareLock.RLock()
    defer state.ShareLock.RUnlock()
    val := state.Shares[share]
    has := state.HasShare[share]
    return val, has
}

func (state *ComputePeerState) SharesSet (share string, value int64) {
    fmt.Println("SharesSet called, locking")
    state.ShareLock.Lock()
    defer state.ShareLock.Unlock()
    fmt.Println("SharesSet called, locked")
    state.Shares[share] = value
    fmt.Printf("Set %v to %v", share, value)
    state.HasShare[share] = true
    fmt.Printf("Set %v to %v", share, true)
}

func (state *ComputePeerState) DispatchAction (action *sproto.Action, r chan<- [][]byte) {
    fmt.Println("Dispatching action")
    var resp *sproto.Response
    switch *action.Action {
        case sproto.Action_Set:
            fmt.Println("Dispatching SET")
            resp = state.SetValue(action)
        case sproto.Action_Add:
            fmt.Println("Dispatching ADD")
            resp = state.Add(action)
        case sproto.Action_Retrieve:
            fmt.Println("Retrieving value")
            resp = state.GetValue(action)
        default:
            fmt.Println("Unimplemented action")
            resp = state.DefaultAction(action)
    }
    respB := ResponseToMsg(resp)
    if resp == nil {
        panic ("Malformed response")
    }
    r <- respB
}

func (state *ComputePeerState) ActionMsg (msg [][]byte) {
    fmt.Println("Received message from coordination channel")
    action := MsgToAction(msg)
    fmt.Println("Converted to action")
    if action == nil {
        panic ("Malformed action")
    }
    go state.DispatchAction(action, state.CoordChannel.Out())
}

func (state *ComputePeerState) PeerMsg (msg [][] byte) {

}

func EventLoop (config *string, client int, q chan int) {
    configStruct := ParseConfig(config, q) 
    state := MakeComputePeerState(client) 
    // Create the 0MQ context
    ctx, err := zmq.NewContext()
    if err != nil {
        fmt.Println("Error creating 0mq context: ", err)
        q <- 1
    }
    // Establish the PUB-SUB connection that will be used to direct all the computation clusters
    state.SubSock, err = ctx.Socket(zmq.Sub)
    if err != nil {
        fmt.Println("Error creating PUB socket: ", err)
        q <- 1
    }
    err = state.SubSock.Connect(configStruct.PubAddress)
    if err != nil {
        fmt.Println("Error binding PUB socket: ", err)
        q <- 1
    }
    // Establish coordination socket
    state.CoordSock, err = ctx.Socket(zmq.Dealer)
    if err != nil {
        fmt.Println("Error creating Dealer socket: ", err)
        q <- 1
    }
    err = state.CoordSock.Connect(configStruct.ControlAddress)
    if err != nil {
        fmt.Println("Error connecting  ", err)
        q <- 1
    }
    state.PeerInSock, err = ctx.Socket(zmq.Router)
    if err != nil {
        fmt.Println("Error creating peer router socket: ", err)
        q <- 1
    }
    err = state.PeerInSock.Bind(configStruct.Clients[client]) // Set up something to listen to peers
    if err != nil {
        fmt.Println("Error binding peer router socket")
        q <- 1
    }
    state.PeerOutSocks = make(map[int] *zmq.Socket, len(configStruct.Clients))
    state.PeerOutChannels = make(map[int] *zmq.Channels, len(configStruct.Clients))
    for index, value := range configStruct.Clients {
        if index != client {
            state.PeerOutSocks[index], err= ctx.Socket(zmq.Dealer)
            if err != nil {
                fmt.Println("Error creating dealer socket: ", err)
                q <- 1
                return
            }
            err  = state.PeerOutSocks[index].Connect(value)
            if err != nil {
                fmt.Println("Error connection ", err)
                q <- 1
                return
            }
            state.PeerOutChannels[index] = state.PeerOutSocks[index].ChannelsBuffer(BUFFER_SIZE)
        }
    }
    state.Sync(q)
    state.SubSock.Subscribe([]byte("CMD"))
    fmt.Println("Receiving")
    // We cannot create channels before finalizing the set of subscriptions, since sockets are
    // not thread safe. Hence first sync, then get channels
    state.SubChannel = state.SubSock.ChannelsBuffer(BUFFER_SIZE)
    state.CoordChannel = state.CoordSock.ChannelsBuffer(BUFFER_SIZE)
    state.PeerInChannel = state.PeerInSock.ChannelsBuffer(BUFFER_SIZE)
    defer func() {
        state.SubSock.Close()
        state.CoordSock.Close()
        ctx.Close()
        fmt.Println("Closed socket")
    }()
    for true {
        fmt.Println("Starting to wait")
        select {
            case msg := <- state.SubChannel.In():
                state.ActionMsg(msg) 
            case msg := <- state.CoordChannel.In():
                state.ActionMsg(msg)
            case msg := <- state.PeerInChannel.In():
                state.PeerMsg(msg)                
            case err = <- state.SubChannel.Errors():
                fmt.Println("Error in SubChannel", err)
                q <- 1
                return
            case err = <- state.CoordChannel.Errors():
                fmt.Println("Error in CoordChannel", err)
                q <- 1
                return
        }
    }
    q <- 0
}

func main() {
    // Start up by setting up a flag for the configuration file
    config := flag.String("config", "conf", "Configuration file")
    client := flag.Int("peer", 0, "Input peer")
    flag.Parse()
    os_channel := make(chan os.Signal)
    signal.Notify(os_channel)
    end_channel := make(chan int)
    go EventLoop(config, *client, end_channel)
    var status = 0
    select {
        case <- os_channel:
        case status = <- end_channel: 
    }
    // <-signal_channel
    os.Exit(status)
}

