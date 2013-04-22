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

func (state *InputPeerState) CreateDumbArray (size int, name string) ([]string) {
    array := make([]string, size)
    for i := 0; i < size; i++ {
        array[i] = state.GetArrayVarName(name, i)
    }
    return array
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
        for j := 0; j < len(vals[i]); j++ {
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

func (state *InputPeerState) Store3DArrayInSmpc (vals [][][]int64, name string, q chan int) ([][][]string) {
    strings := make([][][]string, len(vals))
    chans := make([][][]chan bool, len(vals))
    for i := 0; i < len(vals); i++ {
        strings[i] = make([][]string, len(vals[i]))
        chans[i] = make([][]chan bool, len(vals[i]))
        for j := 0; j < len(vals[i]); j++ {
            strings[i][j] = make([]string, len(vals[i][j]))
            chans[i][j] = make([]chan bool, len(vals[i][j]))
            for k := 0; k < len(vals[i][j]); k++ {
                strings[i][j][k] = state.Get3DArrayVarName(name, i, j, k)
                chans[i][j][k] = state.SetValue(strings[i][j][k], vals[i][j][k], q)
            }
        }
    }
    for i := range chans {
        for j := range chans[i] {
            for k := range chans[i][j] {
                <- chans[i][j][k]
            }
        }
    }
    return strings
}

func (state *InputPeerState) GetArrayVarName (name string, elt int) (string) {
    seq := atomic.AddInt64(&state.RequestID, 1)
    return fmt.Sprintf("%s_%d_%d_%d", name, state.ClusterID, seq, elt)
}

func (state *InputPeerState) Get2DArrayVarName (name string, eltx int, elty int) (string) {
    seq := atomic.AddInt64(&state.RequestID, 1)
    return fmt.Sprintf("%s_%d_%d_%d_%d", name, state.ClusterID, seq, eltx, elty)
}

func (state *InputPeerState) Get3DArrayVarName (name string, eltx int, elty int, eltz int) (string) {
    seq := atomic.AddInt64(&state.RequestID, 1)
    return fmt.Sprintf("%s_%d_%d_%d_%d_%d", name, state.ClusterID, seq, eltx, elty, eltz)
}

func (state *InputPeerState) CascadingAdd (array [][]string, q chan int) {
    limits := make([]int, len(array))
    
    for l := range limits {
        limits[l] = len(array[l])
    }

    ch := make([][]chan bool, len(array))
    for i := range array {
        ch[i] = make([]chan bool, len(array[i]))
    }
    done := false
    for !done {
        done = true
        for index := range array {
            if limits[index] == 1 {
                continue
            }
            if limits[index] == 0 {
                q <- 0
            }
            done = false
            start := limits[index] % 2
            for i := start; i < limits[index]; i+=2 {
                ch[index][(i/2) + start] = state.Add(array[index][i], array[index][i], array[index][i + 1], q)
                array[index][(i/2) + start] = array[index][i]
            }
        }

        if (!done) {
            for index := range ch {
                start := limits[index] % 2
                limits[index] = (limits[index] / 2) + start
                for i := start; i < limits[index]; i++ {
                    <- ch[index][i]
                }
            }
        }
    }
}

