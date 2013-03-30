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

func (state *ComputePeerState) mul (share0 int64, share1 int64, rcode int64, step int32) (int64) {
    inputs := make([]int64, state.NumClients)
    outputs := core.MultShares(share0, share1, int32(state.NumClients)) // Output start distributing
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
    ch := state.ChannelForRequest(*MakeRequestStep(rcode, step))
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
    fmt.Println("Done multiplying\n")
    return share
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
        share := state.mul(share0val, share1val, rcode, 1)
        state.SharesSet(result, share)
        fmt.Println("Done multiplying\n")
        state.UnregisterChannelForRequest(*MakeRequestStep(rcode, 1))
        return state.okResponse(action.GetRequestCode())
    }
    return state.failResponse (action.GetRequestCode())
}

// Equality operation
func (state *ComputePeerState) Cmp (action *sproto.Action) (*sproto.Response) {
    fmt.Println("Comparing values")
    result := *action.Result
    share0 := *action.Share0
    share1 := *action.Share1
    share0val, hasShare0Val := state.SharesGet(share0)
    share1val, hasShare1Val := state.SharesGet(share1)
    rcode := *action.RequestCode
    if !hasShare0Val || !hasShare1Val {
        return state.failResponse (action.GetRequestCode())
    }
    exponent := core.LargePrime - 1 // We are going to raise the number to this power. 
    a := core.Sub(share0val, share1val) 
    res := int64(1)
    step := int32(0)
    for exponent > 0 {
        if (exponent & 1 == 1) {
            res = state.mul(res, a, rcode, step)
            state.UnregisterChannelForRequest(*MakeRequestStep(rcode, step))
            step += 1
        }
        exponent >>= 1
        a = state.mul(a, a, rcode, step)
        state.UnregisterChannelForRequest(*MakeRequestStep(rcode, step))
        step += 1
    }
    one := int64(1) 
    res = core.Sub(one, res)
    state.SharesSet(result, res)
    fmt.Println("Done comparing values\n")
    return state.okResponse (action.GetRequestCode())
}

