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
    //state.Shares[result] = int64(val)
    state.SharesSet(result, int64(val))
    fmt.Println("Set map value, preparing RESPONSE")
    resp := &sproto.Response{}
    rcode := action.GetRequestCode()
    fmt.Println("Set map value, set response code, code is ", rcode)
    resp.RequestCode = &rcode
    status := sproto.Response_OK
    resp.Status = &status
    client := int32(state.Client)
    resp.Client = &client
    fmt.Println("Done setting")
    return resp
}

// Retrieve the value of a share
func (state *ComputePeerState) GetValue (action *sproto.Action) (*sproto.Response) {
    state.OutstandingRequests.Done() // First remove this particular request from what is outstanding
    state.OutstandingRequests.Wait() // Then wait for everything else to finish
    state.OutstandingRequests.Add(1) // Then go back to accounting for this request
    result := *action.Result
    val := state.SharesGet(result)
    fmt.Println("Got the map value, preparing RESPONSE")
    resp := &sproto.Response{}
    rcode := action.GetRequestCode()
    fmt.Println("Set map value, set response code, code is ", rcode)
    resp.RequestCode = &rcode
    resp.Share = &val
    status := sproto.Response_Val
    resp.Status = &status
    client := int32(state.Client)
    resp.Client = &client
    fmt.Println("Done setting")
    return resp
}

// Add two shares
func (state *ComputePeerState) Add (action *sproto.Action) (*sproto.Response) {
    fmt.Println("Adding two values")
    result := *action.Result
    share0 := *action.Share0
    share1 := *action.Share1
    // Maybe do this automically
    state.SharesSet(result, core.Add(state.SharesGet(share0), state.SharesGet(share1)))
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

