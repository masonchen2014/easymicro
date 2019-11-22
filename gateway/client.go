package gateway

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/juju/ratelimit"
	"github.com/masonchen2014/easymicro/client"
	"github.com/masonchen2014/easymicro/discovery"
	"github.com/masonchen2014/easymicro/log"
	"github.com/masonchen2014/easymicro/protocol"
	"github.com/sony/gobreaker"
)

func NewClient(network, address, servicePath string) (*Client, error) {
	client := &Client{
		servicePath: servicePath,
		selectMode:  client.SelectByUser,
	}
	rpcClient, err := NewRPCClient(network, address, servicePath)
	if err != nil {
		return nil, err
	}
	client.defaultRPCClient = rpcClient
	return client, nil
}

func NewDiscoveryClient(servicePath string, dis discovery.DiscoveryMaster) (*Client, error) {
	client := &Client{
		servicePath:  strings.ToLower(servicePath),
		selectMode:   client.RoundRobin,
		selector:     client.NewRoundRobinSelector(),
		cachedClient: make(map[string]*RPCClient),
		discovery:    dis,
	}

	if dis == nil {
		log.Panicf("invalid discovery master")
	}
	return client, nil
}

type Client struct {
	//failMode     FailMode
	selectMode   client.SelectMode
	selector     client.Selector
	mu           sync.RWMutex
	cachedClient map[string]*RPCClient

	defaultRPCClient *RPCClient
	//	breakers     sync.Map
	servicePath string
	//	option       Option

	//	mu sync.RWMutex
	//	servers map[string]string
	discovery discovery.DiscoveryMaster
	//	selector  Selector

	isShutdown bool

	// auth is a string for Authentication, for example, "Bearer mF_9.B5f-4.1JqM"
	auth string

	SerializeType     protocol.SerializeType
	CompressType      protocol.CompressType
	ReconnectTryNums  int
	HeartBeatTryNums  int
	HeartBeatTimeout  int
	HeartBeatInterval int64

	breaker *gobreaker.CircuitBreaker
	bucket  *ratelimit.Bucket
}

func (client *Client) Call(ctx context.Context, serviceMethod string, req *protocol.Message) (*protocol.Message, error) {
	if client.bucket != nil {
		client.bucket.Wait(1)
	}
	if client.breaker != nil {
		_, err := client.breaker.Execute(func() (interface{}, error) {
			rpcClient, err := client.selectRPCClient()
			if err != nil {
				log.Errorf("select rpc client failed for err %v", err)
				return nil, err
			}
			return rpcClient.Call(ctx, strings.ToLower(serviceMethod), req)
		})
		return nil, err
	} else {
		rpcClient, err := client.selectRPCClient()
		if err != nil {
			log.Errorf("select rpc client failed for err %v", err)
			return nil, nil
		}
		return rpcClient.Call(ctx, strings.ToLower(serviceMethod), req)
	}
}

func (client *Client) Go(ctx context.Context, serviceMethod string, req *protocol.Message, done chan *Call) *Call {
	rpcClient, err := client.selectRPCClient()
	if err != nil {
		log.Errorf("select rpc client failed for err %v", err)
		return nil
	}
	return rpcClient.Go(ctx, serviceMethod, req, done)
}

func (client *Client) selectRPCClient() (*RPCClient, error) {
	if client.discovery == nil {
		return client.defaultRPCClient, nil
	}
	var rpcClient *RPCClient
	nodes := client.discovery.GetAllNodes()
	if len(nodes) <= 0 {
		return nil, fmt.Errorf("no avaliable worker nodes")
	}
	selectedNode := client.selector.Pick(nodes)
	if selectedNode == nil {
		return nil, fmt.Errorf("not avaliable worker node")
	}
	client.mu.RLock()
	rpcClient = client.cachedClient[selectedNode.Addr]
	client.mu.RUnlock()
	if rpcClient == nil {
		rCli, err := NewRPCClient(selectedNode.Network, selectedNode.Addr, client.servicePath)
		if err != nil {
			return nil, err
		}
		rpcClient = rCli
		client.mu.Lock()
		client.cachedClient[selectedNode.Addr] = rpcClient
		client.mu.Unlock()
	}
	return rpcClient, nil
}

func (client *Client) Close() {
	if client.defaultRPCClient != nil {
		client.defaultRPCClient.Close()
	}

	for _, rCli := range client.cachedClient {
		rCli.Close()
	}
}
