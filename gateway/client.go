package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/juju/ratelimit"
	"github.com/masonchen2014/easymicro/client"
	"github.com/masonchen2014/easymicro/discovery"
	"github.com/masonchen2014/easymicro/log"
	"github.com/masonchen2014/easymicro/protocol"
	"github.com/sony/gobreaker"
)

func NewClient(network, address, servicePath string) (*Client, error) {
	c := &Client{
		servicePath: servicePath,
		selectMode:  client.SelectByUser,
	}
	rpcClient, err := NewRPCClient(network, address, servicePath, c.DialTimeout)
	if err != nil {
		return nil, err
	}
	c.defaultRPCClient = rpcClient
	return c, nil
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
	DialTimeout       time.Duration

	breaker *gobreaker.CircuitBreaker
	bucket  *ratelimit.Bucket
}

func (c *Client) Call(ctx context.Context, serviceMethod string, req *protocol.Message) (*protocol.Message, error) {
	if c.bucket != nil {
		c.bucket.Wait(1)
	}
	if c.breaker != nil {
		reply, err := c.breaker.Execute(func() (interface{}, error) {
			rpcClient, err := c.selectRPCClient()
			if err != nil {
				log.Errorf("select rpc client failed for err %v", err)
				return nil, err
			}
			return rpcClient.Call(ctx, strings.ToLower(serviceMethod), req)
		})

		if err != nil {
			return nil, err
		}
		replyMsg, b := reply.(*protocol.Message)
		if !b {
			return nil, errors.New("invalid reply")
		}
		return replyMsg, nil
	} else {
		rpcClient, err := c.selectRPCClient()
		if err != nil {
			log.Errorf("select rpc client failed for err %v", err)
			return nil, err
		}
		return rpcClient.Call(ctx, strings.ToLower(serviceMethod), req)
	}
}

func (c *Client) Go(ctx context.Context, serviceMethod string, req *protocol.Message, done chan *Call) *Call {
	rpcClient, err := c.selectRPCClient()
	if err != nil {
		log.Errorf("select rpc client failed for err %v", err)
		return nil
	}
	return rpcClient.Go(ctx, serviceMethod, req, done)
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
	if rpcClient == nil || rpcClient.status == client.ConnClose || rpcClient.status == client.ConnReconnectFail {
		rCli, err := NewRPCClient(selectedNode.Network, selectedNode.Addr, c.servicePath, c.DialTimeout)
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

func (client *Client) Close() {
	if client.defaultRPCClient != nil {
		client.defaultRPCClient.Close()
	}

	for _, rCli := range client.cachedClient {
		rCli.Close()
	}
}
