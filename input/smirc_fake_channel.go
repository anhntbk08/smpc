package main
import (
        "fmt"
        "sync/atomic"
        )
var _ = fmt.Println
// This is a hacky channel to get multiple parallel implmentations to communicate shares. In reality this would be done
// either using a real database thing, or something else

func DiffuseVar(states []*InputPeerState, where string, state int, q chan int) (chan bool) {
    done := make(chan bool, 1)
    go func() {
        getRequestID := atomic.AddInt64(&states[state].RequestID, 1)
        vals, _, _ := states[state].GetRawValue (where, getRequestID, q) 
        chans := make(map[int] chan bool, len(states))
        for i := 0; i < len(states); i++ {
            if i != state {
                chans[i] = make(chan bool)
                go func (i int) {
                    requestID := atomic.AddInt64(&states[i].RequestID, 1)
                    states[i].SetRawValue(where, vals,  requestID, q)
                    chans[i] <- true
                }(i)
            }
        }
        for i := range chans {
            <- chans[i]
        }
        done <- true
    }()
    return done
}
