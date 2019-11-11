package client

import "github.com/masonchen2014/easymicro/protocol"

//call set function
type CallOption func(*Call)

type BeforeOrAfterCallOption struct {
	option CallOption
	after  bool
}

//set heart beat call
func SetCallHeartBeat() BeforeOrAfterCallOption {
	return BeforeOrAfterCallOption{
		option: func(call *Call) {
			call.heartBeat = true
		},
		after: false,
	}
}

//set call compress type
func SetCallCompressType(cType protocol.CompressType) BeforeOrAfterCallOption {
	return BeforeOrAfterCallOption{
		option: func(call *Call) {
			call.compressType = cType
		},
		after: false,
	}
}

//set call serialize type
func SetCallSerializeType(sType protocol.SerializeType) BeforeOrAfterCallOption {
	return BeforeOrAfterCallOption{
		option: func(call *Call) {
			call.serializeType = sType
		},
		after: false,
	}
}

func GetMetadataFromServer(md *map[string]string) BeforeOrAfterCallOption {
	return BeforeOrAfterCallOption{
		option: func(call *Call) {
			*md = call.Metadata
		},
		after: true,
	}
}

/*func SetCallMetaData(meta map[string]string) CallOption {
	return func(call *Call) {
		call.metaData = meta
	}
}*/
