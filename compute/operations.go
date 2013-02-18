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

// Add two shares
func (state *ComputePeerState) Add (action *sproto.Action) (*sproto.Response) {
    fmt.Println("Adding two values")
    result := *action.Result
    share0 := *action.Share0
    share1 := *action.Share1
    share0val, hasShare0val := state.SharesGet(share0)
    share1val, hasShare1val := state.SharesGet(share1)
    if hasShare0val && hasShare1val {
        // Maybe do this automically
        state.SharesSet(result, core.Add(share0val, share1val))
        resp := &sproto.Response{}
        rcode := action.GetRequestCode()
        resp.RequestCode = &rcode
        status := sproto.Response_OK
        resp.Status = &status
        client := int32(state.Client)
        resp.Client = &client
        fmt.Println("Done Adding")
        return resp
    }
    return state.failResponse (action.GetRequestCode())
}

