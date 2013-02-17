package main
import (
        //"github.com/apanda/smpc/core"
        "fmt"
        "encoding/binary"
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
    fmt.Println("Received HELO, client ", int64(state.Client))
    // HELO messages are now useless, unsubscribe
    state.SubSock.Unsubscribe([]byte("HELO"))

    // Inform master of our presence on the network
    resp := make([][]byte, 1)
    resp[0] = make([]byte, binary.MaxVarintLen32)
    binary.PutVarint(resp[0], int64(state.Client))
    r, _ := binary.Varint(resp[0])
    fmt.Printf("Client = %d\n", r)
    err = state.CoordSock.Send(resp)
    if err != nil {
        fmt.Println("Error sending on coordination socket", err)
        q <- 1
    }
}
