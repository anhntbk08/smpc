package main
import (
        "fmt"
        "time"
        )

func circuit (states []*InputPeerState, topoFile *string, dest int64, end_channel chan int) {
    state := states[0]
    val := int64(0)
    _ = val
    jsonTopo := ParseJsonTopology(topoFile)  
    topo := jsonTopo.MakeTopology(state, end_channel)
    if dest != 0 {
        topo.NextHop[dest] = dest
    }
    //topo := state.MakeTestTopology(end_channel)  
    
    nnhop := make(map[int64] int64, len(topo.AdjacencyMatrix))
    elapsed := float64(0)
    iters := 12
    for i := 0; i < iters; i++ {
        t := time.Now()
        nnhop = make(map[int64] int64, len(topo.AdjacencyMatrix))
        ch := make(map[int64] chan int64, len(topo.AdjacencyMatrix))
        for i  := range topo.AdjacencyMatrix {
            ch[i] = state.RunSingleIteration(topo, i, end_channel)
            nnhop[i] = <- ch[i]
        }
        //for i := range topo.AdjacencyMatrix {
        //}
        topo.NextHop = nnhop
        elapsed += (time.Since(t).Seconds())
    }

    fmt.Printf("Two round NextHop, should be 2, 2, 2, 1 Time: %f\n", elapsed/float64(iters))
    for ind := range nnhop {
        //c42 := state.GetValue(nnhop[ind], end_channel)
        //val = <- c42
        fmt.Printf("%d: %d\n", ind, nnhop[ind])
    }
    
    end_channel <- 0
}
