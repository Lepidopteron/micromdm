// Code generated by protoc-gen-go. DO NOT EDIT.
// source: connect.proto

/*
Package connectproto is a generated protocol buffer package.

It is generated from these files:
	connect.proto

It has these top-level messages:
	Event
	Response
*/
package connectproto

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Event struct {
	Id       string    `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Time     int64     `protobuf:"varint,2,opt,name=time" json:"time,omitempty"`
	Response *Response `protobuf:"bytes,3,opt,name=response" json:"response,omitempty"`
	Raw      []byte    `protobuf:"bytes,4,opt,name=raw,proto3" json:"raw,omitempty"`
}

func (m *Event) Reset()                    { *m = Event{} }
func (m *Event) String() string            { return proto.CompactTextString(m) }
func (*Event) ProtoMessage()               {}
func (*Event) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Event) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Event) GetTime() int64 {
	if m != nil {
		return m.Time
	}
	return 0
}

func (m *Event) GetResponse() *Response {
	if m != nil {
		return m.Response
	}
	return nil
}

func (m *Event) GetRaw() []byte {
	if m != nil {
		return m.Raw
	}
	return nil
}

type Response struct {
	Udid        string `protobuf:"bytes,1,opt,name=udid" json:"udid,omitempty"`
	UserId      string `protobuf:"bytes,2,opt,name=user_id,json=userId" json:"user_id,omitempty"`
	Status      string `protobuf:"bytes,3,opt,name=status" json:"status,omitempty"`
	RequestType string `protobuf:"bytes,4,opt,name=request_type,json=requestType" json:"request_type,omitempty"`
	CommandUuid string `protobuf:"bytes,5,opt,name=command_uuid,json=commandUuid" json:"command_uuid,omitempty"`
}

func (m *Response) Reset()                    { *m = Response{} }
func (m *Response) String() string            { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()               {}
func (*Response) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *Response) GetUdid() string {
	if m != nil {
		return m.Udid
	}
	return ""
}

func (m *Response) GetUserId() string {
	if m != nil {
		return m.UserId
	}
	return ""
}

func (m *Response) GetStatus() string {
	if m != nil {
		return m.Status
	}
	return ""
}

func (m *Response) GetRequestType() string {
	if m != nil {
		return m.RequestType
	}
	return ""
}

func (m *Response) GetCommandUuid() string {
	if m != nil {
		return m.CommandUuid
	}
	return ""
}

func init() {
	proto.RegisterType((*Event)(nil), "connectproto.Event")
	proto.RegisterType((*Response)(nil), "connectproto.Response")
}

func init() { proto.RegisterFile("connect.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 225 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x4c, 0x8f, 0xb1, 0x4e, 0xc3, 0x30,
	0x10, 0x86, 0xe5, 0xa4, 0x0d, 0xcd, 0x35, 0x20, 0x74, 0x43, 0xf1, 0x18, 0x3a, 0x65, 0xca, 0x50,
	0x9e, 0x81, 0x81, 0xd5, 0x82, 0x39, 0x0a, 0xf1, 0x0d, 0x1e, 0x62, 0xa7, 0xf6, 0x19, 0xd4, 0x07,
	0xe1, 0x7d, 0x51, 0x8c, 0xa9, 0xba, 0xfd, 0xff, 0xef, 0x4f, 0xfe, 0x74, 0x70, 0x3f, 0x39, 0x6b,
	0x69, 0xe2, 0x7e, 0xf1, 0x8e, 0x1d, 0x36, 0xb9, 0xa6, 0x76, 0x3c, 0xc3, 0xf6, 0xf5, 0x8b, 0x2c,
	0xe3, 0x03, 0x14, 0x46, 0x4b, 0xd1, 0x8a, 0xae, 0x56, 0x85, 0xd1, 0x88, 0xb0, 0x61, 0x33, 0x93,
	0x2c, 0x5a, 0xd1, 0x95, 0x2a, 0x65, 0x3c, 0xc1, 0xce, 0x53, 0x58, 0x9c, 0x0d, 0x24, 0xcb, 0x56,
	0x74, 0xfb, 0xd3, 0xa1, 0xbf, 0xfd, 0xad, 0x57, 0xf9, 0x55, 0x5d, 0x39, 0x7c, 0x84, 0xd2, 0x8f,
	0xdf, 0x72, 0xd3, 0x8a, 0xae, 0x51, 0x6b, 0x3c, 0xfe, 0x08, 0xd8, 0xfd, 0x83, 0xab, 0x26, 0xea,
	0xab, 0x38, 0x65, 0x7c, 0x82, 0xbb, 0x18, 0xc8, 0x0f, 0x46, 0x27, 0x7b, 0xad, 0xaa, 0xb5, 0xbe,
	0x69, 0x3c, 0x40, 0x15, 0x78, 0xe4, 0x18, 0x92, 0xbd, 0x56, 0xb9, 0xe1, 0x33, 0x34, 0x9e, 0xce,
	0x91, 0x02, 0x0f, 0x7c, 0x59, 0x28, 0xc9, 0x6a, 0xb5, 0xcf, 0xdb, 0xfb, 0x65, 0xa1, 0x15, 0x99,
	0xdc, 0x3c, 0x8f, 0x56, 0x0f, 0x31, 0x1a, 0x2d, 0xb7, 0x7f, 0x48, 0xde, 0x3e, 0xa2, 0xd1, 0x9f,
	0x55, 0xba, 0xe1, 0xe5, 0x37, 0x00, 0x00, 0xff, 0xff, 0x0e, 0x78, 0x53, 0x38, 0x30, 0x01, 0x00,
	0x00,
}
