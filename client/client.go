package client

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/juju/ratelimit"
	"github.com/masonchen2014/easymicro/discovery"
	"github.com/masonchen2014/easymicro/log"
	"github.com/masonchen2014/easymicro/protocol"
	"github.com/sony/gobreaker"
)

func NewClient(network, address, servicePath string, opts ...ClientOption) (*Client, error) {
	client := &Client{
		servicePath: servicePath,
		selectMode:  SelectByUser,
	}
	rpcClient, err := NewRPCClient(network, address, servicePath)
	if err != nil {
		return nil, err
	}
	client.defaultRPCClient = rpcClient
	for _, opt := range opts {
		opt(client)
	}
	return client, nil
}

func NewDiscoveryClient(servicePath string, dis discovery.DiscoveryMaster, opts ...ClientOption) (*Client, error) {
	client := &Client{
		servicePath:  strings.ToLower(servicePath),
		selectMode:   RoundRobin,
		selector:     NewRoundRobinSelector(),
		cachedClient: make(map[string]*RPCClient),
		discovery:    dis,
	}

	if dis == nil {
		log.Panicf("invalid discovery master")
	}

	for _, opt := range opts {
		opt(client)
	}
	return client, nil
}

type Client struct {
	//failMode     FailMode
	selectMode   SelectMode
	selector     Selector
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

func (c *Client) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, options ...BeforeOrAfterCallOption) error {
	if c.bucket != nil {
		c.bucket.Wait(1)
	}
	if c.breaker != nil {
		_, err := c.breaker.Execute(func() (interface{}, error) {
			rpcClient, err := c.selectRPCClient()
			if err != nil {
				log.Errorf("select rpc client failed for err %v", err)
				return nil, err
			}
			return nil, rpcClient.Call(ctx, strings.ToLower(serviceMethod), args, reply, options...)
		})
		return err
	} else {
		rpcClient, err := c.selectRPCClient()
		if err != nil {
			log.Errorf("select rpc client failed for err %v", err)
			return nil
		}
		return rpcClient.Call(ctx, strings.ToLower(serviceMethod), args, reply, options...)
	}
}

func (c *Client) Go(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, done chan *Call, options ...BeforeOrAfterCallOption) *Call {
	rpcClient, err := c.selectRPCClient()
	if err != nil {
		log.Errorf("select rpc client failed for err %v", err)
		return nil
	}
	return rpcClient.Go(ctx, serviceMethod, args, reply, done, options...)
}

func (c *Client) selectRPCClient() (*RPCClient, error) {
	if c.discovery == nil {
		return c.defaultRPCClient, nil
	}
	var rpcClient *RPCClient
	nodes := c.discovery.GetAllNodes()
	if len(nodes) <= 0 {
		return nil, fmt.Errorf("no avaliable worker nodes")
	}
	selectedNode := c.selector.Pick(nodes)
	if selectedNode == nil {
		return nil, fmt.Errorf("not avaliable worker node")
	}
	c.mu.RLock()
	rpcClient = c.cachedClient[selectedNode.Addr]
	c.mu.RUnlock()
	if rpcClient == nil || rpcClient.status == ConnClose || rpcClient.status == ConnReconnectFail {
		rCli, err := NewRPCClient(selectedNode.Network, selectedNode.Addr, c.servicePath)
		if err != nil {
			return nil, err
		}
		rpcClient = rCli
		c.mu.Lock()
		c.cachedClient[selectedNode.Addr] = rpcClient
		c.mu.Unlock()
	}
	return rpcClient, nil
}

func (c *Client) Close() {
	if c.defaultRPCClient != nil {
		c.defaultRPCClient.Close()
	}

	for _, rCli := range c.cachedClient {
		rCli.Close()
	}
}
