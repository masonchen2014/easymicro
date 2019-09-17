package client

import "github.com/easymicro/protocol"

//client set function
type ClientOption func(*Client)

//call set function
type CallOption func(*Call)

//set heart beat call
func SetCallHeartBeat() CallOption {
	return func(call *Call) {
		call.heartBeat = true
	}
}

//set call compress type
func SetCallCompressType(cType protocol.CompressType) CallOption {
	return func(call *Call) {
		call.compressType = cType
	}
}

//set call serialize type
func SetCallSerializeType(sType protocol.SerializeType) CallOption {
	return func(call *Call) {
		call.serializeType = sType
	}
}

/*func SetCallMetaData(meta map[string]string) CallOption {
	return func(call *Call) {
		call.metaData = meta
	}
}*/
