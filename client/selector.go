package client

import (
	"sync"

	"github.com/masonchen2014/easymicro/log"
)

type Selector interface {
	Pick([]*RPCClient) *RPCClient
}

type RoundRobinSelector struct {
	mu        sync.Mutex
	LastIndex int64
}

func NewRoundRobinSelector() *RoundRobinSelector {
	return &RoundRobinSelector{}
}

func (s *RoundRobinSelector) Pick(clients []*RPCClient) *RPCClient {
	var rpcCli *RPCClient
	s.mu.Lock()
	s.LastIndex = (s.LastIndex + 1) % int64(len(clients))
	for i := 0; i < len(clients); i++ {
		rpcCli = clients[s.LastIndex]
		log.Infof("Pick rpcCli %+v", *rpcCli)
		if rpcCli.status != ConnAvailable {
			s.LastIndex = (s.LastIndex + 1) % int64(len(clients))
		}
	}
	s.mu.Unlock()
	if rpcCli.status != ConnAvailable {
		return nil
	}
	return rpcCli
}
