/* Implementation for the HELO protocol, used to synchronize the Input peer and
computational peers */
package main
import (
        //"github.com/apanda/smpc/core"
        "fmt"
        "time"
        )
func (state *InputPeerState) SendHello (exitChannel chan int, notificationChannel chan bool) {
    sleepChan := make(chan bool, 1)
    for true {
        msg := make([][]byte, 1)
        msg[0] = []byte("HELO")
        state.PubChannel.Out() <- msg
        go func() {
            time.Sleep(10 * time.Millisecond)
            sleepChan <- true
        }()
        select {
            case <- notificationChannel:
                return
            case <- sleepChan:
            // Do nothing
        }
    }
}

/* Synchronize compute clients with input peer */
func (state *InputPeerState) Sync (q chan int) {
    connectedSoFar := 0
    fmt.Printf("Waiting for %d connections\n", state.Config.Clients)
    // Make sure the notification channel cannot buffer messages, this is some what like a process.join
    notification := make(chan bool, 0)
    // Start a go routine to send HELO
    go state.SendHello(q, notification) 
    
    for connectedSoFar < state.Config.Clients {
        msg := <- state.CoordChannel.In()
        /* if err != nil {
            fmt.Println("Error receiving on coordination socket", err)
            q <- 1
            return
        }*/
        // Save client identity
        state.ComputeSlaves[connectedSoFar] = msg[0]
        connectedSoFar += 1
        fmt.Printf("Waiting for %d connections\n", state.Config.Clients - connectedSoFar)
    }
    // Tell the HELO routine to stop
    notification <- true
}
