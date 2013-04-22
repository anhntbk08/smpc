package main
import (
        "fmt"
        )
var _ = fmt.Printf // fmt is far too useful
//func (state *InputPeerState) ComputeExportStitch (topo *Topology, node int64, result [][]string, q chan int) {
//    ch := make([][]chan bool, len(topo.IndicesLink[node]))
//    for ind := range topo.AdjacencyMatrix[node] {
//        ch[ind] = make([]chan bool, len(topo.IndicesLink[node]))
//        for j := range topo.IndicesLink[node] {
//            ch[ind][j] = state.CmpConst(result[j][ind], topo.IndicesLink[node][j], int64(ind), q)
//        }
//    }
//    for i := range ch {
//        for j := range ch[i] {
//            <- ch[i][j]
//        }
//    }
//}

// This also accounts for has next hop
func (state *InputPeerState) ComputeExportPolicies (topo *Topology, node int64, result []string, q chan int) {
    // We already know next hops, so do some trivial computations for 
    chans := make([]chan bool, len(result))
    for index := range topo.IndicesNode[node] {
        otherNode := topo.IndicesNode[node][index]
        nhop := topo.NextHop[otherNode]
        otherLink := topo.NodeToPortMap[otherNode][node] // Link on another side
        nhopLink := topo.NodeToPortMap[otherNode][nhop]
        nhopBool := int64(0)
        if nhop != 0 {
            nhopBool = int64(1)
        }
        chans[index] = state.MulConst(result[index], topo.Exports[topo.IndicesNode[node][index]][otherLink][nhopLink], nhopBool, q)
    }
    for index := range chans {
        <- chans[index]
    }

}

func (state *InputPeerState) RunSingleIteration (topo *Topology,  node int64, q chan int) (chan int64) {
    ch2 := make(chan int64, 1) 
    go func() {
        export := state.CreateDumbArray(len(topo.AdjacencyMatrix[node]), "export")
        nhop := state.GetArrayVarName("NextHop", int(node)) 
        state.ComputeExportPolicies (topo, node, export, q)
        ch := state.ArgMax(nhop, topo.IndicesNode[node], export, q)
        <- ch
        ch3 := state.GetValue(nhop, q)

        ch2 <- <- ch3
    }()
    return ch2
}
