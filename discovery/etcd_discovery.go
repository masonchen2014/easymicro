package discovery

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/masonchen2014/easymicro/log"

	"github.com/coreos/etcd/clientv3"
	"github.com/masonchen2014/easymicro/share"
)

type EtcdDiscovery struct {
	Info    *ServiceInfo
	stopCh  chan struct{}
	leaseid clientv3.LeaseID
	client  *clientv3.Client
}

func NewEtcdDiscovery(endpoints []string) (*EtcdDiscovery, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 2 * time.Second,
	})

	if err != nil {
		return nil, err
	}

	return &EtcdDiscovery{
		stopCh: make(chan struct{}),
		client: cli,
	}, err
}

func (s *EtcdDiscovery) Register(info *ServiceInfo) {
	s.Info = info
	for {
		ch, err := s.keepAlive()
		if err != nil {
			log.Errorf("etcd discovery watch failed for err %v", err)
			return
		}

	readChanges:
		for {
			select {
			case <-s.stopCh:
				log.Infof("etcd discovery revoke for server is closed")
				s.revoke()
				return

			case ka, ok := <-ch:
				if !ok {
					break readChanges
				} else {
					log.Infof("Recv reply from service: %s, ttl:%d", s.Info.Name, ka.TTL)
				}
			}
		}
		log.Infof("keepalive chan is closed and will rewatch")
	}
}

func (d *EtcdDiscovery) UnRegister() {
	close(d.stopCh)
}

func (s *EtcdDiscovery) keepAlive() (<-chan *clientv3.LeaseKeepAliveResponse, error) {

	info := s.Info

	key := share.BaseServicesPath + info.Name + "/" + s.Info.Addr
	value, _ := json.Marshal(info)
	log.Infof("etcd discovery set key %s value %s", key, value)
	// minimum lease TTL is 5-second
	resp, err := s.client.Grant(context.TODO(), 10)
	if err != nil {
		return nil, err
	}

	log.Infof("etcd discovery lease id %v", resp.ID)
	_, err = s.client.Put(context.TODO(), key, string(value), clientv3.WithLease(resp.ID))
	if err != nil {
		return nil, err
	}
	s.leaseid = resp.ID

	return s.client.KeepAlive(context.TODO(), resp.ID)
}

func (s *EtcdDiscovery) revoke() error {
	_, err := s.client.Revoke(context.TODO(), s.leaseid)
	return err
}

type EtcdDiscoveryMaster struct {
	Path   string
	Client *clientv3.Client
	mutex  sync.RWMutex
	Nodes  map[string]*Node
}

//node is a client
type Node struct {
	Key  string
	Info *ServiceInfo
}

func NewEtcdDiscoveryMaster(endpoints []string, watchPath string) *EtcdDiscoveryMaster {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: time.Second,
	})

	if err != nil {
		log.Panic(err)
	}

	master := &EtcdDiscoveryMaster{
		Path:   watchPath,
		Nodes:  make(map[string]*Node),
		Client: cli,
	}

	if err := master.FetchAllNodesInfo(); err != nil {
		log.Errorf("NewEtcdDiscoveryMaster GetAllNodes failed %v", err)
		return nil
	}

	go master.WatchNodes()
	log.Infof("etcd discovery master %+v", master)
	return master
}

func (m *EtcdDiscoveryMaster) GetAllNodes() []*ServiceInfo {
	nodes := []*ServiceInfo{}
	m.mutex.RLock()
	for _, node := range m.Nodes {
		log.Infof("EtcdDiscoveryMaster node %+v", node)
		nodes = append(nodes, node.Info)
	}
	m.mutex.RUnlock()
	return nodes
}

func (m *EtcdDiscoveryMaster) FetchAllNodesInfo() error {
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	resp, err := m.Client.Get(ctx, m.Path, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, kv := range resp.Kvs {
		info, err := ValueToServiceInfo(kv.Value)
		if err != nil {
			return err
		}
		m.AddNode(string(kv.Key), info)
	}

	return nil
}

func (m *EtcdDiscoveryMaster) AddNode(key string, info *ServiceInfo) {
	node := &Node{
		Key:  key,
		Info: info,
	}

	m.mutex.Lock()
	m.Nodes[node.Key] = node
	m.mutex.Unlock()
}

func (m *EtcdDiscoveryMaster) DelNode(key string) {
	m.mutex.Lock()
	delete(m.Nodes, string(key))
	m.mutex.Unlock()
}

func ValueToServiceInfo(value []byte) (*ServiceInfo, error) {
	info := &ServiceInfo{}
	err := json.Unmarshal(value, info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (m *EtcdDiscoveryMaster) WatchNodes() {
	//TODO 重试机制
	rch := m.Client.Watch(context.Background(), m.Path, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case clientv3.EventTypePut:
				log.Infof("[%s] %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
				info, err := ValueToServiceInfo(ev.Kv.Value)
				if err != nil {
					log.Errorf("EventToServiceInfo error %v for event %v", err, ev)
					continue
				}
				m.AddNode(string(ev.Kv.Key), info)
			case clientv3.EventTypeDelete:
				log.Infof("[%s] %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
				m.DelNode(string(ev.Kv.Key))
			}
		}
	}
	log.Infof("EtcdDiscoveryMaster watch nodes exit")
}
