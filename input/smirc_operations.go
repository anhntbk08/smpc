package main
import (
        "fmt"
        )
var _ = fmt.Printf // fmt is far too useful
func (state *InputPeerState) ComputeExportStitch (topo *Topology, node int64, result [][]string, q chan int) {
    ch := make([][]chan bool, len(topo.IndicesLink[node]))
    for ind := range topo.AdjacencyMatrix[node] {
        ch[ind] = make([]chan bool, len(topo.IndicesLink[node]))
        for j := range topo.IndicesLink[node] {
            ch[ind][j] = state.CmpConst(result[j][ind], topo.IndicesLink[node][j], int64(ind), q)
        }
    }
    for i := range ch {
        for j := range ch[i] {
            <- ch[i][j]
        }
    }
}

// This also accounts for has next hop
func (state *InputPeerState) ComputeExportPolicies (topo *Topology, node int64, result []string, q chan int) {
    // We already know next hops, so do some trivial computations for 
    tempVar := make([]string, len(topo.IndicesLink[node]))
    for index := range topo.AdjacencyMatrix[node] {
        otherNode := topo.AdjacencyMatrix[node][index]
        nhop := topo.NextHop[otherNode]
        otherLink := topo.NodeToPortMap[otherNode][node] // Link on another side
        nhopLink := topo.NodeToPortMap[otherNode][nhop]
        tempVar[index] = topo.Exports[topo.AdjacencyMatrix[node][index]][otherLink][nhopLink]
    }

    tempVar3 := make([]string, len(tempVar))
    for i := range topo.AdjacencyMatrix[node] {
        onode := topo.AdjacencyMatrix[node][i]
        link := topo.NodeToPortMap[node][onode]
        tempVar3[link] = tempVar[i]
    }
    tempVar = tempVar3

    // Rearrange based on ordering
    tempVar2 := make([][]string, len(topo.IndicesLink[node]))
    ch2 := make([][]chan bool, len(topo.IndicesLink[node]))
    for onodeIndex := range tempVar2 {
        tempVar2[onodeIndex] = make([]string, len(topo.IndicesLink[node]))
        //fmt.Printf("onodeIndex %d %d, indices %d\n",onodeIndex, len(result), len(topo.IndicesLink[node]))
        ch2[onodeIndex] = make([]chan bool, len(topo.IndicesLink[node]))
        tempVar2[onodeIndex][0] = result[onodeIndex]
        ch2[onodeIndex][0] = state.Mul(tempVar2[onodeIndex][0], tempVar[0], topo.StitchingConsts[node][onodeIndex][0], q)
        for index := 1; index < len(ch2[onodeIndex]); index++ {
            tempVar2[onodeIndex][index] = state.Get2DArrayVarName("peerExport2", onodeIndex, index)
            defer state.DeleteTmpValue(tempVar2[onodeIndex][index], q)
            ch2[onodeIndex][index] = state.Mul(tempVar2[onodeIndex][index], tempVar[index], topo.StitchingConsts[node][onodeIndex][index], q)
        }
    }

    for i := range ch2 {
        for j := range ch2[i] {
            <- ch2[i][j]
        }
    }
    state.CascadingAdd(tempVar2, q)
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
