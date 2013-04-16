package main
import (
        "fmt"
        )

func (state *InputPeerState) PrintMatrix (matrix [][]string, q chan int) {
    for i := range matrix {
        for j := range matrix[i] {
            ch := state.GetValue(matrix[i][j], q)
            val := <-ch
            fmt.Printf("%d ", val)
        }
        fmt.Printf("\n")
    }
}

func (state *InputPeerState) PrintArray (matrix []string, q chan int) {
    for i := range matrix {
        ch := state.GetValue(matrix[i], q)
        val := <-ch
        fmt.Printf("%d ", val)
    }
    fmt.Printf("\n")
}
