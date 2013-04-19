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
    chans := make([]chan bool, len(states))
    for i := range states {
        chans[i] = make(chan bool)
        go func () {
           topos[i] = json.MakeTopology(states[i], q)
           chans[i] <- true
        }()
    }
    for i := range chans {
        <- chans[i]
    }
    return topos
}
