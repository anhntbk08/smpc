package main
import (
        "fmt"
        "io/ioutil"
        "encoding/json"
        "strconv"
        )

var _ = fmt.Printf

type JsonTopology struct {
    AdjacencyMatrix map[string] []int64
    PortToNodeMap map[string] []int64
    NodeToPortMap map[string] map[string] int64
    ExportTables map[string] [][]int64 
    IndicesLink map[string] []int64
    IndicesNode map[string] []int64
}

func ParseJsonTopology (config *string) (*JsonTopology) {
    contents, err := ioutil.ReadFile(*config)
    if err != nil {
        panic (fmt.Sprintf("Could not read topology file, error = %s", err))
        return nil
    }
    var jsonTopo JsonTopology
    // Parse configuration, produce an object. We assume configuration is in JSON
    err = json.Unmarshal(contents, &jsonTopo)
    if err != nil {
        panic (fmt.Sprintf("Error reading json topology file: %v", err))
        return nil
    }
    return &jsonTopo
}

func (json *JsonTopology) MakeTopology (state *InputPeerState, q chan int) (*Topology) {
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

    for node := range json.ExportTables {
        nint32, _  := strconv.Atoi(node)
        nint := int64(nint32)
        topo.Exports[nint] = state.Store2DArrayInSmpc(json.ExportTables[node], fmt.Sprintf("export_%s", node), q)
    }
    
    for node := range json.IndicesLink {
        nint32, _  := strconv.Atoi(node)
        nint := int64(nint32)
        topo.IndicesLink[nint] = json.IndicesLink[node]
        topo.IndicesNode[nint] = json.IndicesNode[node]
    }

    for i := int64(1); i < int64(nodes) + 1; i++ {
        topo.StitchingConsts[i] = make([][]string, len(topo.IndicesLink[i]))
        for j := range topo.IndicesLink[i] {
            topo.StitchingConsts[i][j] = make([]string, len(topo.IndicesLink[i]))
            for k := range topo.IndicesLink[i] {
                topo.StitchingConsts[i][j][k] = state.Get3DArrayVarName("stitching", int(i), j, k)
            }
        }
        topo.NextHop[i] = int64(0)
    }
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
    IndicesLink map[int64] []int64
    IndicesNode map[int64] []int64
    Exports map[int64] [][]string
    NextHop map[int64] int64
}

func (topo *Topology) InitTopology (nodes int) {
    topo.AdjacencyMatrix = make(map[int64] []int64, nodes)
    topo.NodeToPortMap = make(map[int64] map[int64] int64, nodes)
    topo.PortToNodeMap = make(map[int64] map[int64] int64, nodes)
    topo.StitchingConsts = make(map[int64] [][]string, nodes)
    topo.IndicesLink = make(map[int64] []int64, nodes)
    topo.IndicesNode = make(map[int64] []int64, nodes)
    topo.Exports = make(map[int64] [][]string, nodes)
    topo.NextHop = make(map[int64] int64, nodes)

    for i := int64(0); i < int64(nodes); i++ {
        topo.NodeToPortMap[i + 1] = make(map[int64] int64, nodes)
        topo.PortToNodeMap[i + 1] = make(map[int64] int64, nodes)
    }
}

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
    topo.PortToNodeMap[1][1] = 4
    topo.PortToNodeMap[1][2] = 2
    topo.PortToNodeMap[1][0] = 1

    topo.NodeToPortMap[2][1] = 3
    topo.NodeToPortMap[2][3] = 1
    topo.NodeToPortMap[2][4] = 2
    topo.NodeToPortMap[2][2] = 0
    topo.PortToNodeMap[2][3] = 1
    topo.PortToNodeMap[2][1] = 3
    topo.PortToNodeMap[2][2] = 4
    topo.PortToNodeMap[2][0] = 2

    topo.NodeToPortMap[3][2] = 1
    topo.NodeToPortMap[3][4] = 2
    topo.NodeToPortMap[3][3] = 0
    topo.PortToNodeMap[3][1] = 2
    topo.PortToNodeMap[3][2] = 4
    topo.PortToNodeMap[3][3] = 3

    topo.NodeToPortMap[4][1] = 3
    topo.NodeToPortMap[4][3] = 1
    topo.NodeToPortMap[4][2] = 2
    topo.NodeToPortMap[4][4] = 0
    topo.PortToNodeMap[4][3] = 1
    topo.PortToNodeMap[4][1] = 3
    topo.PortToNodeMap[4][2] = 2
    topo.PortToNodeMap[4][0] = 4
    
    topo.IndicesLink[1] = []int64 {2, 1, 0}
    topo.IndicesNode[1] = []int64 {2, 4, 1}
    // Destination
    topo.IndicesLink[2] = []int64 {0, 2, 1, 3}
    topo.IndicesNode[2] = []int64 {2, 4, 3, 1}

    topo.IndicesLink[3] = []int64 {1, 2, 0}
    topo.IndicesNode[3] = []int64 {2, 4, 3}

    topo.IndicesLink[4] = []int64 {3, 2, 1, 0}
    topo.IndicesNode[4] = []int64 {1, 2, 3, 4}


    for i := int64(1); i < 5; i++ {
        topo.StitchingConsts[i] = make([][]string, len(topo.IndicesLink[i]))
        for j := range topo.IndicesLink[i] {
            topo.StitchingConsts[i][j] = make([]string, len(topo.IndicesLink[i]))
            for k := range topo.IndicesLink[i] {
                topo.StitchingConsts[i][j][k] = state.Get3DArrayVarName("stitching", int(i), j, k)
            }
        }
    }

    hasNext := []int64 {0, 1, 0, 0}
    nextHop := []int64 {0, 2, 0, 0}
    for i := range hasNext {
        topo.NextHop[int64(i + 1)] = nextHop[i]
    }
    topo.Exports[1] = state.Store2DArrayInSmpc([][]int64 { []int64 {0, 0, 0}, []int64 {0, 0, 1}, []int64 {0, 1, 0}}, "export1", q)
    topo.Exports[2] = state.Store2DArrayInSmpc([][]int64 { []int64 {1, 0, 0, 0}, []int64 {1, 0, 0, 0}, []int64 {1, 0, 0, 0}, []int64 {1, 0,0,0}}, "export2", q)
    topo.Exports[3] = state.Store2DArrayInSmpc([][]int64 { []int64 {0, 0, 0}, []int64 {0, 0, 0}, []int64 {0, 0, 0}}, "export3", q)
    topo.Exports[4] = state.Store2DArrayInSmpc([][]int64 { []int64 {0, 0, 0, 0}, []int64 {0, 0, 1, 1}, []int64 {0, 1, 0, 0}, []int64 {0, 1,0,0}}, "export4", q)
    return topo
}

