package main
import (
        "fmt"
        "strconv"
        topology "github.com/apanda/smpc/topology"
        )

var _ = fmt.Printf

func MakeTopology (json *topology.JsonTopology, states []*InputPeerState, q chan int) (*Topology) {
    state := states[0]
    nodes := len(json.AdjacencyMatrix)
    topo := &Topology{}
    topo.InitTopology (nodes)
    for node := range json.AdjacencyMatrix {
        nint32, _  := strconv.Atoi(node)
        nint :=  int64(nint32)
        topo.AdjacencyMatrix[nint] = json.AdjacencyMatrix[node]
    }
    for node := range json.PortToNodeMap {
        nint32, _  := strconv.Atoi(node)
        nint :=  int64(nint32)
        for link := range json.PortToNodeMap[node] {
            topo.PortToNodeMap[nint][int64(link)] = json.PortToNodeMap[node][link]
        }
    }

    for node := range json.NodeToPortMap {
        nint32, _  := strconv.Atoi(node)
        nint := int64(nint32)
        for onode := range json.NodeToPortMap[node] {
            onint32, _ := strconv.Atoi(onode)
            onodeint := int64(onint32)
            topo.NodeToPortMap[nint][onodeint] = json.NodeToPortMap[node][onode]
        }
    }

    fmt.Printf("Storing export tables\n")
    exportChans := make([][][]chan bool, len(json.ExportTables))
    for node := range json.ExportTables {
        nint32, _  := strconv.Atoi(node)
        nint := int64(nint32)
        topo.Exports[nint], exportChans[nint32 - 1] = states[nint32 % len(states)].Store2DArrayInSmpc(json.ExportTables[node], fmt.Sprintf("export_%s", node), q)
    }

    for idx := range exportChans {
        for idxi := range exportChans[idx] {
            for idxj := range exportChans[idx][idxi] {
                <- exportChans[idx][idxi][idxj]
            }
        }
    }

    fmt.Printf("Storing stitching consts and preferences\n")
    
    for node := range json.IndicesLink {
        nint32, _  := strconv.Atoi(node)
        nint := int64(nint32)
        var chansLink []chan bool
        var chansNode []chan bool
        var chansStitching [][]chan bool
        topo.IndicesLink[nint], chansLink = states[nint32 % len(states)].StoreArrayInSmpc(json.IndicesLink[node], fmt.Sprintf("inlink_%s", node), q)
        topo.IndicesNode[nint], chansNode = states[nint32 % len(states)].StoreArrayInSmpc(json.IndicesNode[node], fmt.Sprintf("innode_%s", node), q)
        topo.StitchingConsts[nint], chansStitching = states[nint32 % len(states)].Store2DArrayInSmpc(json.StitchingConsts[node], fmt.Sprintf("stitching_%s", node), q)
        for idx := range chansLink {
            <- chansLink[idx]
            <- chansNode[idx]
        }
        for idxi := range chansStitching {
            for idxj := range chansStitching[idxi] {
                <- chansStitching[idxi][idxj]
            }
        }
    }
    fmt.Printf("Done storing stitching consts and preferences\n")
    fmt.Printf("Storing Next hop\n")
    nextHop := state.CreateDumbArray(nodes, "nhop")
    ch := make([]chan bool, nodes)
    for i := int64(1); i < int64(nodes) + 1; i++ {
        ch[i - 1] = states[int(i) % len(states)].SetValue(nextHop[i-1], 0, q)
        topo.NextHop[i] = nextHop[i - 1]
    }
    for c := range ch {
        <- ch[c]
    }
    fmt.Printf("Done storing Next Hop\n")
    return topo
}

type Topology struct {
    // Map from node to other connected nodes
    // Node -> link -> node (links are ordered i.e. 0, 1, 2, 3...)
    AdjacencyMatrix map[int64] []int64
    // Node -> node -> link
    NodeToPortMap map[int64] map[int64] int64
    PortToNodeMap map[int64] map[int64] int64
    // Node -> int -> int -> bool
    // Node -> rank -> index -> bool (says whether for a node, rank x is link y)
    // For node is rank 0 index foo
    StitchingConsts map[int64] [][]string 
    IndicesLink map[int64] []string
    IndicesNode map[int64] []string
    Exports map[int64] [][]string
    NextHop map[int64] string
}

func (topo *Topology) InitTopology (nodes int) {
    topo.AdjacencyMatrix = make(map[int64] []int64, nodes)
    topo.NodeToPortMap = make(map[int64] map[int64] int64, nodes)
    topo.PortToNodeMap = make(map[int64] map[int64] int64, nodes)
    topo.StitchingConsts = make(map[int64] [][]string, nodes)
    topo.IndicesLink = make(map[int64] []string, nodes)
    topo.IndicesNode = make(map[int64] []string, nodes)
    topo.Exports = make(map[int64] [][]string, nodes)
    topo.NextHop = make(map[int64] string, nodes)

    for i := int64(0); i < int64(nodes); i++ {
        topo.PortToNodeMap[i + 1] = make(map[int64] int64, nodes)
        topo.NodeToPortMap[i + 1] = make(map[int64] int64, nodes)
    }
}

