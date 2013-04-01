// Code generated by protoc-gen-go.
// source: smpc.proto
// DO NOT EDIT!

package proto

import proto1 "code.google.com/p/goprotobuf/proto"
import json "encoding/json"
import math "math"

// Reference proto, json, and math imports to suppress error if they are not otherwise used.
var _ = proto1.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type Action_Action int32

const (
	Action_Add      Action_Action = 0
	Action_Mul      Action_Action = 1
	Action_Set      Action_Action = 2
	Action_Retrieve Action_Action = 3
	Action_Cmp      Action_Action = 4
	Action_Neq      Action_Action = 5
	Action_Eqz      Action_Action = 6
	Action_Neqz     Action_Action = 7
)

var Action_Action_name = map[int32]string{
	0: "Add",
	1: "Mul",
	2: "Set",
	3: "Retrieve",
	4: "Cmp",
	5: "Neq",
	6: "Eqz",
	7: "Neqz",
}
var Action_Action_value = map[string]int32{
	"Add":      0,
	"Mul":      1,
	"Set":      2,
	"Retrieve": 3,
	"Cmp":      4,
	"Neq":      5,
	"Eqz":      6,
	"Neqz":     7,
}

func (x Action_Action) Enum() *Action_Action {
	p := new(Action_Action)
	*p = x
	return p
}
func (x Action_Action) String() string {
	return proto1.EnumName(Action_Action_name, int32(x))
}
func (x Action_Action) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}
func (x *Action_Action) UnmarshalJSON(data []byte) error {
	value, err := proto1.UnmarshalJSONEnum(Action_Action_value, data, "Action_Action")
	if err != nil {
		return err
	}
	*x = Action_Action(value)
	return nil
}

type Response_Status int32

const (
	Response_OK    Response_Status = 0
	Response_Error Response_Status = 1
	Response_Val   Response_Status = 2
)

var Response_Status_name = map[int32]string{
	0: "OK",
	1: "Error",
	2: "Val",
}
var Response_Status_value = map[string]int32{
	"OK":    0,
	"Error": 1,
	"Val":   2,
}

func (x Response_Status) Enum() *Response_Status {
	p := new(Response_Status)
	*p = x
	return p
}
func (x Response_Status) String() string {
	return proto1.EnumName(Response_Status_name, int32(x))
}
func (x Response_Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}
func (x *Response_Status) UnmarshalJSON(data []byte) error {
	value, err := proto1.UnmarshalJSONEnum(Response_Status_value, data, "Response_Status")
	if err != nil {
		return err
	}
	*x = Response_Status(value)
	return nil
}

type IntermediateData_DataType int32

const (
	IntermediateData_Mul                IntermediateData_DataType = 0
	IntermediateData_SyncBeacon         IntermediateData_DataType = 1
	IntermediateData_SyncBeaconReceived IntermediateData_DataType = 2
)

var IntermediateData_DataType_name = map[int32]string{
	0: "Mul",
	1: "SyncBeacon",
	2: "SyncBeaconReceived",
}
var IntermediateData_DataType_value = map[string]int32{
	"Mul":                0,
	"SyncBeacon":         1,
	"SyncBeaconReceived": 2,
}

func (x IntermediateData_DataType) Enum() *IntermediateData_DataType {
	p := new(IntermediateData_DataType)
	*p = x
	return p
}
func (x IntermediateData_DataType) String() string {
	return proto1.EnumName(IntermediateData_DataType_name, int32(x))
}
func (x IntermediateData_DataType) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}
func (x *IntermediateData_DataType) UnmarshalJSON(data []byte) error {
	value, err := proto1.UnmarshalJSONEnum(IntermediateData_DataType_value, data, "IntermediateData_DataType")
	if err != nil {
		return err
	}
	*x = IntermediateData_DataType(value)
	return nil
}

type Action struct {
	RequestCode      *int64         `protobuf:"varint,1,req,name=request_code" json:"request_code,omitempty"`
	Action           *Action_Action `protobuf:"varint,2,req,name=action,enum=proto.Action_Action" json:"action,omitempty"`
	Result           *string        `protobuf:"bytes,3,req,name=result" json:"result,omitempty"`
	Share0           *string        `protobuf:"bytes,4,opt,name=share0" json:"share0,omitempty"`
	Share1           *string        `protobuf:"bytes,5,opt,name=share1" json:"share1,omitempty"`
	Value            *int64         `protobuf:"varint,6,opt,name=value" json:"value,omitempty"`
	XXX_unrecognized []byte         `json:"-"`
}

