package main
import (
        //"github.com/apanda/smpc/core"
        sproto "github.com/apanda/smpc/proto"
        "code.google.com/p/goprotobuf/proto" 
        "fmt"
        )
/*
We assume the message structure
   envelope <- address or empty
   data <- the actual data
*/
func MsgToAction (msg [][]byte) (*sproto.Action) {
    fmt.Println("Unmarshaling action", len(msg))
    action := &sproto.Action{}
    // msg[0] is the reply envelope, hence use msg[1]
    err := proto.Unmarshal(msg[1], action)
    fmt.Println("Unmarshaled")
    if err != nil {
        fmt.Println("Error unmarshaling", err)
        return nil
    }
    return action
}

func ResponseToMsg (resp *sproto.Response) ([][]byte) {
    fmt.Println("Marshalling response")
    msg := make([][]byte, 2)
    msg[0] = []byte("")
    var err error
    msg[1], err = proto.Marshal(resp)
    fmt.Println("Done  Marshalling response")
    if err != nil {
        fmt.Println("Error marshalling", err)
        return nil
    }
    return msg
}

