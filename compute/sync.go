package main
import (
        //"github.com/apanda/smpc/core"
        "fmt"
        )
func (state *ComputePeerState) Sync (q chan int) {
    // Subscribe to HELO messages
    state.SubSock.Subscribe([]byte("HELO"))
    fmt.Println("Starting to wait to receive HELO")
    // Wait to receive one HELO message
    _, err := state.SubSock.Recv()
    if err != nil {
        fmt.Println("Receiving over subscription socket failed", err)
        q <- 1
        return
    }
    fmt.Println("Received HELO")
    // HELO messages are now useless, unsubscribe
    state.SubSock.Unsubscribe([]byte("HELO"))

    // Inform master of our presence on the network
    resp := make([][]byte, 3)
    resp[0] = []byte("")
    resp[1] = []byte("")
    err = state.CoordSock.Send(resp)
    if err != nil {
        fmt.Println("Error sending on coordination socket", err)
        q <- 1
    }
}
