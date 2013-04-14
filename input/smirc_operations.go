package main
import (
        "fmt"
        "sync/atomic"
        )
func (state *InputPeerState) DeleteTmpValue (variable string, q chan int) {
    ch := state.DelValue(variable, q)
    go func() {
        <- ch
    }()
}

func (state *InputPeerState) FanInOrForSmirc ( result string, vars []string, q chan int) (chan bool) {
    done := make(chan bool, 1) //Buffer to avoid hangs
    go func() {
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
                defer state.DeleteTmpValue(tmpVar, q) 
                vars[i/2 + start] = tmpVar
            }
            for ch := range chans {
                <- chans[ch]
            }
            lenVar = (lenVar / 2) + start
        }
        <- state.Neqz(result, vars[0], q)
        close(done)
    }()
    return done
}

func (state *InputPeerState) ArgMax (result string, indices []string, values []string, q chan int) (chan bool) {
    done := make(chan bool, 1)
    go func() {
        if len(indices) != len(values) {
            done <- false
        }
        mungingConst := atomic.AddInt64(&state.RequestID, 1) 
        fannedIn := make([]string, len(values))
        chans := make([]chan bool, len(values))
        fannedIn[0] = fmt.Sprintf("__ArgMaxSmirc_0_%d_fior", mungingConst)
        defer state.DeleteTmpValue(fannedIn[0], q)
        chans[0] = state.Neqz(fannedIn[0], values[0], q)
        for i := 1; i < len(values); i++ {
            fannedIn[i] = fmt.Sprintf("__ArgMaxSmirc_%d_%d_fior", i, mungingConst)
            defer state.DeleteTmpValue(fannedIn[i], q)
            vars := make([]string, i)
            k := 0
            for j := 0; j < i; j++ {
                vars[k] = values[j]
                k++
            }
            chans[i] = state.FanInOrForSmirc (fannedIn[i], vars, q)
        }
        neqzChans := make([]chan bool, len(values))
        neqzResults := make([]string, len(values))
        for i := 0; i < len(values); i++ {
            neqzResults[i] = fmt.Sprintf("__ArgMaxSmirc_%d_%d_neqz", i, mungingConst)
            neqzChans[i] = state.Neqz(neqzResults[i], values[i], q)
        }
        for ch := range chans {
            <- chans[ch]
        }
        oneSub := make([]chan bool, len(values))
        for i := 0; i < len(values); i++ {
            oneSub[i] = state.OneSub(fannedIn[i], fannedIn[i], q)
        }
        for ch := range oneSub {
            <- oneSub[ch]
        }
        mulResults := make([]string, len(values))
        mulChans := make([]chan bool, len(values))
        mulResults[0] = result
        mulChans[0] = state.Mul(mulResults[0], indices[0], fannedIn[0], q)
        for i := 1; i < len(values); i++ {
            mulResults[i] = fmt.Sprintf("__ArgMaxSmirc_%d_%d_mul", i, mungingConst)
            defer state.DeleteTmpValue(mulResults[i], q)
            mulChans[i] = state.Mul(mulResults[i], indices[i], fannedIn[i], q)
        }
        for ch := range mulChans {
            <- mulChans[ch]
        }

        for ch := range neqzChans {
            <- neqzChans[ch]
        }

        for i := 0; i < len(values); i++ {
            mulChans[i] = state.Mul(mulResults[i], mulResults[i], neqzResults[i], q)
        }

        for ch := range mulChans {
            <- mulChans[ch]
        }

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
        fmt.Printf("Done ArgMax result is at %s\n", mulResults[0])
        close(done)
    }()
    return done
}

func (state *InputPeerState) StoreArrayInSmpc (vals []int64, name string, q chan int) ([]string) {
    array := make([]string, len(vals))
    chans := make([]chan bool, len(vals))
    for i := 0; i < len(vals); i++ {
        array[i] = state.GetArrayVarName(name, i)
        chans[i] = state.SetValue(array[i], vals[i], q)
    }
    for ch := range chans {
        <- chans[ch]
    }
    return array
}

func (state *InputPeerState) Store2DArrayInSmpc (vals [][]int64, name string, q chan int) ([][]string) {
    strings := make([][]string, len(vals))
    chans := make([][]chan bool, len(vals))
    for i := 0; i < len(vals); i++ {
        strings[i] = make([]string, len(vals[i]))
        chans[i] = make([]chan bool, len(vals[i]))
        for j := 0; j < len(vals); j++ {
            strings[i][j] = state.Get2DArrayVarName(name, i, j)
            chans[i][j] = state.SetValue(strings[i][j], vals[i][j], q)
        }
    }
    for i := range chans {
        for j := range chans[i] {
            <- chans[i][j]
        }
    }
    return strings
}

func (state *InputPeerState) GetArrayVarName (name string, elt int) (string) {
    seq := atomic.AddInt64(&state.RequestID, 1)
    return fmt.Sprintf("%s_%d_[%d]", name, seq, elt)
}

func (state *InputPeerState) Get2DArrayVarName (name string, eltx int, elty int) (string) {
    seq := atomic.AddInt64(&state.RequestID, 1)
    return fmt.Sprintf("%s_%d_[%d,%d]", name, seq, eltx, elty)
}
