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
	router.POST("/*servicePath", s.handleGatewayRequest)
	router.GET("/*servicePath", s.handleGatewayRequest)
	if err := http.ListenAndServe(s.gateWayAddr, router); err != nil {
		log.Panicf("error in gateway Serve: %s", err)
	}
}

func (s *Server) handleGatewayRequest(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if r.Header.Get(EmServicePath) == "" {
		servicePath := params.ByName("servicePath")
		if strings.HasPrefix(servicePath, "/") {
			servicePath = servicePath[1:]
		}
		r.Header.Set(EmServicePath, servicePath)
	}
	servicePath := r.Header.Get(EmServicePath)
	wh := w.Header()
	req, err := HTTPRequest2EmRpcRequest(r)
	defer protocol.FreeMsg(req)

	//set headers
	wh.Set(EmVersion, r.Header.Get(EmVersion))
	wh.Set(EmMessageID, r.Header.Get(EmMessageID))

	if err == nil && servicePath == "" {
		err = errors.New("empty servicepath")
	} else {
		wh.Set(EmServicePath, servicePath)
	}

	if err == nil && r.Header.Get(EmServiceMethod) == "" {
		err = errors.New("empty servicemethod")
	} else {
		wh.Set(EmServiceMethod, r.Header.Get(EmServiceMethod))
	}

	if err == nil && r.Header.Get(EmSerializeType) == "" {
		err = errors.New("empty serialized type")
	} else {
		wh.Set(EmSerializeType, r.Header.Get(EmSerializeType))
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
		return
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
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
	seq := h.Get(EmMessageID)
	if seq != "" {
		id, err := strconv.ParseUint(seq, 10, 64)
		if err != nil {
			return nil, err
		}
		req.SetSeq(id)
	}

	heartbeat := h.Get(EmHeartbeat)
	if heartbeat != "" {
		req.SetHeartbeat(true)
	}

	oneway := h.Get(EmOneway)
	if oneway != "" {
		req.SetOneway(true)
	}

	st := h.Get(EmSerializeType)
	if st != "" {
		rst, err := strconv.Atoi(st)
		if err != nil {
			return nil, err
		}
		req.SetSerializeType(protocol.SerializeType(rst))
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

	sp := h.Get(EmServicePath)
	if sp != "" {
		req.ServicePath = sp
	}

	sm := h.Get(EmServiceMethod)
	if sm != "" {
		req.ServiceMethod = sm
	}

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	req.Payload = payload

	return req, nil
}