/*
func (state *InputPeerState) MakeTestTopology (q chan int) (*Topology) {
    topo := &Topology {}
    topo.InitTopology (4)
    topo.AdjacencyMatrix[1] = []int64 {1, 2, 4}
    topo.AdjacencyMatrix[2] = []int64 {1, 2, 3, 4}
    topo.AdjacencyMatrix[3] = []int64 {2, 3, 4}
    topo.AdjacencyMatrix[4] = []int64 {1, 2, 3, 4}

    topo.NodeToPortMap[1][4] = 1
    topo.NodeToPortMap[1][2] = 2
    topo.NodeToPortMap[1][1] = 0

    topo.NodeToPortMap[2][1] = 3
    topo.NodeToPortMap[2][3] = 1
    topo.NodeToPortMap[2][4] = 2
    topo.NodeToPortMap[2][2] = 0

    topo.NodeToPortMap[3][2] = 1
    topo.NodeToPortMap[3][4] = 2
    topo.NodeToPortMap[3][3] = 0

    topo.NodeToPortMap[4][1] = 3
    topo.NodeToPortMap[4][3] = 1
    topo.NodeToPortMap[4][2] = 2
    topo.NodeToPortMap[4][4] = 0
    
    topo.IndicesLink[1] = state.StoreArrayInSmpc ([]int64 {2, 1, 0}, "indices0", q)
    topo.IndicesNode[1] = state.StoreArrayInSmpc ([]int64 {2, 4, 1}, "indicesNode0", q)
    // Destination
    topo.IndicesLink[2] = state.StoreArrayInSmpc ([]int64 {0, 2, 1, 3}, "indices1", q)
    topo.IndicesNode[2] = state.StoreArrayInSmpc ([]int64 {2, 4, 3, 1}, "indicesNode1", q)

    topo.IndicesLink[3] = state.StoreArrayInSmpc ([]int64 {1, 2, 0}, "indices2", q)
    topo.IndicesNode[3] = state.StoreArrayInSmpc ([]int64 {2, 4, 3}, "indicesNode2", q)

    topo.IndicesLink[4] = state.StoreArrayInSmpc ([]int64 {3, 2, 1, 0}, "indices3", q)
    topo.IndicesNode[4] = state.StoreArrayInSmpc ([]int64 {1, 2, 3, 4}, "indicesNode3", q)


    for i := int64(1); i < 5; i++ {
        topo.StitchingConsts[i] = make([][]string, len(topo.IndicesLink[i]))
        for j := range topo.IndicesLink[i] {
            topo.StitchingConsts[i][j] = make([]string, len(topo.IndicesLink[i]))
            for k := range topo.IndicesLink[i] {
                topo.StitchingConsts[i][j][k] = state.Get3DArrayVarName("stitching", int(i), j, k)
            }
        }
    }

    state.ComputeExportStitch(topo, 1, topo.StitchingConsts[1], q)
    //fmt.Printf ("Export Stitch 1\n")
    //state.PrintMatrix(topo.StitchingConsts[1], q)
    state.ComputeExportStitch(topo, 2, topo.StitchingConsts[2], q)
    //fmt.Printf ("Export Stitch 2\n")
    //state.PrintMatrix(topo.StitchingConsts[2], q)
    state.ComputeExportStitch(topo, 3, topo.StitchingConsts[3], q)
    //fmt.Printf ("Export Stitch 3\n")
    //state.PrintMatrix(topo.StitchingConsts[3], q)
    state.ComputeExportStitch(topo, 4, topo.StitchingConsts[4], q)
    //fmt.Printf ("Export Stitch 4\n")
    //state.PrintMatrix(topo.StitchingConsts[4], q)

    nextHop := state.StoreArrayInSmpc ([]int64 {0, 2, 0, 0}, "nextHop", q)
    for i := range nextHop {
        topo.NextHop[int64(i + 1)] = nextHop[i]
    }
    
    topo.Exports[1] = state.Store2DArrayInSmpc([][]int64 { []int64 {0, 0, 0}, []int64 {0, 0, 1}, []int64 {0, 1, 0}}, "export1", q)
    topo.Exports[2] = state.Store2DArrayInSmpc([][]int64 { []int64 {1, 0, 0, 0}, []int64 {1, 0, 0, 0}, []int64 {1, 0, 0, 0}, []int64 {1, 0,0,0}}, "export2", q)
    topo.Exports[3] = state.Store2DArrayInSmpc([][]int64 { []int64 {0, 0, 0}, []int64 {0, 0, 0}, []int64 {0, 0, 0}}, "export3", q)
    topo.Exports[4] = state.Store2DArrayInSmpc([][]int64 { []int64 {0, 0, 0, 0}, []int64 {0, 0, 1, 1}, []int64 {0, 1, 0, 0}, []int64 {0, 1,0,0}}, "export4", q)
    return topo
}
*/
