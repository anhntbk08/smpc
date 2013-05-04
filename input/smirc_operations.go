package main
import (
        "fmt"
        "time"
        )
var _ = fmt.Printf // fmt is far too useful
func (state *InputPeerState) ComputeExportStitch (topo *Topology, node int64, result [][]string, q chan int) {
    ch := make([][]chan bool, len(topo.IndicesLink[node]))
    fmt.Printf("Computing stitching for node %d\n", node)
    for ind := range topo.AdjacencyMatrix[node] {
        ch[ind] = make([]chan bool, len(topo.IndicesLink[node]))
        for j := range topo.IndicesLink[node] {
            ch[ind][j] = state.CmpConst(result[j][ind], topo.IndicesLink[node][j], int64(ind), q)
        }
    }
    for i := range ch {
        for j := range ch[i] {
            //fmt.Printf("Waiting for %d %d for node %d\n", i, j, node)
            <- ch[i][j]
        }
    }
}

// This also accounts for has next hop
func (state *InputPeerState) ComputeExportPolicies (topo *Topology, node int64, result []string, q chan int) {
    ch := make([][]chan bool, len(topo.IndicesLink[node]))
    tempVar := make([][]string, len(topo.IndicesLink[node]))
    fmt.Printf("For node %d computing what is nexthop (Step 1)\n", node)
    // First start by computing what is the next hop
    for index := range topo.AdjacencyMatrix[node] {
        otherNode := topo.AdjacencyMatrix[node][index]
        nhop := topo.NextHop[otherNode]
        ch[index] = make([]chan bool, len(topo.AdjacencyMatrix[otherNode]))
        tempVar[index] = make([]string, len(topo.AdjacencyMatrix[otherNode]))
        for possibleLinks := range topo.AdjacencyMatrix[otherNode] {
            tempVar[index][possibleLinks] = state.Get2DArrayVarName("peerExport", index, possibleLinks)
            defer state.DeleteTmpValue(tempVar[index][possibleLinks], q)
            ch[index][possibleLinks] = state.CmpConst(tempVar[index][possibleLinks], nhop, topo.AdjacencyMatrix[otherNode][int64(possibleLinks)], q)
        }
    }

    for i := range ch {
        for j := range ch[i] {
            <- ch[i][j]
        }
    }

    //fmt.Printf("What is next hop\n")
    //state.PrintMatrix(tempVar, q)
    //fmt.Printf("\n")

    fmt.Printf("For node %d computing export policies (Step 2)\n", node)
    // Compute the export policies based on next hop
    for index := range topo.AdjacencyMatrix[node] {
        otherNode := topo.AdjacencyMatrix[node][index]
        otherLink := topo.NodeToPortMap[otherNode][node] // Link on another side
        for possibleLinks := range topo.AdjacencyMatrix[otherNode] {
            pLink := topo.NodeToPortMap[otherNode][topo.AdjacencyMatrix[otherNode][possibleLinks]]
            ch[index][possibleLinks] = state.Mul(tempVar[index][possibleLinks], tempVar[index][possibleLinks], topo.Exports[otherNode][otherLink][pLink], q)
        }
    }

    for i := range ch {
        for j := range ch[i] {
            <- ch[i][j]
        }
    }

    // Extract a single export vector
    fmt.Printf("For node %d combining export policies (Step 3)\n", node)
    state.CascadingAdd(tempVar, q)
    //fmt.Printf("After cascading add\n")
    //for i := range tempVar {
    //    fmt.Printf("%d: %s\n", i, tempVar[i][0])
    //}
    //fmt.Printf("\n")
    //fmt.Printf("Export vector\n")
    //state.PrintMatrix(tempVar, q)
    //fmt.Printf("\n")

    tempVar3 := make([]string, len(tempVar))
    for i := range topo.AdjacencyMatrix[node] {
        onode := topo.AdjacencyMatrix[node][i]
        link := topo.NodeToPortMap[node][onode]
        tempVar3[link] = tempVar[i][0]
    //    fmt.Printf("(Node: %d) tempVar %d -> tempVar3 %d\n", node, i, link)
    }
    //fmt.Printf("Translated to tempVar3\n")
    //for i := range tempVar {
    //    fmt.Printf("%d: %s\n", i, tempVar3[i])
    //}
    //fmt.Printf("\n")
    

    fmt.Printf("For node %d rearranging based on import policy (Step 4)\n", node)
    // Rearrange based on ordering
    tempVar2 := make([][]string, len(topo.IndicesLink[node]))
    ch2 := make([][]chan bool, len(topo.IndicesLink[node]))
    for onodeIndex := range tempVar2 {
        tempVar2[onodeIndex] = make([]string, len(topo.IndicesLink[node]))
        //fmt.Printf("onodeIndex %d %d, indices %d\n",onodeIndex, len(result), len(topo.IndicesLink[node]))
        ch2[onodeIndex] = make([]chan bool, len(topo.IndicesLink[node]))
        tempVar2[onodeIndex][0] = result[onodeIndex]
        ch2[onodeIndex][0] = state.Mul(tempVar2[onodeIndex][0], tempVar3[0], topo.StitchingConsts[node][onodeIndex][0], q)
        for index := 1; index < len(ch2[onodeIndex]); index++ {
            tempVar2[onodeIndex][index] = state.Get2DArrayVarName("peerExport2", onodeIndex, index)
            defer state.DeleteTmpValue(tempVar2[onodeIndex][index], q)
            ch2[onodeIndex][index] = state.Mul(tempVar2[onodeIndex][index], tempVar3[index], topo.StitchingConsts[node][onodeIndex][index], q)
        }
    }

    for i := range ch2 {
        for j := range ch2[i] {
            <- ch2[i][j]
        }
    }
    //fmt.Printf("Rearranged\n")
    //state.PrintMatrix(tempVar2, q)
    //fmt.Printf("\n")
    fmt.Printf("For node %d combining for answer (Step 5)\n", node)
    state.CascadingAdd(tempVar2, q)
    //fmt.Printf("Final\n")
    //state.PrintMatrix(tempVar2, q)
    //fmt.Printf("\n")
}

