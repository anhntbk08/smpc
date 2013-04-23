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
    topo := jsonTopo.MakeTopology(state, end_channel)
    if dest != 0 {
        ch := state.SetValue(topo.NextHop[dest], dest, end_channel)
        <- ch
    }
    //topo := state.MakeTestTopology(end_channel)  
    
    nnhop := make(map[int64] string, len(topo.AdjacencyMatrix))
    elapsed := float64(0)
    iters :=30
    for it := 0; it < iters; it++ {
        t := time.Now()
        nnhop = make(map[int64] string, len(topo.AdjacencyMatrix))
        ch := make(map[int64] chan string, len(topo.AdjacencyMatrix))
        for i := range topo.AdjacencyMatrix {
            ch[i] = states[int(i) % len(states)].RunSingleIteration(topo, i, end_channel)
        }
        fakech := make(map[int64] chan bool, len(topo.AdjacencyMatrix))
        for i  := range topo.AdjacencyMatrix {
            nnhop[i] = <- ch[i]
        }
        for i := range fakech {
            <- fakech[i]
        }
        topo.NextHop = nnhop
        elapsed += (time.Since(t).Seconds())
    }

    fmt.Printf("Two round NextHop, should be 2, 2, 2, 1 Time: %f\n", elapsed/float64(iters))
    for ind := range nnhop {
        c42 := state.GetValue(nnhop[ind], end_channel)
        val = <- c42
        fmt.Printf("%s %d (%d): %d\n",nnhop[ind], ind, (int(ind) % len(states)), val)
    }
    
    end_channel <- 0
}
