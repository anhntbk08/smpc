package main
import (
        "github.com/apanda/smpc/core"
        sproto "github.com/apanda/smpc/proto"
        "fmt"
        "sync/atomic"
        "runtime"
        )
func (state *InputPeerState) SetRawValue (name string, shares []int64, requestID int64, q chan int) {
    status := make(chan *sproto.Response, INITIAL_CHANNEL_SIZE)
    state.SetChannelForRequest(requestID, status)
    for index, _ := range state.ComputeSlaves {
        action := &sproto.Action{}
        t := sproto.Action_Set
        action.Action = &t
        action.Result = &name
        action.RequestCode = &requestID
        action.Value = &shares[index]
        //fmt.Printf("%s[%d] = %d\n", name, index, shares[index])
        state.CoordNaggleChannel <- ActionToCoordChannelMessage (action, index)  
        //state.CoordChannel.Out() <- msg
    }
    received := 0
    for received < len(state.ComputeSlaves) {
       //fmt.Printf("Set value waiting for %d compute nodes\n", len(state.ComputeSlaves) - received)
       response := <- status
       //fmt.Printf("Received message\n")
       if response.GetRequestCode() == requestID {
           received += 1
       } else {
           fmt.Printf("Set value saw an unexpected message\n")
       }
    }
    state.DelChannelForRequest(requestID)
}

/* Set value */
func (state *InputPeerState) SetValue (name string, value int64, q chan int) (chan bool) {
    done := make(chan bool, INITIAL_CHANNEL_SIZE) // Buffered so we can be done even if no one is listening
    requestID := atomic.AddInt64(&state.RequestID, 1)
    go func() {
        shares := core.DistributeSecret(value, int32(len(state.ComputeSlaves)))
        //fmt.Println("Done setting")
        state.SetRawValue (name, shares, requestID, q)
        done <- true
        runtime.Goexit()
    }()
    return done
}

