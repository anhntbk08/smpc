package main
import (
        "github.com/apanda/smpc/core"
        "fmt"
        sproto "github.com/apanda/smpc/proto"
        )

var _ = fmt.Println
// Set the value of a share
func (state *ComputePeerState) SetValue (action *sproto.Action) (*sproto.Response) {
    //fmt.Println("Setting value ", *action.Result, *action.Value)
    result := *action.Result
    val := *action.Value
    state.SharesSet(result, int64(val))
    //fmt.Println("Returned from sharesset")
    resp := &sproto.Response{}
    rcode := action.GetRequestCode()
    resp.RequestCode = &rcode
    status := sproto.Response_OK
    resp.Status = &status
    client := int32(state.Client)
    resp.Client = &client
    //fmt.Println("Done setting value")
    return resp
}

func (state *ComputePeerState) failResponse (request int64) (*sproto.Response) {
    resp := &sproto.Response{}
    resp.RequestCode = &request
    status := sproto.Response_Error
    resp.Status = &status
    client := int32(state.Client)
    resp.Client = &client
    //fmt.Println("Done setting")
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

// Remove the value for a share
func (state *ComputePeerState) RemoveValue (action *sproto.Action) (*sproto.Response) {
    result := *action.Result
    _, hasVal := state.SharesGet(result)
    if hasVal {
        //fmt.Println("Deleting value ", *action.Result)
        result := *action.Result
        state.SharesDelete(result)
        //fmt.Println("Returned from sharesdelete")
        resp := &sproto.Response{}
        rcode := action.GetRequestCode()
        resp.RequestCode = &rcode
        status := sproto.Response_OK
        resp.Status = &status
        client := int32(state.Client)
        resp.Client = &client
        //fmt.Println("Done setting value")
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
    //fmt.Println("Adding two values")
    result := *action.Result
    share0 := *action.Share0
    share1 := *action.Share1
    share0val, hasShare0val := state.SharesGet(share0)
    share1val, hasShare1val := state.SharesGet(share1)
    if hasShare0val && hasShare1val {
        state.SharesSet(result, core.Add(share0val, share1val))
        //fmt.Println("Done Adding")
        return state.okResponse(action.GetRequestCode())
    }
    //fmt.Printf("Failed additions, response would have been %s (%s, %v, %s, %v)\n", result, share0, hasShare0val, share1, hasShare1val)
    return state.failResponse (action.GetRequestCode())
}

func (state *ComputePeerState) OneSub (action *sproto.Action) (*sproto.Response) {
    //fmt.Println("Subtraction one from value")
    result := *action.Result
    share0 := *action.Share0
    share0val, hasShare0val := state.SharesGet(share0)
    if hasShare0val {
        val := core.Sub(1, share0val)
        state.SharesSet(result, val)
    }
    return state.failResponse (action.GetRequestCode())

}

func (state *ComputePeerState) DefaultAction (action *sproto.Action) (*sproto.Response) {
    //fmt.Println("Illegal action")
    return state.failResponse(action.GetRequestCode())
}

func (state *ComputePeerState) mul (share0 int64, share1 int64, rcode int64, step int32) (int64) {
   //fmt.Printf("Attempting to mul for %d %d\n", rcode, step)
    ch := state.ChannelForRequest(*MakeRequestStep(rcode, step))
   //fmt.Printf("Making input array for %d %d\n", rcode, step)
    inputs := make([]int64, state.NumClients)
   //fmt.Printf("Computing mult shares for %d %d\n", rcode, step)
    outputs := core.MultShares(share0, share1, int32(state.NumClients)) // Output start distributing
   //fmt.Printf("Setting self share for %d %d\n", rcode, step)
    inputs[state.Client] = outputs[state.Client]
    for k := 0; k < state.NumClients; k++ {
       if k == state.Client {
           continue
       }
      //fmt.Printf("Sending share to %d for %d %d\n", k, rcode, step)
       intermediate := &sproto.IntermediateData{}
       rcodee := rcode
       intermediate.RequestCode = &rcodee
       t := sproto.IntermediateData_Mul
       intermediate.Type = &t
       intermediate.Step = &step
       client := int32(state.Client)
       intermediate.Client = &client
       intermediate.Data = &outputs[k]
       fmt.Printf("Sending %d -> %d (%d %d)\n", state.Client, k, rcode, step)
       m := IntermediateToMsg(intermediate)
       //fmt.Printf("Done creating share %d for %d %d\n", k, rcode, step)
       if m == nil {
           panic("Error with intermediate\n")
       }
      fmt.Printf("Sending intermediate for mul for %d %d\n", rcode, step)
       state.PeerOutChannels[k].Out() <- m
    }
    //fmt.Println("Done sending multiplication data")
    responses := 1 // We already have our own response
    for responses < state.NumClients {
        fmt.Printf("waiting for %d intermediate data now %d %d\n", state.NumClients - responses, rcode, step)
        var intermediate *sproto.IntermediateData
        intermediate = <- ch
        fmt.Printf("Mul received %d->%d (%d %d)\n", *intermediate.Client, state.Client, rcode, step)
        responses += 1
        inputs[*intermediate.Client] = *intermediate.Data
    }
    //fmt.Printf("Intermediate data collected\n")
    share := core.MultCombineShares(&inputs, int32(state.NumClients))
    //fmt.Printf("Done multiplying\n")
    return share
}
// Multiply two shares
func (state *ComputePeerState) Mul (action *sproto.Action) (*sproto.Response) {
    //fmt.Println("Multiplying two values")
    result := *action.Result
    share0 := *action.Share0
    share1 := *action.Share1
    share0val, hasShare0val := state.SharesGet(share0)
    share1val, hasShare1val := state.SharesGet(share1)
    //fmt.Printf("Multiplying, will set %s\n", result)
    rcode := *action.RequestCode
    if hasShare0val && hasShare1val {
        share := state.mul(share0val, share1val, rcode, 1)
        //fmt.Printf("Done multiplying, setting %s\n", result)
        state.SharesSet(result, share)
        //fmt.Printf("Done multiplying, done setting %s\n", result)
        //fmt.Printf("Done multiplying\n")
        state.UnregisterChannelForRequest(*MakeRequestStep(rcode, 1))
        return state.okResponse(action.GetRequestCode())
    } else {
        //fmt.Printf("Not setting %s, could not find operands %s %v %s %v\n", result, share0, hasShare0val, share1, hasShare1val)
    }
    return state.failResponse (action.GetRequestCode())
}

// Test if a value is 0
func (state* ComputePeerState) neqz (val int64, rcode int64) (int64) {
    exponent := core.LargePrime - 1 // We are going to raise the number to this power. 
    res := int64(1)
    step := int32(0)
    fmt.Printf("Computing neqz for rcode %d\n", rcode)
    for exponent > 0 {
        if (exponent & 1 == 1) {
            fmt.Printf("Computing neqz (mul) for rcode %d, step %d\n", rcode, step)
            res = state.mul(res, val, rcode, step)
            fmt.Printf("Done computing neqz (mul) for rcode %d, step %d\n", rcode, step)
            state.UnregisterChannelForRequest(*MakeRequestStep(rcode, step))
            step += 1
        }
        exponent >>= 1
        fmt.Printf("Computing neqz (mul) for rcode %d, step %d\n", rcode, step)
        val = state.mul(val, val, rcode, step)
        fmt.Printf("Done computing neqz (mul) for rcode %d, step %d\n", rcode, step)
        state.UnregisterChannelForRequest(*MakeRequestStep(rcode, step))
        step += 1
    }
    fmt.Printf("Done computing neqz %d\n", rcode)
    return res
}

func (state* ComputePeerState) ncmp (share0val int64, share1val int64, rcode int64) (int64) {
    a := core.Sub(share0val, share1val) 
    return state.neqz (a, rcode)
}

// Inequality
func (state *ComputePeerState) Neq (action *sproto.Action) (*sproto.Response) {
    //fmt.Println("Comparing values")
    result := *action.Result
    share0 := *action.Share0
    share1 := *action.Share1
    share0val, hasShare0Val := state.SharesGet(share0)
    share1val, hasShare1Val := state.SharesGet(share1)
    rcode := *action.RequestCode
    if !hasShare0Val || !hasShare1Val {
        return state.failResponse (action.GetRequestCode())
    }
    res := state.ncmp(share0val, share1val, rcode)
    state.SharesSet(result, res)
    //fmt.Printf("Done comparing values\n")
    return state.okResponse (action.GetRequestCode())
}

// Equality operation
func (state *ComputePeerState) Cmp (action *sproto.Action) (*sproto.Response) {
    //fmt.Println("Comparing values")
    result := *action.Result
    share0 := *action.Share0
    share1 := *action.Share1
    share0val, hasShare0Val := state.SharesGet(share0)
    share1val, hasShare1Val := state.SharesGet(share1)
    rcode := *action.RequestCode
    if !hasShare0Val || !hasShare1Val {
        return state.failResponse (action.GetRequestCode())
    }
    res := state.ncmp(share0val, share1val, rcode)
    one := int64(1) 
    res = core.Sub(one, res)
    state.SharesSet(result, res)
    //fmt.Printf("Done comparing values\n")
    return state.okResponse (action.GetRequestCode())
}

// Test equality to zero
func (state *ComputePeerState) Eqz (action *sproto.Action) (*sproto.Response) {
    //fmt.Println("Comparing values")
    result := *action.Result
    share0 := *action.Share0
    share0val, hasShare0Val := state.SharesGet(share0)
    rcode := *action.RequestCode
    if !hasShare0Val {
        return state.failResponse (action.GetRequestCode())
    }
    res := state.neqz(share0val, rcode)
    one := int64(1) 
    res = core.Sub(one, res)
    state.SharesSet(result, res)
    //fmt.Printf("Done testing value for zero\n")
    return state.okResponse (action.GetRequestCode())
}

// Test inequality to zero
func (state *ComputePeerState) Neqz (action *sproto.Action) (*sproto.Response) {
    //fmt.Println("Comparing values")
    result := *action.Result
    share0 := *action.Share0
    share0val, hasShare0Val := state.SharesGet(share0)
    rcode := *action.RequestCode
    if !hasShare0Val {
        //fmt.Printf("Failing, variable %s not found\n", share0)
        return state.failResponse (action.GetRequestCode())
    }
    res := state.neqz(share0val, rcode)
    state.SharesSet(result, res)
    //fmt.Printf("Done testing value for not zero\n")
    return state.okResponse (action.GetRequestCode())
}

// Compare to const
func (state *ComputePeerState) CmpConst (action *sproto.Action) (*sproto.Response) {
    //fmt.Println("Comparing to constant")
    result := *action.Result
    share0 := *action.Share0
    val := *action.Value
    share0val, hasShare0Val := state.SharesGet(share0)
    rcode := *action.RequestCode
    if !hasShare0Val {
        return state.failResponse (action.GetRequestCode())
    }
   //fmt.Printf("For rcode %d (CMPCONST) subtracting\n", rcode)
    res := core.Sub(val, share0val)
   //fmt.Printf("For rcode %d (CMPCONST) neqz\n", rcode)
    res = state.neqz(res, rcode)
   //fmt.Printf("For rcode %d (CMPCONST) done with neqz\n", rcode)
    one := int64(1)
   //fmt.Printf("For rcode %d (CMPCONST) subtracting\n", rcode)
    res = core.Sub(one, res)
   //fmt.Printf("For rcode %d (CMPCONST) setting share\n", rcode)
    state.SharesSet(result, res)
    //fmt.Printf("Done comparing to const\n")
    return state.okResponse(action.GetRequestCode())
}

// Compare to const
func (state *ComputePeerState) NeqConst (action *sproto.Action) (*sproto.Response) {
    //fmt.Println("Comparing to constant")
    result := *action.Result
    share0 := *action.Share0
    val := *action.Value
    share0val, hasShare0Val := state.SharesGet(share0)
    rcode := *action.RequestCode
    if !hasShare0Val {
        //fmt.Printf("Failing, variable %s not found\n", share0)
        return state.failResponse (action.GetRequestCode())
    }
    res := core.Sub(val, share0val)
    //fmt.Printf("Subtractiong const %d", val)
    res = state.neqz(res, rcode)
    state.SharesSet(result, res)
    //fmt.Printf("Done comparing to const\n")
    return state.okResponse(action.GetRequestCode())
}

// Multiply const
func (state *ComputePeerState) MulConst (action *sproto.Action) (*sproto.Response) {
    //fmt.Println("Multiplying with constants")
    result := *action.Result
    share0 := *action.Share0
    val := *action.Value
    share0val, hasShare0Val := state.SharesGet(share0)
    rcode := *action.RequestCode
    if !hasShare0Val {
        //fmt.Printf("Failing, variable %s not found\n", share0)
        return state.failResponse (action.GetRequestCode())
    }
    res := core.MultUnderMod(val, share0val)
    state.SharesSet(result, res)
    //fmt.Println("Done multiplying constants")
    return state.okResponse(rcode)
}
// Add const
// Sub const
