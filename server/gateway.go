package server

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/masonchen2014/easymicro/log"
	"github.com/masonchen2014/easymicro/protocol"
	"github.com/masonchen2014/easymicro/share"
	"github.com/soheilhy/cmux"
)

const (
	EmVersion           = "Easymicro-Version"
	EmMessageType       = "Easymicro-MesssageType"
	EmHeartbeat         = "Easymicro-Heartbeat"
	EmOneway            = "Easymicro-Oneway"
	EmMessageStatusType = "Easymicro-MessageStatusType"
	EmSerializeType     = "Easymicro-SerializeType"
	EmMessageID         = "Easymicro-MessageID"
	EmServicePath       = "Easymicro-ServicePath"
	EmServiceMethod     = "Easymicro-ServiceMethod"
	EmMeta              = "Easymicro-Meta"
	EmErrorMessage      = "Easymicro-ErrorMessage"
)

func (s *Server) startGateway(ln net.Listener) net.Listener {
	m := cmux.New(ln)
	eLn := m.Match(easyMicroPrefixByteMatcher())
	httpLn := m.Match(cmux.HTTP1Fast())
	go s.startHTTP1APIGateway(httpLn)
	go m.Serve()
	return eLn
}

func easyMicroPrefixByteMatcher() cmux.Matcher {
	magic := protocol.MagicNumber()
	return func(r io.Reader) bool {
		buf := make([]byte, 1)
		n, _ := r.Read(buf)
		return n == 1 && buf[0] == magic
	}
}

func (s *Server) startHTTP1APIGateway(ln net.Listener) {
	router := httprouter.New()
	router.POST("/:servicePath/:serviceMethod", s.handleGatewayRequest)
	router.GET("/:servicePath/:serviceMethod", s.handleGatewayRequest)
	if err := http.ListenAndServe(s.gateWayAddr, router); err != nil {
		log.Panicf("error in gateway Serve: %s", err)
	}
}

func (s *Server) handleGatewayRequest(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var err error
	var req *protocol.Message
	wh := w.Header()
	//set headers
	switch {
	default:
		version := r.Header.Get(EmVersion)
		wh.Set(EmVersion, version)

		messageId := r.Header.Get(EmMessageID)
		wh.Set(EmMessageID, messageId)

		servicePath := params.ByName("servicePath")
		if servicePath == "" {
			err = errors.New("empty servicepath")
			break
		}

		wh.Set(EmServicePath, servicePath)
		serviceMethod := params.ByName("serviceMethod")
		if serviceMethod == "" {
			err = errors.New("empty servicemethod")
			break
		}
		wh.Set(EmServiceMethod, serviceMethod)

		serializedType := r.Header.Get(EmSerializeType)
		if serializedType == "" {
			err = errors.New("empty serialized type")
			break
		}
		serializedTypeInt, err := strconv.Atoi(serializedType)
		if err != nil {
			err = errors.New("invalid serialized type")
			break
		}
		wh.Set(EmSerializeType, r.Header.Get(EmSerializeType))

		messageIdInt, err := strconv.ParseUint(messageId, 10, 64)
		if err != nil {
			err = errors.New("invalid message id")
			break
		}

		req, err = HTTPRequest2EmRpcRequest(r)
		if err != nil {
			break
		}

		req.SetSeq(messageIdInt)
		req.SetSerializeType(protocol.SerializeType(serializedTypeInt))
		req.ServicePath = strings.ToLower(servicePath)
		req.ServiceMethod = strings.ToLower(serviceMethod)
		defer protocol.FreeMsg(req)
	}

	if err != nil {
		rh := r.Header
		for k, v := range rh {
			if strings.HasPrefix(k, "Easymicro-") && len(v) > 0 {
				wh.Set(k, v[0])
			}
		}

		wh.Set(EmMessageStatusType, "Error")
		wh.Set(EmErrorMessage, err.Error())
		w.WriteHeader(400)
		return
	}

	resMetadata := make(map[string]string)
	newCtx, err := extractClientMdContexFromMd(context.Background(), req.Metadata)
	res, err := s.handleRequest(newCtx, req)
	defer protocol.FreeMsg(res)

	if err != nil {
		log.Warnf("easymicro: failed to handle gateway request: %v", err)
		wh.Set(EmMessageStatusType, "Error")
		wh.Set(EmErrorMessage, err.Error())
		w.WriteHeader(500)
		return
	}

	if len(resMetadata) > 0 { //copy meta in context to request
		meta := res.Metadata
		if meta == nil {
			res.Metadata = resMetadata
		} else {
			for k, v := range resMetadata {
				meta[k] = v
			}
		}
	}

	meta := url.Values{}
	for k, v := range res.Metadata {
		meta.Add(k, v)
	}
	wh.Set(EmMeta, meta.Encode())
	w.Write(res.Payload)
}

func HTTPRequest2EmRpcRequest(r *http.Request) (*protocol.Message, error) {
	req := protocol.GetPooledMsg()
	req.SetMessageType(protocol.Request)

	h := r.Header
	heartbeat := h.Get(EmHeartbeat)
	if heartbeat != "" {
		req.SetHeartbeat(true)
	}

	oneway := h.Get(EmOneway)
	if oneway != "" {
		req.SetOneway(true)
	}

	meta := h.Get(EmMeta)
	if meta != "" {
		metadata, err := url.ParseQuery(meta)
		if err != nil {
			return nil, err
		}
		mm := make(map[string]string)
		for k, v := range metadata {
			if len(v) > 0 {
				mm[k] = v[0]
			}
		}
		req.Metadata = mm
	}

	auth := h.Get("Authorization")
	if auth != "" {
		if req.Metadata == nil {
			req.Metadata = make(map[string]string)
		}
		req.Metadata[share.AuthKey] = auth
	}

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	req.Payload = payload
	return req, nil
}
