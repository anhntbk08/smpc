package main
import (
        "fmt"
        "time"
        )
func circuit (states []*InputPeerState, topoFile *string, dest int64, end_channel chan int) {
    state := states[0]
    val := int64(0)
    _ = val
    fmt.Printf("Starting circuit\n")
    jsonTopo := ParseJsonTopology(topoFile)  
    topos := jsonTopo.MakeBroadcastTopology(states, end_channel)
    if dest != 0 {
        ch := make([]chan bool, len(states))
        for i := range states {
            ch[i] = states[i].SetValue(topos[i].NextHop[dest], dest, end_channel)
        }
        for i := range ch {
            <- ch[i]
        }
    }
    topo := topos[0]
    //topo := state.MakeTestTopology(end_channel)  
    
    nnhop := make(map[int64] string, len(topo.AdjacencyMatrix))
    elapsed := float64(0)
    iters :=12
    for it := 0; it < iters; it++ {
        t := time.Now()
        nnhop = make(map[int64] string, len(topo.AdjacencyMatrix))
        ch := make(map[int64] chan string, len(topo.AdjacencyMatrix))
        for i := range topo.AdjacencyMatrix {
            ch[i] = state.RunSingleIteration(topo, i, end_channel)
        }
        for i  := range topo.AdjacencyMatrix {
            nnhop[i] = <- ch[i]
        }
        topo.NextHop = nnhop
        elapsed += (time.Since(t).Seconds())
    }

    fmt.Printf("Two round NextHop, should be 2, 2, 2, 1 Time: %f\n", elapsed/float64(iters))
    for ind := range nnhop {
        c42 := state.GetValue(nnhop[ind], end_channel)
        val = <- c42
        fmt.Printf("%d: %d\n", ind, val)
    }
    
    end_channel <- 0
}