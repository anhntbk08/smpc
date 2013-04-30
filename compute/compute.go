package main
import (
        zmq "github.com/apanda/go-zmq"
        "fmt"
        "flag"
        "os"
        "os/signal"
        sproto "github.com/apanda/smpc/proto"
        "sync"
        redis "github.com/apanda/radix/redis"
        "time"
         "runtime/pprof"
        )
var _ = fmt.Println

func keepAlive () {
    for {
        //fmt.Printf("Alive now at %v\n", time.Now().String())
        time.Sleep(30 * time.Second)
    }

}

type RequestStepPair struct {
    Request int64
    Step int32
}

func MakeRequestStep (req int64, step int32) (ret *RequestStepPair) {
    ret = &RequestStepPair{}
    ret.Request = req
    ret.Step = step
    return ret
}

type ComputePeerState struct {
    SubSock *zmq.Socket
    CoordSock *zmq.Socket
    PeerInSock *zmq.Socket
    PeerOutSocks map[int] *zmq.Socket
    SubChannel *zmq.Channels
    CoordChannel *zmq.Channels
    PeerInChannel *zmq.Channels
    PeerOutChannels map[int] *zmq.Channels
    RedisClient *redis.Client
    Client int
    NumClients int
    ChannelMap map[RequestStepPair] chan *sproto.IntermediateData 
    ChannelLock sync.Mutex
    SquelchTraffic map[RequestStepPair] bool
}

const INITIAL_MAP_CAPACITY int = 1000
const INITIAL_CHANNEL_CAPACITY int = 100

func MakeComputePeerState (client int, numClients int) (*ComputePeerState) {
    state := &ComputePeerState{}
    state.Client = client
    state.NumClients = numClients
    state.PeerOutSocks = make(map[int] *zmq.Socket, numClients)
    state.PeerOutChannels = make(map[int] *zmq.Channels, numClients)
    state.ChannelMap = make(map[RequestStepPair] chan *sproto.IntermediateData, INITIAL_MAP_CAPACITY)
    state.SquelchTraffic = make(map[RequestStepPair] bool, INITIAL_MAP_CAPACITY)
    return state
}

const BUFFER_SIZE int = 10

func (state *ComputePeerState) UnregisterChannelForRequest (request RequestStepPair) {
    //fmt.Printf("Deleting channel %d %d\n", request.Request, request.Step)
    state.ChannelLock.Lock()
    //fmt.Printf("Deleting channel (acquired lock) %d %d\n", request.Request, request.Step)
    defer state.ChannelLock.Unlock()
    //fmt.Printf("Deleting channel (wrote to squelch) %d %d\n", request.Request, request.Step)
    state.SquelchTraffic[request] = true
    delete(state.ChannelMap, request)
    //fmt.Printf("Deleting channel (returning) %d %d\n", request.Request, request.Step)
}

func (state *ComputePeerState) ChannelForRequest (request RequestStepPair) (chan *sproto.IntermediateData) {
    //fmt.Printf("Channel requested for %d %d\n", request.Request, request.Step)
    state.ChannelLock.Lock()
    //fmt.Printf("Channel requested for %d %d (acquired lock)\n", request.Request, request.Step)
    defer state.ChannelLock.Unlock()
    if !state.SquelchTraffic[request] {
        ch := state.ChannelMap[request]
        //fmt.Printf("Channel requested for %d %d (found in map)\n", request.Request, request.Step)
        if ch == nil {
            state.ChannelMap[request] = make(chan *sproto.IntermediateData, INITIAL_CHANNEL_CAPACITY)
            //fmt.Printf("Channel requested for %d %d (created channel)\n", request.Request, request.Step)
            ch = state.ChannelMap[request]
        }
        //fmt.Printf("Channel requested for %d %d (returning)\n", request.Request, request.Step)
        return ch
    } 
    return nil
}

func (state *ComputePeerState) MaybeSendOnChannel (request RequestStepPair, intermediate *sproto.IntermediateData) {
    //fmt.Printf("MaybeSendOnChannel requested for %d %d\n", request.Request, request.Step)
    state.ChannelLock.Lock()
    //fmt.Printf("MaybeSendOnChannel requested for %d %d (acquired lock)\n", request.Request, request.Step)
    defer state.ChannelLock.Unlock()
    if !state.SquelchTraffic[request] {
        ch := state.ChannelMap[request]
        //fmt.Printf("MaybeSendOnChannel requested for %d %d (found in map)\n", request.Request, request.Step)
        if ch == nil {
            state.ChannelMap[request] = make(chan *sproto.IntermediateData, INITIAL_CHANNEL_CAPACITY)
            //fmt.Printf("MaybeSendOnChannel requested for %d %d (created channel)\n", request.Request, request.Step)
            ch = state.ChannelMap[request]
        }
        //fmt.Printf("MaybeSendOnChannel requested for %d %d (found)\n", request.Request, request.Step)
        ch <- intermediate
        //fmt.Printf("MaybeSendOnChannel requested for %d %d (returning sent)\n", request.Request, request.Step)
    }
}

