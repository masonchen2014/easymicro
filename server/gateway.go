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
		version := r.Header.Get(share.EmVersion)
		wh.Set(share.EmVersion, version)

		messageId := r.Header.Get(share.EmMessageID)
		wh.Set(share.EmMessageID, messageId)

		servicePath := params.ByName("servicePath")
		if servicePath == "" {
			err = errors.New("empty servicepath")
			break
		}

		wh.Set(share.EmServicePath, servicePath)
		serviceMethod := params.ByName("serviceMethod")
		if serviceMethod == "" {
			err = errors.New("empty servicemethod")
			break
		}
		wh.Set(share.EmServiceMethod, serviceMethod)

		serializedType := r.Header.Get(share.EmSerializeType)
		if serializedType == "" {
			err = errors.New("empty serialized type")
			break
		}
		serializedTypeInt, e := strconv.Atoi(serializedType)
		if e != nil {
			err = errors.New("invalid serialized type")
			break
		}
		wh.Set(share.EmSerializeType, r.Header.Get(share.EmSerializeType))

		messageIdInt, e := strconv.ParseUint(messageId, 10, 64)
		if e != nil {
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

		wh.Set(share.EmMessageStatusType, "Error")
		wh.Set(share.EmErrorMessage, err.Error())
		w.WriteHeader(400)
		return
	}

	newCtx, err := extractClientMdContexFromMd(context.Background(), req.Metadata)
	res, err := s.handleRequest(newCtx, req)
	defer protocol.FreeMsg(res)

	if err != nil {
		log.Warnf("easymicro: failed to handle gateway request: %v", err)
		wh.Set(share.EmMessageStatusType, "Error")
		wh.Set(share.EmErrorMessage, err.Error())
		w.WriteHeader(500)
		return
	}

	if res.Metadata == nil {
		res.Metadata = make(map[string]string)
	}

	meta := url.Values{}
	for k, v := range res.Metadata {
		meta.Add(k, v)
	}
	wh.Set(share.EmMeta, meta.Encode())
	w.Write(res.Payload)
}

func HTTPRequest2EmRpcRequest(r *http.Request) (*protocol.Message, error) {
	req := protocol.GetPooledMsg()
	req.SetMessageType(protocol.Request)

	h := r.Header
	heartbeat := h.Get(share.EmHeartbeat)
	if heartbeat != "" {
		req.SetHeartbeat(true)
	}

	oneway := h.Get(share.EmOneway)
	if oneway != "" {
		req.SetOneway(true)
	}

	meta := h.Get(share.EmMeta)
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
