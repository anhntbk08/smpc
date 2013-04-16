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
    ch := make([][]chan bool, len(topo.IndicesLink[node]))
    tempVar := make([][]string, len(topo.IndicesLink[node]))
    // First start by computing what is the next hop
    for index := range topo.IndicesNode[node] {
        otherNode := topo.IndicesNode[node][index]
        nhop := topo.NextHop[otherNode]
        ch[index] = make([]chan bool, len(topo.IndicesNode[otherNode]))
        tempVar[index] = make([]string, len(topo.IndicesNode[otherNode]))
        tempVar[index][0] = result[index]
        ch[index][0] = state.CmpConst(tempVar[index][0], nhop, topo.IndicesNode[otherNode][0], q)
        for possibleLinks := 1; possibleLinks < len(topo.IndicesNode[otherNode]); possibleLinks++ {
            tempVar[index][possibleLinks] = state.Get2DArrayVarName("peerExport", index, possibleLinks)
            defer state.DeleteTmpValue(tempVar[index][possibleLinks], q)
            ch[index][possibleLinks] = state.CmpConst(tempVar[index][possibleLinks], nhop, topo.IndicesNode[otherNode][possibleLinks], q)
        }
    }

    for i := range ch {
        for j := range ch[i] {
            <- ch[i][j]
        }
    }

    //state.PrintMatrix(tempVar, q)

    // Compute the export policies based on next hop
    for index := range topo.IndicesNode[node] {
        otherNode := topo.IndicesNode[node][index]
        otherLink := topo.NodeToPortMap[otherNode][node] // Link on another side
        for possibleLinks := range topo.IndicesNode[otherNode] {
            pLink := topo.NodeToPortMap[otherNode][topo.IndicesNode[otherNode][possibleLinks]]
            ch[index][possibleLinks] = state.Mul(tempVar[index][possibleLinks], tempVar[index][possibleLinks], topo.Exports[otherNode][otherLink][pLink], q)
        }
    }

    for i := range ch {
        for j := range ch[i] {
            <- ch[i][j]
        }
    }
    //state.PrintMatrix(tempVar, q)

    // Extract a single export vector
    state.CascadingAdd(tempVar, q)
    //state.PrintMatrix(tempVar, q)

    /*tempVar3 := make([][]string, len(tempVar))
    for i := range topo.AdjacencyMatrix[node] {
        onode := topo.AdjacencyMatrix[node][i]
        link := topo.NodeToPortMap[node][onode]
        tempVar3[link] = tempVar[i]
    }
    tempVar = tempVar3*/
}

func (state *InputPeerState) RunSingleIteration (topo *Topology,  node int64, q chan int) (chan string) {
    ch2 := make(chan string, 1) 
    go func() {
        export := state.CreateDumbArray(len(topo.AdjacencyMatrix[node]), "export")
        nhop := state.GetArrayVarName("NextHop", int(node)) 
        //func (state *InputPeerState) ComputeExportPolicies (topo *Topology, node int64, result []string, q chan int) {
        state.ComputeExportPolicies (topo, node, export, q)
        //state.PrintArray(export, q)
        // fmt.Printf("Indices for node %d: ", node)
        //state.PrintArray(topo.IndicesNode[node], q)
        //func (state *InputPeerState) ArgMax (result string, indices []string, values []string, q chan int) (chan bool) {
        //fmt.Printf("Starting ArgMax\n")
        ch := state.ArgMax(nhop, topo.IndicesNode[node], export, q)
        <- ch
        ch2 <- nhop
    }()
    return ch2
}
