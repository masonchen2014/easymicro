package share

import (
	"github.com/masonchen2014/easymicro/codec"
	"github.com/masonchen2014/easymicro/protocol"
)

const (
	// AuthKey is used in metadata.
	AuthKey          = "__AUTH"
	BaseServicesPath = "services/"

	PrefixTracerState = "ot-tracer-"
	PrefixBaggage     = "ot-baggage-"

	TracerStateFieldCount = 3
	FieldNameTraceID      = PrefixTracerState + "traceid"
	FieldNameSpanID       = PrefixTracerState + "spanid"
	FieldNameSampled      = PrefixTracerState + "sampled"
)

var (
	// Codecs are codecs supported by rpcx. You can add customized codecs in Codecs.
	Codecs = map[protocol.SerializeType]codec.Codec{
		protocol.SerializeNone: &codec.ByteCodec{},
		protocol.JSON:          &codec.JSONCodec{},
		protocol.ProtoBuffer:   &codec.PBCodec{},
		protocol.MsgPack:       &codec.MsgpackCodec{},
		protocol.Thrift:        &codec.ThriftCodec{},
	}
)

// RegisterCodec register customized codec.
func RegisterCodec(t protocol.SerializeType, c codec.Codec) {
	Codecs[t] = c
}

// ReqMetaDataKey is used to set metatdata in context of requests.
type ReqMetaDataKey struct{}

// ResMetaDataKey is used to set metatdata in context of responses.
type ResMetaDataKey struct{}

// SpanMetaDataKey is used to set span metatdata in context of requests.
type SpanMetaDataKey struct{}
