package client

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/masonchen2014/easymicro/discovery"
	"github.com/masonchen2014/easymicro/log"
	"github.com/masonchen2014/easymicro/protocol"
	"github.com/sony/gobreaker"
)

type Client struct {
	network      string
	address      string
	selectMode   SelectMode
	selector     Selector
	mu           sync.RWMutex
	cachedClient []*RPCClient
	// cachedClient map[string]*RPCClient
	subscriber *discovery.EtcdSubscriber

	defaultRPCClient *RPCClient
	//	breakers     sync.Map
	servicePath string
	//	option       Option

	//	mu sync.RWMutex
	//	servers map[string]string
	// discovery discovery.DiscoveryMaster
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

	breakerConfig *gobreaker.Settings
	limiterConfig *LimiterConfig
}

type ClientConfig struct {
	Network     string
	Address     string
	ServicePath string
}

type DiscoverClientConfig struct {
	Endpoints   []string
	ServiceName string
}

type RPCClientConfig struct {
	network     string
	address     string
	servicePath string
	dialTimeout time.Duration
}

func NewClient(conf *ClientConfig, opts ...ClientOption) (*Client, error) {
	c := &Client{
		servicePath: conf.ServicePath,
		network:     conf.Network,
		address:     conf.Address,
		selectMode:  SelectByUser,
	}
	for _, opt := range opts {
		opt(c)
	}
	rpcClient, err := NewRPCClient(&RPCClientConfig{
		network:     c.network,
		address:     c.address,
		servicePath: c.servicePath,
		dialTimeout: c.DialTimeout,
	})
	if err != nil {
		return nil, err
	}
	rpcClient.SetCircuitBreaker(c.breakerConfig)
	rpcClient.SetRateLimiter(c.limiterConfig)
	c.defaultRPCClient = rpcClient

	return c, nil
}

func NewDiscoveryClient(conf *DiscoverClientConfig, opts ...ClientOption) (*Client, error) {
	client := &Client{
		servicePath:  strings.ToLower(conf.ServiceName),
		selectMode:   RoundRobin,
		selector:     NewRoundRobinSelector(),
		cachedClient: []*RPCClient{},
	}

	sub := discovery.NewEtcdSubscriber(
		&discovery.EtcdSubscriberConf{
			Endpoints:   conf.Endpoints,
			ServiceName: conf.ServiceName,
			AddNodeFunc: client.addNodeFunc(),
			DelNodeFunc: client.delNodeFunc(),
		},
	)

	client.subscriber = sub
	if sub == nil {
		log.Panicf("invalid service subscriber")
	}

	for _, opt := range opts {
		opt(client)
	}
	return client, nil
}

type LimiterConfig struct {
	FillInterval time.Duration
	Capacity     int64
	Quantum      int64
}

func (c *Client) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, options ...BeforeOrAfterCallOption) error {
	rpcClient, err := c.selectRPCClient()
	if err != nil {
		log.Errorf("select rpc client failed for err %v", err)
		return nil
	}
	return rpcClient.Call(ctx, strings.ToLower(serviceMethod), args, reply, options...)
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
	if c.subscriber == nil {
		return c.defaultRPCClient, nil
	}
	rpcClient := c.selector.Pick(c.cachedClient)
	if rpcClient == nil {
		return nil, fmt.Errorf("not avaliable worker node")
	}
	log.Debugf("selectRPCClient choose client ***************** %s", rpcClient.address)
	return rpcClient, nil
}

func (c *Client) addNodeFunc() func(*discovery.ServiceInfo) error {
	return func(info *discovery.ServiceInfo) error {
		newNode := true
		c.mu.Lock()
		for _, rCli := range c.cachedClient {
			if rCli.servicePath == info.Name && rCli.address == info.Addr {
				newNode = false
				break
			}
		}
		c.mu.Unlock()
		if newNode {
			rCli, err := NewRPCClient(&RPCClientConfig{
				network:     info.Network,
				address:     info.Addr,
				servicePath: info.Name,
				dialTimeout: c.DialTimeout,
			})
			if err != nil {
				return err
			}
			rCli.SetCircuitBreaker(c.breakerConfig)
			rCli.SetRateLimiter(c.limiterConfig)
			c.mu.Lock()
			c.cachedClient = append(c.cachedClient, rCli)
			log.Infof("addNodeFunc cachedClient %+v", c.cachedClient)
			c.mu.Unlock()
		}
		return nil
	}
}

func (c *Client) delNodeFunc() func(*discovery.ServiceInfo) error {
	return func(info *discovery.ServiceInfo) error {
		c.mu.Lock()
		for i, rCli := range c.cachedClient {
			if rCli.servicePath == info.Name && rCli.address == info.Addr {
				if len(c.cachedClient) == i+1 {
					c.cachedClient = c.cachedClient[:i]
				} else {
					c.cachedClient = append(c.cachedClient[:i], c.cachedClient[i+1:]...)
				}

			}
		}
		c.mu.Unlock()
		return nil
	}
}

func (c *Client) Close() {
	if c.defaultRPCClient != nil {
		c.defaultRPCClient.Close()
	}

	for _, rCli := range c.cachedClient {
		rCli.Close()
	}
}
