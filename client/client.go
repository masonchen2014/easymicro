package client

import (
	"context"
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
		servicePath: servicePath,
		selectMode:  RoundRobin,
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
	if client.selectMode == SelectByUser {
		rpcClient = client.defaultRPCClient
		return rpcClient, nil
	}
	nodes := client.discovery.GetAllNodes()
	selectedNode := ""
	switch client.selectMode {
	case RandomSelect:
		selectedNode = nodes[0]

		//TODO
	case RoundRobin:
		//TODO
	}
	client.mu.RLock()
	rpcClient = client.cachedClient[selectedNode]
	client.mu.RUnlock()
	if rpcClient == nil {
		rCli, err := NewRPCClient("tcp", selectedNode, client.servicePath)
		if err != nil {
			return nil, err
		}
		rpcClient = rCli
		client.mu.Lock()
		client.cachedClient[selectedNode] = rpcClient
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