func (state *InputPeerState) GetRawValue (name string, requestID int64,  q chan int) ([]int64, int, []bool) {
    status := make(chan *sproto.Response, INITIAL_CHANNEL_SIZE)
    state.SetChannelForRequest(requestID, status)
    for index, _ := range state.ComputeSlaves {
        action := &sproto.Action{}
        t := sproto.Action_Retrieve
        action.Action = &t
        action.Result = &name
        action.RequestCode = &requestID
        state.CoordNaggleChannel <- ActionToCoordChannelMessage (action, index)  
    }
    received := 0
    shares := make([]int64, len(state.ComputeSlaves))
    got_share := make([]bool, len(state.ComputeSlaves))
    shares_found := 0
    for received < len(state.ComputeSlaves) {
       response := <- status
       if response.GetRequestCode() == requestID {
           switch *response.Status {
               case sproto.Response_Val:
                   client := int(*response.Client)
                   shares[client] = *response.Share
                   got_share[client] = true
                   //fmt.Printf("%s[%d] = %d\n", name, client, shares[client])
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
    //fmt.Println("Done getting")
    state.DelChannelForRequest(requestID)
    return shares, shares_found, got_share
}

/* Get value */
func (state *InputPeerState) GetValue (name string, q chan int) (chan int64){
    done := make(chan int64, 1) // Buffered so we can be done even if no one is litening
    requestID := atomic.AddInt64(&state.RequestID, 1)
    go func() {
        shares, shares_found, got_share := state.GetRawValue (name, requestID, q)
        if shares_found > ((len(state.ComputeSlaves) - 1) >> 1) {
            done <- core.ReconstructSecret(&shares, &got_share, int32(len(state.ComputeSlaves)))
        }  else {
            done <- int64(0)
        }
        runtime.Goexit()
    }()
    return done
}

func (state *InputPeerState) Add (result string, left string, right string, q chan int) (chan bool){
    done := make(chan bool, 1) //Buffer to avoid hangs
    go func() {
        requestID := atomic.AddInt64(&state.RequestID, 1) 
        status := make(chan *sproto.Response, INITIAL_CHANNEL_SIZE)
        state.SetChannelForRequest(requestID, status)
        action := &sproto.Action{}
        t := sproto.Action_Add
        action.Action = &t
        action.Result = &result
        action.RequestCode = &requestID
        action.Share0 = &left
        action.Share1 = &right
        state.PubNaggleChannel <- action
        received := 0
        for received < len(state.ComputeSlaves) {
            <- status
            received += 1
        }
        state.DelChannelForRequest(requestID)
        done <- true
        runtime.Goexit()
    }()
    return done
}

func (state *InputPeerState) Mul (result string, left string, right string, q chan int) (chan bool){
    done := make(chan bool, 1) //Buffer to avoid hangs
    go func() {
        requestID := atomic.AddInt64(&state.RequestID, 1) 
        status := make(chan *sproto.Response, INITIAL_CHANNEL_SIZE)
        state.SetChannelForRequest(requestID, status)
        action := &sproto.Action{}
        t := sproto.Action_Mul
        action.Action = &t
        action.Result = &result
        action.RequestCode = &requestID
        action.Share0 = &left
        action.Share1 = &right
        state.PubNaggleChannel <- action
        received := 0
        for received < len(state.ComputeSlaves) {
            <- status
            received += 1
            //fmt.Println("Returned mul return")
        }
        state.DelChannelForRequest(requestID)
        done <- true
        runtime.Goexit()
    }()
    return done
}

func (state *InputPeerState) Cmp (result string, left string, right string, q chan int) (chan bool){
    done := make(chan bool, 1) //Buffer to avoid hangs
    go func() {
        requestID := atomic.AddInt64(&state.RequestID, 1) 
        status := make(chan *sproto.Response, INITIAL_CHANNEL_SIZE)
        state.SetChannelForRequest(requestID, status)
        action := &sproto.Action{}
        t := sproto.Action_Cmp
        action.Action = &t
        action.Result = &result
        action.RequestCode = &requestID
        action.Share0 = &left
        action.Share1 = &right
        state.PubNaggleChannel <- action
        received := 0
        for received < len(state.ComputeSlaves) {
            <- status
            received += 1
            //fmt.Println("Returned cmp return")
        }
        state.DelChannelForRequest(requestID)
        done <- true
        runtime.Goexit()
    }()
    return done
}

func (state *InputPeerState) Neq (result string, left string, right string, q chan int) (chan bool) {
    done := make(chan bool, 1) //Buffer to avoid hangs
    go func() {
        requestID := atomic.AddInt64(&state.RequestID, 1) 
        status := make(chan *sproto.Response, INITIAL_CHANNEL_SIZE)
        state.SetChannelForRequest(requestID, status)
        action := &sproto.Action{}
        t := sproto.Action_Neq
        action.Action = &t
        action.Result = &result
        action.RequestCode = &requestID
        action.Share0 = &left
        action.Share1 = &right
        state.PubNaggleChannel <- action
        received := 0
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

func (state *InputPeerState) Neqz (result string, left string, q chan int) (chan bool) {
    done := make(chan bool, 1) //Buffer to avoid hangs
    go func() {
        requestID := atomic.AddInt64(&state.RequestID, 1) 
        status := make(chan *sproto.Response, INITIAL_CHANNEL_SIZE)
        state.SetChannelForRequest(requestID, status)
        action := &sproto.Action{}
        t := sproto.Action_Neqz
        action.Action = &t
        action.Result = &result
        action.RequestCode = &requestID
        action.Share0 = &left
        state.PubNaggleChannel <- action
        received := 0
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

func (state *InputPeerState) Eqz (result string, left string, q chan int) (chan bool) {
    done := make(chan bool, 1) //Buffer to avoid hangs
    go func() {
        requestID := atomic.AddInt64(&state.RequestID, 1) 
        status := make(chan *sproto.Response, INITIAL_CHANNEL_SIZE)
        state.SetChannelForRequest(requestID, status)
        action := &sproto.Action{}
        t := sproto.Action_Eqz
        action.Action = &t
        action.Result = &result
        action.RequestCode = &requestID
        action.Share0 = &left
        state.PubNaggleChannel <- action
        received := 0
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

func (state *InputPeerState) OneSub (result string, left string, q chan int) (chan bool) {
    done := make(chan bool, 1) //Buffer to avoid hangs
    go func() {
        requestID := atomic.AddInt64(&state.RequestID, 1) 
        status := make(chan *sproto.Response, INITIAL_CHANNEL_SIZE)
        state.SetChannelForRequest(requestID, status)
        action := &sproto.Action{}
        t := sproto.Action_OneSub
        action.Action = &t
        action.Result = &result
        action.RequestCode = &requestID
        action.Share0 = &left
        state.PubNaggleChannel <- action
        received := 0
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

func (state *InputPeerState) DelValue (result string,  q chan int) (chan bool) {
    done := make(chan bool, 1) //Buffer to avoid hangs
    go func() {
        requestID := atomic.AddInt64(&state.RequestID, 1) 
        status := make(chan *sproto.Response, INITIAL_CHANNEL_SIZE)
        state.SetChannelForRequest(requestID, status)
        action := &sproto.Action{}
        t := sproto.Action_Del
        action.Action = &t
        action.Result = &result
        action.RequestCode = &requestID
        state.PubNaggleChannel <- action
        received := 0
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

func (state *InputPeerState) CmpConst (result string, left string, val int64, q chan int) (chan bool) {
    done := make(chan bool, 1) //Buffer to avoid hangs
    go func() {
        requestID := atomic.AddInt64(&state.RequestID, 1) 
        status := make(chan *sproto.Response, INITIAL_CHANNEL_SIZE)
        state.SetChannelForRequest(requestID, status)
        action := &sproto.Action{}
        t := sproto.Action_CmpConst
        action.Action = &t
        action.Result = &result
        action.RequestCode = &requestID
        action.Share0 = &left
        action.Value = &val
        state.PubNaggleChannel <- action
        received := 0
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

func (state *InputPeerState) NeqConst (result string, left string, val int64, q chan int) (chan bool) {
    done := make(chan bool, 1) //Buffer to avoid hangs
    go func() {
        requestID := atomic.AddInt64(&state.RequestID, 1) 
        status := make(chan *sproto.Response, INITIAL_CHANNEL_SIZE)
        state.SetChannelForRequest(requestID, status)
        action := &sproto.Action{}
        t := sproto.Action_NeqConst
        action.Action = &t
        action.Result = &result
        action.RequestCode = &requestID
        action.Share0 = &left
        action.Value = &val
        state.PubNaggleChannel <- action
        received := 0
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

func (state *InputPeerState) MulConst (result string, left string, val int64, q chan int) (chan bool) {
    done := make(chan bool, 1) //Buffer to avoid hangs
    go func() {
        requestID := atomic.AddInt64(&state.RequestID, 1) 
        status := make(chan *sproto.Response, INITIAL_CHANNEL_SIZE)
        state.SetChannelForRequest(requestID, status)
        action := &sproto.Action{}
        t := sproto.Action_MulConst
        action.Action = &t
        action.Result = &result
        action.RequestCode = &requestID
        action.Share0 = &left
        action.Value = &val
        state.PubNaggleChannel <- action
        received := 0
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
