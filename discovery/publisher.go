package discovery

import (
	"context"
	"encoding/json"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/masonchen2014/easymicro/log"
	"github.com/masonchen2014/easymicro/share"
)

type EtcdPublisher struct {
	Info    *ServiceInfo
	stopCh  chan struct{}
	leaseid clientv3.LeaseID
	client  *clientv3.Client
}

func NewEtcdPublisher(endpoints []string) (*EtcdPublisher, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 2 * time.Second,
	})

	if err != nil {
		return nil, err
	}

	return &EtcdPublisher{
		stopCh: make(chan struct{}),
		client: cli,
	}, err
}

func (s *EtcdPublisher) Register(info *ServiceInfo) {
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

func (d *EtcdPublisher) UnRegister() {
	close(d.stopCh)
}

func (s *EtcdPublisher) keepAlive() (<-chan *clientv3.LeaseKeepAliveResponse, error) {

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

func (s *EtcdPublisher) revoke() error {
	_, err := s.client.Revoke(context.TODO(), s.leaseid)
	return err
}
