package client

import (
	"math/rand"
	"sync"
	"time"

	"github.com/easymicro/discovery"
	"github.com/easymicro/log"
)

type Selector interface {
	Pick([]*discovery.ServiceInfo) *discovery.ServiceInfo
}

type RoundRobinSelector struct {
	mu        sync.Mutex
	LastIndex int64
}

func NewRoundRobinSelector() *RoundRobinSelector {
	return &RoundRobinSelector{}
}

func (s *RoundRobinSelector) Pick(nodes []*discovery.ServiceInfo) *discovery.ServiceInfo {
	s.mu.Lock()
	s.LastIndex = (s.LastIndex + 1) % int64(len(nodes))
	log.Infof("LastIndex %d-------------", s.LastIndex)
	node := nodes[s.LastIndex]
	s.mu.Unlock()
	return node
}

type RandomSelector struct {
}

func NewRandomSelector() *RandomSelector {
	return &RandomSelector{}
}

func (s *RandomSelector) Pick(nodes []*discovery.ServiceInfo) *discovery.ServiceInfo {
	rand.Seed(int64(time.Now().UnixNano()))
	return nodes[rand.Intn(len(nodes))]

}
