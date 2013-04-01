package main
import (
        "fmt"
        "sync/atomic"
        )

func (state *InputPeerState) FanInOrForSmirc ( result string, vars []string, q chan int) (chan bool) {
    //tmpVar := Sprintf("__FanInOrForSmirc_%d_tmp", mungingConst)
    mungingConst := atomic.AddInt64(&state.RequestID, 1) 
    lenVar := len(vars)
    for lenVar > 1 {
        start := 0
        start += (lenVar % 2)
        chans := make([]chan bool, lenVar / 2)
        for i := start; i < lenVar; i += 2 {
            tmpVar := fmt.Sprintf("__FanInOrForSmirc_%d_%d_tmp", i, mungingConst)
            chans[i/2] = state.Add(tmpVar, vars[i], vars[i+1], q)
            vars[i/2 + start] = tmpVar
        }
        for ch := range chans {
            <- chans[ch]
        }
        lenVar = (lenVar / 2) + start
    }
    return state.Neqz(result, vars[0], q)
}
