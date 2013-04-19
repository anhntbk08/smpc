package main
import (
        "fmt"
        )
var _ = fmt.Println
func BroadcastSetValue (states []*InputPeerState, result string, value int64, q chan int) (chan bool) {
    ch := make(chan bool)
    go func() {
        chans := make([]chan bool, len(states))
        for state := range states {
            chans[state] = states[state].SetValue(result, value, q)
        }
        for state := range chans {
            <- chans[state]
        }
        close(ch)
    }()
    return ch
}

func (json *JsonTopology) MakeBroadcastTopology (states []*InputPeerState, q chan int) ([]*Topology) {
    topos := make([]*Topology, len(states))
    fmt.Printf("Constructing topology\n")
    for i := range topos {
        topos[i] = json.MakeTopology(states[i], q)
        fmt.Printf("Done constructing topology %d\n", i)
    }
    return topos
}