func (state *InputPeerState) RunSingleIteration (topo *Topology,  node int64, q chan int) (chan string) {
    ch2 := make(chan string, INITIAL_CHANNEL_SIZE) 
    go func() {
        export := state.CreateDumbArray(len(topo.AdjacencyMatrix[node]), "export")
        nhop := state.GetArrayVarName("NextHop", int(node)) 
        //func (state *InputPeerState) ComputeExportPolicies (topo *Topology, node int64, result []string, q chan int) {
        now := time.Now()
        fmt.Printf("Starting round for %d (%v)\n", node ,now.String())
        fmt.Printf("Begining computing export policies %d\n", node)
        state.ComputeExportPolicies (topo, node, export, q)
        fmt.Printf("Finished computing export policies %d\n", node)
        //fmt.Printf("Done computing export policies %d (%v)\n", node, time.Since(now).String())
        //state.PrintArray(export, q)
        // fmt.Printf("Indices for node %d: ", node)
        //state.PrintArray(topo.IndicesNode[node], q)
        //func (state *InputPeerState) ArgMax (result string, indices []string, values []string, q chan int) (chan bool) {
        //fmt.Printf("Starting ArgMax\n")
        //fmt.Printf("Start computing argmax %d (%v)\n", node ,now.String())
        fmt.Printf("Begining computing arg max %d\n", node)
        ch := state.ArgMax(nhop, topo.IndicesNode[node], export, q)
        <- ch
        fmt.Printf("Done round for %d (%v)\n", node, time.Since(now).String())
        ch2 <- nhop
        fmt.Printf("Notified channel for %d (%v)\n", node, ch2)
    }()
    return ch2
}
