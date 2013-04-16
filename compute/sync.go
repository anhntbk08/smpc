package main
import (
        //"github.com/apanda/smpc/core"
        "time"
        "fmt"
        "encoding/binary"
        sproto "github.com/apanda/smpc/proto"
        )
var _ = fmt.Println
func (state *ComputePeerState) Sync (q chan int) {
    // Subscribe to HELO messages
    state.SubSock.Subscribe([]byte("HELO"))
    //fmt.Println("Starting to wait to receive HELO")
    // Wait to receive one HELO message
    _, err := state.SubSock.Recv()
    if err != nil {
        //fmt.Println("Receiving over subscription socket failed", err)
        q <- 1
        return
    }
    //fmt.Println("Received HELO, client ", int64(state.Client))
    // HELO messages are now useless, unsubscribe
    state.SubSock.Unsubscribe([]byte("HELO"))

    // Inform master of our presence on the network
    resp := make([][]byte, 1)
    resp[0] = make([]byte, binary.MaxVarintLen32)
    binary.PutVarint(resp[0], int64(state.Client))
    r, _ := binary.Varint(resp[0])
    _ = r
    //fmt.Printf("Client = %d\n", r)
    err = state.CoordSock.Send(resp)
    if err != nil {
        //fmt.Println("Error sending on coordination socket", err)
        q <- 1
    }
}

func (state *ComputePeerState) IntermediateSync (q chan int) {
    beaconReceived := make([]bool, state.NumClients)
    clientSeen := make([]bool, state.NumClients)
    beaconReceived[state.Client] = true
    clientSeen[state.Client] = true
    
    beacon := &sproto.IntermediateData{}
    rcode := int64(0)
    step := int32(1)
    beacon.RequestCode = &rcode
    beacon.Step = &step
    client := int32(state.Client)
    beacon.Client = &client
    t := sproto.IntermediateData_SyncBeacon
    beacon.Type  = &t

    beaconRcvd := &sproto.IntermediateData{}
    beaconRcvd.RequestCode = &rcode
    beaconRcvd.Step = &step
    beaconRcvd.Client = &client
    t2 := sproto.IntermediateData_SyncBeaconReceived
    beaconRcvd.Type  = &t2
    done := false
    //fmt.Println("SYNC Entering intermediate sync")
    for !done {
        for i, r := range beaconReceived {
            if !r {
                msg := IntermediateToMsg(beacon)
                state.PeerOutChannels[i].Out() <- msg
            }
        }
        sleep := make(chan bool, 1)
        go func() {
            time.Sleep(10 * time.Millisecond)
            sleep <- true
            
        }()
        ch := state.ChannelForRequest(*MakeRequestStep(rcode,step))
        select {
            case rcvd := <- ch :
                clientSeen[*rcvd.Client] = true
                if *rcvd.Type ==  sproto.IntermediateData_SyncBeaconReceived {
                    beaconReceived[*rcvd.Client] = true
                    //fmt.Printf("SYNC %d -> %d beacon received\n", *rcvd.Client, state.Client)
                } else {
                    state.PeerOutChannels[int(*rcvd.Client)].Out() <- IntermediateToMsg(beaconRcvd)
                    //fmt.Printf("SYNC %d -> %d beacon\n", *rcvd.Client, state.Client)
                }
            case <- sleep:
        }
        done = true
        for _, v := range beaconReceived {
            done = done && v
        }
    }
    
    state.UnregisterChannelForRequest(*MakeRequestStep(rcode, step))
    //fmt.Println("SYNC Exiting intermediate sync")
}
