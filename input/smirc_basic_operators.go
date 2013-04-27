package main
import (
        "fmt"
        "sync/atomic"
        )

func (state *InputPeerState) FanInOrForSmirc ( result string, vars []string, q chan int) (chan bool) {
    done := make(chan bool, INITIAL_CHANNEL_SIZE) //Buffer to avoid hangs
    mungingConst := atomic.AddInt64(&state.RequestID, 1) 
    go func() {
        //tmpVar := Sprintf("__FanInOrForSmirc_%d_tmp", mungingConst)
        lenVar := len(vars)
        iters := 0
        for lenVar > 1 {
            start := 0
            start += (lenVar % 2)
            chans := make([]chan bool, lenVar / 2)
            for i := start; i < lenVar; i += 2 {
                tmpVar := fmt.Sprintf("__FanInOrForSmirc_%d_%d_%d_%d_tmp", state.ClusterID, i, iters, mungingConst)
                chans[i/2] = state.Add(tmpVar, vars[i], vars[i+1], q)
                defer state.DeleteTmpValue(tmpVar, q) 
                vars[i/2 + start] = tmpVar
            }
            for ch := range chans {
                <- chans[ch]
            }
            lenVar = (lenVar / 2) + start
            iters += 1
        }
        <- state.Neqz(result, vars[0], q)
        done <- true
    }()
    return done
}

func (state *InputPeerState) ArgMax (result string, indices []string, values []string, q chan int) (chan bool) {
    done := make(chan bool, INITIAL_CHANNEL_SIZE)
    go func() {

        if len(indices) != len(values) {
            fmt.Printf("Leaving, len(indices) != len(values)")
            done <- false
        }

        // Just for variable making, refactor (change to call one of the Get* functions)
        mungingConst := atomic.AddInt64(&state.RequestID, 1) 


        fannedIn := make([]string, len(values))
        chans := make([]chan bool, len(values))
        
        fannedIn[0] = fmt.Sprintf("__ArgMaxSmirc_0_%d_%d_fior", state.ClusterID, mungingConst)
        defer state.DeleteTmpValue(fannedIn[0], q)
        // For the first thing, we just check to see if the first value is 0
        chans[0] = state.SetValue(fannedIn[0], 0, q)
        
        for i := 1; i < len(values); i++ {
            fannedIn[i] = fmt.Sprintf("__ArgMaxSmirc_%d_%d_%d_fior", i, state.ClusterID, mungingConst)
            defer state.DeleteTmpValue(fannedIn[i], q)
            vars := make([]string, i)
            k := 0
            for j := 0; j < i; j++ {
                vars[k] = values[j]
                k++
            }
            // For everything else check if things are 0
            chans[i] = state.FanInOrForSmirc (fannedIn[i], vars, q)
        }
        neqzChans := make([]chan bool, len(values))
        neqzResults := make([]string, len(values))
        for i := 0; i < len(values); i++ {
            // Again making sure the values for a particular thing are actually 0
            neqzResults[i] = fmt.Sprintf("__ArgMaxSmirc_%d_%d_%d_neqz", i, state.ClusterID,  mungingConst)
            neqzChans[i] = state.Neqz(neqzResults[i], values[i], q)
        }
        for ch := range chans {
            <- chans[ch]
        }
        // fmt.Printf("Looking at fanned in values ")
        // state.PrintArray(fannedIn, q)

        oneSub := make([]chan bool, len(values))
        for i := 0; i < len(values); i++ {
            oneSub[i] = state.OneSub(fannedIn[i], fannedIn[i], q)
        }
        for ch := range oneSub {
            <- oneSub[ch]
        }
        // fmt.Printf("Looking at fanned in values (after 1 sub) ")
        // state.PrintArray(fannedIn, q)

        mulResults := make([]string, len(values))
        mulChans := make([]chan bool, len(values))
        mulResults[0] = result
        mulChans[0] = state.Mul(mulResults[0], indices[0], fannedIn[0], q)
        for i := 1; i < len(values); i++ {
            mulResults[i] = fmt.Sprintf("__ArgMaxSmirc_%d_%d_%d_mul", i, state.ClusterID,  mungingConst)
            defer state.DeleteTmpValue(mulResults[i], q)
            mulChans[i] = state.Mul(mulResults[i], indices[i], fannedIn[i], q)
        }
        for ch := range mulChans {
            <- mulChans[ch]
        }

        // fmt.Printf("Looking at fanned in values * indices")
        // state.PrintArray(mulResults, q)

        for ch := range neqzChans {
            <- neqzChans[ch]
        }

        for i := 0; i < len(values); i++ {
            mulChans[i] = state.Mul(mulResults[i], mulResults[i], neqzResults[i], q)
        }

        for ch := range mulChans {
            <- mulChans[ch]
        }

        // fmt.Printf("Looking at fanned in values * indices * (value != 0)")
        // state.PrintArray(mulResults, q)

        lenMul := len(mulResults)
        for lenMul > 1 {
            start := 0
            if lenMul % 2 != 0 {
                start = 1
            }
            sumChans := make([]chan bool, lenMul/2)
            for i := start; i < lenMul; i+=2 {
                sumChans[i/2] = state.Add(mulResults[i], mulResults[i], mulResults[i+1], q)
                mulResults[i/2 + start] = mulResults[i]
            }

            for ch := range sumChans {
                <- sumChans[ch]
            }

            lenMul = (lenMul / 2) + start
        }
        done <- true
    }()
    return done
}

