package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/easymicro/discovery"
	"github.com/easymicro/log"
	"github.com/easymicro/protocol"
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
		servicePath:  servicePath,
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
}

func (client *Client) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, options ...BeforeOrAfterCallOption) error {
	rpcClient, err := client.selectRPCClient()
	if err != nil {
		log.Errorf("select rpc client failed for err %v", err)
		return nil
	}
	return rpcClient.Call(ctx, serviceMethod, args, reply, options...)
}

func (client *Client) Go(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, done chan *Call, options ...BeforeOrAfterCallOption) *Call {
	rpcClient, err := client.selectRPCClient()
	if err != nil {
		log.Errorf("select rpc client failed for err %v", err)
		return nil
	}
	return rpcClient.Go(ctx, serviceMethod, args, reply, done, options...)
}

func (client *Client) selectRPCClient() (*RPCClient, error) {
	var rpcClient *RPCClient
	nodes := client.discovery.GetAllNodes()
	if len(nodes) <= 0 {
		return nil, fmt.Errorf("no avaliable worker nodes")
	}
	selectedNode := client.selector.Pick(nodes)
	if selectedNode == nil {
		return nil, fmt.Errorf("not avaliable worker node")
	}
	log.Infof("selected node %+v for call..............", selectedNode)
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
