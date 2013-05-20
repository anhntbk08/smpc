package main
import (
    "reflect"
)

func WaitForChannels (chans []chan bool) {
    received := 0
    rcvd := make([]bool, len(chans))
    selChans := make([]reflect.SelectCase, len(chans))
    for i := range chans {
        rcvd[i] = false
        selChans[i] = reflect.SelectCase {
            Dir: reflect.SelectRecv,
            Chan: reflect.ValueOf(chans[i]),
        }
    }
    for received < len(chans) {
        chosen, _, _ := reflect.Select(selChans)
        if !rcvd[chosen] {
            rcvd[chosen] = true
            received ++
        }
    }
}

func LenFor2DArray (arr [][]chan bool) int {
    l := 0
    for i := range arr {
        l += len(arr[i])
    }
    return l
}

func WaitForChannels2D (chans [][]chan bool) {
    received := 0
    chanLen := LenFor2DArray(chans)
    rcvd := make([]bool, chanLen)
    selChans := make([]reflect.SelectCase, chanLen)
    idx := 0
    for i := range chans {
        for j :=  range chans[i] {
            rcvd[idx] = false
            selChans[idx] = reflect.SelectCase {
                Dir: reflect.SelectRecv,
                Chan: reflect.ValueOf(chans[i][j]),
            }
            idx++
        }
    }
    for received < chanLen {
        chosen, _, _ := reflect.Select(selChans)
        if !rcvd[chosen] {
            rcvd[chosen] = true
            received ++
        }
    }
}
type IndexedString struct {
    Index int64
    Value string
}
func WaitForResults (chans map[int64]chan string) (chan IndexedString) {
    r := make(chan IndexedString, 1)
    go func(chans map[int64]chan string, r chan IndexedString) {
        received := 0
        rcvd := make([]bool, len(chans))
        selChans := make([]reflect.SelectCase, len(chans))
        indexList := make(map[int] int64, len(chans))
        idx := 0
        for i := range chans {
            rcvd[idx] = false
            
            selChans[idx] = reflect.SelectCase {
                Dir: reflect.SelectRecv,
                Chan: reflect.ValueOf(chans[i]),
            }
            indexList[idx] = i
            idx++
        }
        for received < len(chans) {
            chosen, recv, recvOk := reflect.Select(selChans)
            if !rcvd[chosen] {
                rcvd[chosen] = true
                received ++
            }
            if !recvOk {
                panic("Prematurely closed channel")
            }
            r <- IndexedString {
                Index: indexList[chosen],
                Value: recv.Interface().(string),
            }
        }
        close(r)
    }(chans, r)
    return r
}