func (state *ComputePeerState) ReceiveFromPeers () {
    defer state.PeerInChannel.Close()
    keepaliveCh := make(chan bool, 1)
    go func(ch chan bool) {
        time.Sleep(5 * time.Second)
        ch <- true
    }(keepaliveCh)
    for {
        //fmt.Printf("Core is now waiting for messages\n")
        select {
            case msg := <- state.PeerInChannel.In():
                //fmt.Println("Message on peer channel")
                intermediate := MsgToIntermediate(msg)
                //fmt.Printf("Core received %d->%d request=%d\n", *intermediate.Client, state.Client, *intermediate.RequestCode)
                if intermediate == nil {
                    panic ("Could not read intermediate message")
                }
                key := MakeRequestStep(*intermediate.RequestCode, *intermediate.Step)
                state.MaybeSendOnChannel (*key, intermediate)
            case <- keepaliveCh:
                fmt.Printf("ReceiveFromPeers is alive\n")
            
        }
    }
}

func (state *ComputePeerState) SharesGet (share string) (int64, bool) {
    // state.ShareLock.RLock()
    // defer state.ShareLock.RUnlock()
    // val := state.Shares[share]
    // has := state.HasShare[share]
    r0 :=  state.RedisClient.Get(share)
    isNil := false
    if r0 == nil {
        isNil = true
    } 
    r, err := r0.Int64()
    if err != nil {
        fmt.Printf("Error %s: %v %v %v %v\n", share, err, isNil, r0.Type, r0.Err)
    }
    return r, (err == nil)
}

func (state *ComputePeerState) SharesSet (share string, value int64) {
    //fmt.Println("SharesSet called, locking")
    // state.ShareLock.Lock()
    // defer state.ShareLock.Unlock()
    // //fmt.Println("SharesSet called, locked")
    // state.Shares[share] = value
    // state.HasShare[share] = true
    resp := state.RedisClient.Set(share, value)
    if resp.Err != nil {
        fmt.Printf("Error setting %s\n", share)
    }
}

func (state *ComputePeerState) SharesDelete (share string) {
    fmt.Printf("Delete %s\n", share)
    state.RedisClient.Del(share)
}

