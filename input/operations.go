package main
import (
        "github.com/apanda/smpc/core"
        sproto "github.com/apanda/smpc/proto"
        "fmt"
        "code.google.com/p/goprotobuf/proto"
        "sync/atomic"
        )

/* Set value */
func (state *InputPeerState) SetValue (name string, value int64, q chan int) (chan bool) {
    done := make(chan bool, 1) // Buffered so we can be done even if no one is listening
    requestID := atomic.AddInt64(&state.RequestID, 1)
    go func() {
        shares := core.DistributeSecret(value, int32(len(state.ComputeSlaves)))
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
            fmt.Printf("%s[%d] = %d\n", name, index, shares[index])
            msg[2], err = proto.Marshal(action)
            if err != nil {
                fmt.Println("Error marshaling SET message: ", err)
                q <- 1
            }
            state.CoordChannel.Out() <- msg
        }
        status := make(chan *sproto.Response, 1)
        state.SetChannelForRequest(requestID, status)
        received := 0
        for received < len(state.ComputeSlaves) {
           fmt.Printf("Set value waiting for %d compute nodes\n", len(state.ComputeSlaves) - received)
           response := <- status
           fmt.Printf("Received message\n")
           if response.GetRequestCode() == requestID {
               received += 1
           } else {
               fmt.Printf("Set value saw an unexpected message\n")
           }
        }
        state.DelChannelForRequest(requestID)
        fmt.Println("Done setting")
        done <- true
    }()
    return done
}

/* Get value */
func (state *InputPeerState) GetValue (name string, q chan int) (chan int64){
    done := make(chan int64, 1) // Buffered so we can be done even if no one is litening
    requestID := atomic.AddInt64(&state.RequestID, 1)
    go func() {
        for _, value := range state.ComputeSlaves {
            var err error
            msg := make([][]byte, 3)
            msg[0] = value
            msg[1] = []byte("")
            action := &sproto.Action{}
            t := sproto.Action_Retrieve
            action.Action = &t
            action.Result = &name
            action.RequestCode = &requestID
            msg[2], err = proto.Marshal(action)
            if err != nil {
                fmt.Println("Error marshaling GET message: ", err)
                q <- 1
            }
            state.CoordChannel.Out() <- msg
        }
        received := 0
        shares := make([]int64, len(state.ComputeSlaves))
        got_share := make([]bool, len(state.ComputeSlaves))
        status := make(chan *sproto.Response, 1)
        state.SetChannelForRequest(requestID, status)
        shares_found := 0
        for received < len(state.ComputeSlaves) {
           response := <- status
           if response.GetRequestCode() == requestID {
               switch *response.Status {
                   case sproto.Response_Val:
                       client := int(*response.Client)
                       shares[client] = *response.Share
                       got_share[client] = true
                       fmt.Printf("%s[%d] = %d\n", name, client, shares[client])
                       received += 1
                       shares_found += 1
                   case sproto.Response_Error:
                      client := int(*response.Client)
                      shares[client] = int64(0)
                      got_share[client] = false
                      fmt.Printf("%s[%d] not found\n", name, client)
                      received += 1
                   default:
                      panic(fmt.Sprintf("Request %d saw an unusual response", requestID))
               }
           } else {
                panic("Get value saw an unexpected message")
           }
        }
        fmt.Println("Done getting")
        state.DelChannelForRequest(requestID)
        if shares_found > ((len(state.ComputeSlaves) - 1) >> 1) {
            done <- core.ReconstructSecret(&shares, &got_share, int32(len(state.ComputeSlaves)))
        }  else {
            done <- int64(0)
        }
    }()
    return done
}

func (state *InputPeerState) Add (result string, left string, right string, q chan int) (chan bool){
    done := make(chan bool, 1) //Buffer to avoid hangs
    go func() {
        requestID := atomic.AddInt64(&state.RequestID, 1) 
        msg := make([][]byte, 2)
        msg[0] = []byte("CMD")
        action := &sproto.Action{}
        t := sproto.Action_Add
        action.Action = &t
        action.Result = &result
        action.RequestCode = &requestID
        action.Share0 = &left
        action.Share1 = &right
        var err error
        msg[1], err = proto.Marshal(action)
        if err != nil {
            fmt.Println("Error marshaling ADD message: ", err)
            q <- 1
        }
        state.PubChannel.Out() <- msg
        received := 0
        status := make(chan *sproto.Response, 1)
        state.SetChannelForRequest(requestID, status)
        for received < len(state.ComputeSlaves) {
            <- status
            received += 1
        }
        state.DelChannelForRequest(requestID)
        done <- true
        return
    }()
    return done
}

func (state *InputPeerState) Mul (result string, left string, right string, q chan int) (chan bool){
    done := make(chan bool, 1) //Buffer to avoid hangs
    go func() {
        requestID := atomic.AddInt64(&state.RequestID, 1) 
        msg := make([][]byte, 2)
        msg[0] = []byte("CMD")
        action := &sproto.Action{}
        t := sproto.Action_Mul
        action.Action = &t
        action.Result = &result
        action.RequestCode = &requestID
        action.Share0 = &left
        action.Share1 = &right
        var err error
        msg[1], err = proto.Marshal(action)
        if err != nil {
            fmt.Println("Error marshaling ADD message: ", err)
            q <- 1
        }
        state.PubChannel.Out() <- msg
        received := 0
        status := make(chan *sproto.Response, 1)
        state.SetChannelForRequest(requestID, status)
        for received < len(state.ComputeSlaves) {
            <- status
            received += 1
            fmt.Println("Returned mul return")
        }
        state.DelChannelForRequest(requestID)
        done <- true
        return
    }()
    return done
}

