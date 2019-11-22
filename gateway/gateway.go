package gateway

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/julienschmidt/httprouter"
	dis "github.com/masonchen2014/easymicro/discovery"
	"github.com/masonchen2014/easymicro/log"
	"github.com/masonchen2014/easymicro/protocol"
	"github.com/masonchen2014/easymicro/server"
	"github.com/masonchen2014/easymicro/share"
)

type Gateway struct {
	addr          string
	discoveryEnds []string
	rwmutex       sync.RWMutex
	clientMap     map[string]*Client
}

func NewGateway(laddr string, ends []string) *Gateway {
	return &Gateway{
		addr:          laddr,
		discoveryEnds: ends,
		clientMap:     make(map[string]*Client),
	}
}

func (gw *Gateway) StartGateway() {
	router := httprouter.New()
	router.POST("/:servicePath/:serviceMethod", gw.handleRequest)
	router.GET("/:servicePath/:serviceMethod", gw.handleRequest)
	if err := http.ListenAndServe(gw.addr, router); err != nil {
		log.Panicf("error in gateway Serve: %s", err)
	}
}

func (gw *Gateway) GetClient(service string) (*Client, error) {
	var c *Client
	gw.rwmutex.RLock()
	c = gw.clientMap[service]
	gw.rwmutex.RUnlock()

	if c == nil {
		client, err := NewDiscoveryClient(service, dis.NewEtcdDiscoveryMaster(gw.discoveryEnds, service))
		if err != nil {
			return nil, err
		}
		c = client
		gw.rwmutex.Lock()
		gw.clientMap[service] = c
		gw.rwmutex.Unlock()
	}

	return c, nil
}

func (gw *Gateway) handleRequest(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
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

		req, err = server.HTTPRequest2EmRpcRequest(r)
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

	resMetadata := make(map[string]string)
	client, err := gw.GetClient(req.ServicePath)
	if err != nil {
		log.Warnf("easymicro: failed to get %s client: %v", req.ServicePath, err)
		wh.Set(share.EmMessageStatusType, "Error")
		wh.Set(share.EmErrorMessage, err.Error())
		w.WriteHeader(500)
		return
	}

	res, err := client.Call(context.Background(), req.ServicePath, req)
	defer protocol.FreeMsg(res)

	if err != nil {
		log.Warnf("easymicro: failed to handle gateway request: %v", err)
		wh.Set(share.EmMessageStatusType, "Error")
		wh.Set(share.EmErrorMessage, err.Error())
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
	wh.Set(share.EmMeta, meta.Encode())
	w.Write(res.Payload)
}
