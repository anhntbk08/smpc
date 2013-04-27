package topology
import (
        "io/ioutil"
        "encoding/json"
        "fmt"
        )

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
    }
    var jsonTopo JsonTopology
    // Parse configuration, produce an object. We assume configuration is in JSON
    err = json.Unmarshal(contents, &jsonTopo)
    if err != nil {
        panic (fmt.Sprintf("Error reading json topology file: %v", err))
    }
    return &jsonTopo
}