func (state *ComputePeerState) DispatchAction (action *sproto.Action, r chan<- [][]byte) {
    //fmt.Println("Dispatching action")
    var resp *sproto.Response
    switch *action.Action {
        case sproto.Action_Set:
           //fmt.Println("Dispatching SET")
            resp = state.SetValue(action)
        case sproto.Action_Add:
           //fmt.Println("Dispatching ADD")
            resp = state.Add(action)
        case sproto.Action_Retrieve:
           //fmt.Println("Retrieving value")
            resp = state.GetValue(action)
        case sproto.Action_Mul:
           //fmt.Println("Dispatching mul")
            resp = state.Mul(action)
            //fmt.Println("Return from mul")
        case sproto.Action_Cmp:
           //fmt.Println("Dispatching CMP")
            resp = state.Cmp(action)
            //fmt.Println("Return from cmp")
        case sproto.Action_Neq:
           //fmt.Println("Dispatching NEQ")
            resp = state.Neq(action)
            //fmt.Println("Return from NEQ")
        case sproto.Action_Eqz:
           //fmt.Println("Dispatching EQZ")
            resp = state.Eqz(action)
            //fmt.Println("Returning from EQZ")
        case sproto.Action_Neqz:
           //fmt.Println("Dispatching NEQZ")
            resp = state.Neqz(action)
            //fmt.Println("Returning from NEQZ")
        case sproto.Action_Del:
           //fmt.Println("Dispatching DEL")
            resp = state.RemoveValue(action)
            //fmt.Println("Return from DEL")
        case sproto.Action_OneSub:
           //fmt.Println("Dispatching 1SUB")
            resp = state.OneSub(action)
            //fmt.Println("Return from 1SUB")
        case sproto.Action_CmpConst:
           //fmt.Println("Dispatching CmpConst")
            resp = state.CmpConst(action)
           //fmt.Println("Returning from CmpConst")
        case sproto.Action_NeqConst:
           //fmt.Println("Dispatching NeqConst")
            resp = state.NeqConst(action)
           //fmt.Println("Returning from NeqConst")
        case sproto.Action_MulConst:
           //fmt.Println("Dispatching MulConst")
            resp = state.MulConst(action)
           //fmt.Println("Returning from MulConst")
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
    //fmt.Println("Received message from coordination channel")
    action := MsgToAction(msg)
    //fmt.Println("Converted to action")
    if action == nil {
        panic ("Malformed action")
    }
    go state.DispatchAction(action, state.CoordChannel.Out())
}

func EventLoop (config *string, client int, q chan int) {
    configStruct := ParseConfig(config, q) 
    state := MakeComputePeerState(client, len(configStruct.Clients)) 
    redisConfig := redis.DefaultConfig()
    redisConfig.Network = "tcp"
    redisConfig.Address = configStruct.Databases[client].Address
    redisConfig.Database = configStruct.Databases[client].Database
    state.RedisClient = redis.NewClient(redisConfig)
    fmt.Printf("Using redis at %s with db %d\n", configStruct.Databases[client].Address,configStruct.Databases[client].Database)
    // Create the 0MQ context
    ctx, err := zmq.NewContext()
    if err != nil {
        //fmt.Println("Error creating 0mq context: ", err)
        q <- 1
    }
    // Establish the PUB-SUB connection that will be used to direct all the computation clusters
    state.SubSock, err = ctx.Socket(zmq.Sub)
    if err != nil {
        //fmt.Println("Error creating PUB socket: ", err)
        q <- 1
    }
    err = state.SubSock.Connect(configStruct.PubAddress)
    if err != nil {
        //fmt.Println("Error binding PUB socket: ", err)
        q <- 1
    }
    // Establish coordination socket
    state.CoordSock, err = ctx.Socket(zmq.Dealer)
    if err != nil {
        //fmt.Println("Error creating Dealer socket: ", err)
        q <- 1
    }
    err = state.CoordSock.Connect(configStruct.ControlAddress)
    if err != nil {
        //fmt.Println("Error connecting  ", err)
        q <- 1
    }
    state.PeerInSock, err = ctx.Socket(zmq.Router)
    if err != nil {
        //fmt.Println("Error creating peer router socket: ", err)
        q <- 1
    }
    err = state.PeerInSock.Bind(configStruct.Clients[client]) // Set up something to listen to peers
    if err != nil {
        //fmt.Println("Error binding peer router socket")
        q <- 1
    }
    for index, value := range configStruct.Clients {
        if index != client {
            state.PeerOutSocks[index], err= ctx.Socket(zmq.Dealer)
            if err != nil {
                //fmt.Println("Error creating dealer socket: ", err)
                q <- 1
                return
            }
            err  = state.PeerOutSocks[index].Connect(value)
            if err != nil {
                //fmt.Println("Error connection ", err)
                q <- 1
                return
            }
            state.PeerOutChannels[index] = state.PeerOutSocks[index].ChannelsBuffer(BUFFER_SIZE)
            defer state.PeerOutChannels[index].Close()
        }
    }
    state.PeerInChannel = state.PeerInSock.Channels()
    go state.ReceiveFromPeers()
    state.Sync(q)
    state.IntermediateSync(q)
    state.SubSock.Subscribe([]byte("CMD"))
    //fmt.Println("Receiving")
    // We cannot create channels before finalizing the set of subscriptions, since sockets are
    // not thread safe. Hence first sync, then get channels
    state.SubChannel = state.SubSock.ChannelsBuffer(BUFFER_SIZE)
    state.CoordChannel = state.CoordSock.ChannelsBuffer(BUFFER_SIZE)

    defer func() {
        state.CoordChannel.Close()
        state.SubChannel.Close()
        state.SubSock.Close()
        state.CoordSock.Close()
        for idx := range state.PeerOutSocks {
            state.PeerOutSocks[idx].Close()
        }
        ctx.Close()
        //fmt.Println("Closed socket")
    }()
    keepaliveCh := make(chan bool, 1)
    go func(ch chan bool) {
        time.Sleep(5 * time.Second)
        ch <- true
    }(keepaliveCh)
    for true {
        //fmt.Println("Starting to wait")
        select {
            case msg := <- state.SubChannel.In():
                //fmt.Println("Message on SubChannel")
                state.ActionMsg(msg) 
            case msg := <- state.CoordChannel.In():
                //fmt.Println("Message on CoordChannel")
                state.ActionMsg(msg)
            case err = <- state.SubChannel.Errors():
                //fmt.Println("Error in SubChannel", err)
                q <- 1
                return
            case err = <- state.CoordChannel.Errors():
                //fmt.Println("Error in CoordChannel", err)
                q <- 1
                return
            case <- keepaliveCh:
                fmt.Printf("Sub and coord channel receive alive\n")
        }
    }
    q <- 0
}

func main() {
    // Start up by setting up a flag for the configuration file
    config := flag.String("config", "conf", "Configuration file")
    client := flag.Int("peer", 0, "Input peer")
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
    os_channel := make(chan os.Signal)
    signal.Notify(os_channel)
    end_channel := make(chan int)
    go EventLoop(config, *client, end_channel)
    //go keepAlive()
    var status = 0
    select {
        case <- os_channel:
        case status = <- end_channel: 
    }
    // <-signal_channel
    os.Exit(status)
}

