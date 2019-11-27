// Code generated by protoc-gen-go. DO NOT EDIT.
// source: arith.proto

/*
Package arith is a generated protocol buffer package.

It is generated from these files:
	arith.proto

It has these top-level messages:
	MulReq
	MulReply
*/
package arith

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

type MulReq struct {
	A int64 `protobuf:"varint,1,opt,name=a" json:"a,omitempty"`
	B int64 `protobuf:"varint,2,opt,name=b" json:"b,omitempty"`
}

func (m *MulReq) Reset()                    { *m = MulReq{} }
func (m *MulReq) String() string            { return proto.CompactTextString(m) }
func (*MulReq) ProtoMessage()               {}
func (*MulReq) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *MulReq) GetA() int64 {
	if m != nil {
		return m.A
	}
	return 0
}

func (m *MulReq) GetB() int64 {
	if m != nil {
		return m.B
	}
	return 0
}

type MulReply struct {
	C int64 `protobuf:"varint,1,opt,name=c" json:"c,omitempty"`
}

func (m *MulReply) Reset()                    { *m = MulReply{} }
func (m *MulReply) String() string            { return proto.CompactTextString(m) }
func (*MulReply) ProtoMessage()               {}
func (*MulReply) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *MulReply) GetC() int64 {
	if m != nil {
		return m.C
	}
	return 0
}

func init() {
	proto.RegisterType((*MulReq)(nil), "arith.MulReq")
	proto.RegisterType((*MulReply)(nil), "arith.MulReply")
}

func init() { proto.RegisterFile("arith.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 123 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x4e, 0x2c, 0xca, 0x2c,
	0xc9, 0xd0, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x05, 0x73, 0x94, 0x54, 0xb8, 0xd8, 0x7c,
	0x4b, 0x73, 0x82, 0x52, 0x0b, 0x85, 0x78, 0xb8, 0x18, 0x13, 0x25, 0x18, 0x15, 0x18, 0x35, 0x98,
	0x83, 0x18, 0x13, 0x41, 0xbc, 0x24, 0x09, 0x26, 0x08, 0x2f, 0x49, 0x49, 0x82, 0x8b, 0x03, 0xac,
	0xaa, 0x20, 0xa7, 0x12, 0x24, 0x93, 0x0c, 0x53, 0x97, 0x6c, 0x64, 0xc0, 0xc5, 0xea, 0x08, 0x32,
	0x48, 0x48, 0x9d, 0x8b, 0xd9, 0xb7, 0x34, 0x47, 0x88, 0x57, 0x0f, 0x62, 0x09, 0xc4, 0x50, 0x29,
	0x7e, 0x64, 0x6e, 0x41, 0x4e, 0xa5, 0x12, 0x43, 0x12, 0x1b, 0xd8, 0x7e, 0x63, 0x40, 0x00, 0x00,
	0x00, 0xff, 0xff, 0xa7, 0xed, 0x91, 0x00, 0x8e, 0x00, 0x00, 0x00,
}