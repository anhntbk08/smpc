package main
import (
        "github.com/apanda/smpc/core"
        "fmt"
        sproto "github.com/apanda/smpc/proto"
        )

// Set the value of a share
func (state *ComputePeerState) SetValue (action *sproto.Action) (*sproto.Response) {
    fmt.Println("Setting value ", *action.Result, *action.Value)
    result := *action.Result
    val := *action.Value
    state.SharesSet(result, int64(val))
    fmt.Println("Returned from sharesset")
    resp := &sproto.Response{}
    rcode := action.GetRequestCode()
    resp.RequestCode = &rcode
    status := sproto.Response_OK
    resp.Status = &status
    client := int32(state.Client)
    resp.Client = &client
    fmt.Println("Done setting value")
    return resp
}

func (state *ComputePeerState) failResponse (request int64) (*sproto.Response) {
    resp := &sproto.Response{}
    resp.RequestCode = &request
    status := sproto.Response_Error
    resp.Status = &status
    client := int32(state.Client)
    resp.Client = &client
    fmt.Println("Done setting")
    return resp
}

// Retrieve the value of a share
func (state *ComputePeerState) GetValue (action *sproto.Action) (*sproto.Response) {
    result := *action.Result
    val, hasVal := state.SharesGet(result)
    if hasVal {
        resp := &sproto.Response{}
        rcode := action.GetRequestCode()
        resp.RequestCode = &rcode
        resp.Share = &val
        status := sproto.Response_Val
        resp.Status = &status
        client := int32(state.Client)
        resp.Client = &client
        return resp
    }
    return state.failResponse (action.GetRequestCode())
}

func (state *ComputePeerState) okResponse(rcode int64) (*sproto.Response) {
    resp := &sproto.Response{}
    resp.RequestCode = &rcode
    status := sproto.Response_OK
    resp.Status = &status
    client := int32(state.Client)
    resp.Client = &client
    return resp
}

// Add two shares
func (state *ComputePeerState) Add (action *sproto.Action) (*sproto.Response) {
    fmt.Println("Adding two values")
    result := *action.Result
    share0 := *action.Share0
    share1 := *action.Share1
    share0val, hasShare0val := state.SharesGet(share0)
    share1val, hasShare1val := state.SharesGet(share1)
    if hasShare0val && hasShare1val {
        state.SharesSet(result, core.Add(share0val, share1val))
        fmt.Println("Done Adding")
        return state.okResponse(action.GetRequestCode())
    }
    return state.failResponse (action.GetRequestCode())
}

func (state *ComputePeerState) DefaultAction (action *sproto.Action) (*sproto.Response) {
    fmt.Println("Illegal action")
    return state.failResponse(action.GetRequestCode())
}

// Multiply two shares
func (state *ComputePeerState) Mul (action *sproto.Action) (*sproto.Response) {
    fmt.Println("Multiplying two values")
    result := *action.Result
    share0 := *action.Share0
    share1 := *action.Share1
    share0val, hasShare0val := state.SharesGet(share0)
    share1val, hasShare1val := state.SharesGet(share1)
    rcode := *action.RequestCode
    if hasShare0val && hasShare1val {
        inputs := make([]int64, state.NumClients)
        outputs := core.MultShares(share0val, share1val, int32(state.NumClients)) // Output start distributing
        inputs[state.Client] = outputs[state.Client]
        for k := 0; k < state.NumClients; k++ {
           if k == state.Client {
               continue
           }
           intermediate := &sproto.IntermediateData{}
           rcodee := rcode
           intermediate.RequestCode = &rcodee
           t := sproto.IntermediateData_Mul
           intermediate.Type = &t
           step := int32(1)
           intermediate.Step = &step
           client := int32(state.Client)
           intermediate.Client = &client
           intermediate.Data = &outputs[k]
           fmt.Printf("Sending %d -> %d\n", state.Client, k)
           m := IntermediateToMsg(intermediate)
           if m == nil {
               panic("Error with intermediate\n")
           }
           state.PeerOutChannels[k].Out() <- m
        }
        fmt.Println("Done sending multiplication data")
        responses := 1 // We already have our own response
        ch := state.ChannelForRequest(rcode)
        for responses < state.NumClients {
            fmt.Printf("waiting for %d intermediate data now\n", state.NumClients - responses)
            var intermediate *sproto.IntermediateData
            select {
                case intermediate = <- ch:
            }
            fmt.Printf("Mul received %d->%d\n", *intermediate.Client, state.Client)
            responses += 1
            inputs[*intermediate.Client] = *intermediate.Data
        }
        fmt.Printf("Intermediate data collected\n")
        share := core.MultCombineShares(&inputs, int32(state.NumClients))
        state.SharesSet(result, share)
        fmt.Println("Done multiplying\n")
        state.UnregisterChannelForRequest(rcode)
        return state.okResponse(action.GetRequestCode())
    }
    return state.failResponse (action.GetRequestCode())
}