func (this *Action) Reset()         { *this = Action{} }
func (this *Action) String() string { return proto1.CompactTextString(this) }
func (*Action) ProtoMessage()       {}

func (this *Action) GetRequestCode() int64 {
	if this != nil && this.RequestCode != nil {
		return *this.RequestCode
	}
	return 0
}

func (this *Action) GetAction() Action_Action {
	if this != nil && this.Action != nil {
		return *this.Action
	}
	return 0
}

func (this *Action) GetResult() string {
	if this != nil && this.Result != nil {
		return *this.Result
	}
	return ""
}

func (this *Action) GetShare0() string {
	if this != nil && this.Share0 != nil {
		return *this.Share0
	}
	return ""
}

func (this *Action) GetShare1() string {
	if this != nil && this.Share1 != nil {
		return *this.Share1
	}
	return ""
}

func (this *Action) GetValue() int64 {
	if this != nil && this.Value != nil {
		return *this.Value
	}
	return 0
}

type Response struct {
	RequestCode      *int64           `protobuf:"varint,1,req,name=request_code" json:"request_code,omitempty"`
	Client           *int32           `protobuf:"varint,2,req,name=client" json:"client,omitempty"`
	Status           *Response_Status `protobuf:"varint,3,req,name=status,enum=proto.Response_Status" json:"status,omitempty"`
	Share            *int64           `protobuf:"varint,4,opt,name=share" json:"share,omitempty"`
	XXX_unrecognized []byte           `json:"-"`
}

func (this *Response) Reset()         { *this = Response{} }
func (this *Response) String() string { return proto1.CompactTextString(this) }
func (*Response) ProtoMessage()       {}

func (this *Response) GetRequestCode() int64 {
	if this != nil && this.RequestCode != nil {
		return *this.RequestCode
	}
	return 0
}

func (this *Response) GetClient() int32 {
	if this != nil && this.Client != nil {
		return *this.Client
	}
	return 0
}

func (this *Response) GetStatus() Response_Status {
	if this != nil && this.Status != nil {
		return *this.Status
	}
	return 0
}

func (this *Response) GetShare() int64 {
	if this != nil && this.Share != nil {
		return *this.Share
	}
	return 0
}

type IntermediateData struct {
	Type             *IntermediateData_DataType `protobuf:"varint,1,req,name=type,enum=proto.IntermediateData_DataType" json:"type,omitempty"`
	RequestCode      *int64                     `protobuf:"varint,2,req,name=request_code" json:"request_code,omitempty"`
	Client           *int32                     `protobuf:"varint,3,req,name=client" json:"client,omitempty"`
	Step             *int32                     `protobuf:"varint,4,req,name=step" json:"step,omitempty"`
	Data             *int64                     `protobuf:"varint,5,opt,name=data" json:"data,omitempty"`
	XXX_unrecognized []byte                     `json:"-"`
}

func (this *IntermediateData) Reset()         { *this = IntermediateData{} }
func (this *IntermediateData) String() string { return proto1.CompactTextString(this) }
func (*IntermediateData) ProtoMessage()       {}

func (this *IntermediateData) GetType() IntermediateData_DataType {
	if this != nil && this.Type != nil {
		return *this.Type
	}
	return 0
}

func (this *IntermediateData) GetRequestCode() int64 {
	if this != nil && this.RequestCode != nil {
		return *this.RequestCode
	}
	return 0
}

func (this *IntermediateData) GetClient() int32 {
	if this != nil && this.Client != nil {
		return *this.Client
	}
	return 0
}

func (this *IntermediateData) GetStep() int32 {
	if this != nil && this.Step != nil {
		return *this.Step
	}
	return 0
}

func (this *IntermediateData) GetData() int64 {
	if this != nil && this.Data != nil {
		return *this.Data
	}
	return 0
}

func init() {
	proto1.RegisterEnum("proto.Action_Action", Action_Action_name, Action_Action_value)
	proto1.RegisterEnum("proto.Response_Status", Response_Status_name, Response_Status_value)
	proto1.RegisterEnum("proto.IntermediateData_DataType", IntermediateData_DataType_name, IntermediateData_DataType_value)
}
