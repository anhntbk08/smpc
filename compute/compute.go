package main
import (
        "github.com/apanda/smpc/core"
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
    SubChannel *zmq.Channels
    CoordChannel *zmq.Channels
    Shares map[string] int64
    ShareLock sync.RWMutex
    OutstandingRequests sync.WaitGroup
    Client int

}

const BUFFER_SIZE int = 10

func (state *ComputePeerState) SharesGet (share string) (int64) {
    state.ShareLock.RLock()
    defer state.ShareLock.RUnlock()
    return state.Shares[share]
}

func (state *ComputePeerState) SharesSet (share string, value int64) {
    state.ShareLock.Lock()
    defer state.ShareLock.Unlock()
    state.Shares[share] = value
}

// Set the value of a share
func (state *ComputePeerState) SetValue (action *sproto.Action) (*sproto.Response) {
    fmt.Println("Setting value ", *action.Result, *action.Value)
    result := *action.Result
    val := *action.Value
    //state.Shares[result] = int64(val)
    state.SharesSet(result, int64(val))
    fmt.Println("Set map value, preparing RESPONSE")
    resp := &sproto.Response{}
    rcode := action.GetRequestCode()
    fmt.Println("Set map value, set response code, code is ", rcode)
    resp.RequestCode = &rcode
    status := sproto.Response_OK
    resp.Status = &status
    client := int32(state.Client)
    resp.Client = &client
    fmt.Println("Done setting")
    return resp
}

// Retrieve the value of a share
func (state *ComputePeerState) GetValue (action *sproto.Action) (*sproto.Response) {
    state.OutstandingRequests.Done() // First remove this particular request from what is outstanding
    state.OutstandingRequests.Wait() // Then wait for everything else to finish
    state.OutstandingRequests.Add(1) // Then go back to accounting for this request
    result := *action.Result
    val := state.SharesGet(result)
    fmt.Println("Got the map value, preparing RESPONSE")
    resp := &sproto.Response{}
    rcode := action.GetRequestCode()
    fmt.Println("Set map value, set response code, code is ", rcode)
    resp.RequestCode = &rcode
    resp.Share = &val
    status := sproto.Response_Val
    resp.Status = &status
    client := int32(state.Client)
    resp.Client = &client
    fmt.Println("Done setting")
    return resp
}

// Add two shares
func (state *ComputePeerState) Add (action *sproto.Action) (*sproto.Response) {
    fmt.Println("Adding two values")
    result := *action.Result
    share0 := *action.Share0
    share1 := *action.Share1
    // Maybe do this automically
    state.SharesSet(result, core.Add(state.SharesGet(share0), state.SharesGet(share1)))
    resp := &sproto.Response{}
    rcode := action.GetRequestCode()
    resp.RequestCode = &rcode
    status := sproto.Response_OK
    resp.Status = &status
    client := int32(state.Client)
    resp.Client = &client
    fmt.Println("Done Adding")
    return resp
}

func (state *ComputePeerState) DispatchAction (action *sproto.Action) (*sproto.Response) {
    state.OutstandingRequests.Add(1)
    defer state.OutstandingRequests.Done()
    fmt.Println("Dispatching action")
    switch *action.Action {
        case sproto.Action_Set:
            fmt.Println("Dispatching SET")
            return state.SetValue(action)
        case sproto.Action_Add:
            fmt.Println("Dispatching ADD")
            return state.Add(action)
        case sproto.Action_Retrieve:
            fmt.Println("Retrieving value")
            return state.GetValue(action)
        default:
            fmt.Println("Unimplemented action")
            return nil
    }
    return nil
}

func (state *ComputePeerState) CoordMsg (msg [][]byte, q chan int) {
    fmt.Println("Received message from coordination channel")
    action := MsgToAction(msg)
    fmt.Println("Converted to action")
    if action == nil {
        q <- 1
        return
    }
    resp :=  ResponseToMsg(state.DispatchAction(action))
    fmt.Println("Sending response, ", len(resp))
    if resp == nil {
        q <- 1
        return
    }
    state.CoordChannel.Out() <- resp
}

func (state *ComputePeerState) SubMsg (msg [][]byte, q chan int) {
    fmt.Println("Received message from coordination channel")
    action := MsgToAction(msg)
    fmt.Println("Converted to action")
    if action == nil {
        q <- 1
        return
    }
    go state.DispatchAction(action)
}

func EventLoop (config *string, client int, q chan int) {
    configStruct := ParseConfig(config, q) 
    // Create the 0MQ context
    ctx, err := zmq.NewContext()
    if err != nil {
        fmt.Println("Error creating 0mq context: ", err)
        q <- 1
    }
    state := &ComputePeerState{}
    state.Client = client
    state.Shares = make(map[string] int64, 1000)
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
    state.Sync(q)
    state.SubSock.Subscribe([]byte("CMD"))
    fmt.Println("Receiving")
    // We cannot create channels before finalizing the set of subscriptions, since sockets are
    // not thread safe. Hence first sync, then get channels
    state.SubChannel = state.SubSock.ChannelsBuffer(BUFFER_SIZE)
    state.CoordChannel = state.CoordSock.ChannelsBuffer(BUFFER_SIZE)
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
                state.SubMsg(msg, q) 
            case msg := <- state.CoordChannel.In():
                state.CoordMsg(msg, q)
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

