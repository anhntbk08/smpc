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
        ch <- true
    }()
    return ch
}
