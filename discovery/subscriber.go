package discovery

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/masonchen2014/easymicro/log"
	"github.com/masonchen2014/easymicro/share"
)

type EtcdSubscriber struct {
	Path        string
	Client      *clientv3.Client
	addNodeFunc func(*ServiceInfo) error
	delNodeFunc func(*ServiceInfo) error
}

type EtcdSubscriberConf struct {
	Endpoints   []string
	ServiceName string
	AddNodeFunc func(*ServiceInfo) error
	DelNodeFunc func(*ServiceInfo) error
}

func NewEtcdSubscriber(conf *EtcdSubscriberConf) *EtcdSubscriber {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   conf.Endpoints,
		DialTimeout: time.Second,
	})

	if err != nil {
		log.Panic(err)
	}

	sub := &EtcdSubscriber{
		Path:        strings.ToLower(share.BaseServicesPath + conf.ServiceName + "/"),
		Client:      cli,
		addNodeFunc: conf.AddNodeFunc,
		delNodeFunc: conf.DelNodeFunc,
	}

	if err := sub.FetchAllNodesInfo(); err != nil {
		log.Errorf("NewEtcdSubscriber GetAllNodes failed %v", err)
		return nil
	}

	go sub.WatchNodes()
	log.Infof("etcd subscriber %+v", sub)
	return sub
}

func (m *EtcdSubscriber) FetchAllNodesInfo() error {
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
		if err := m.addNodeFunc(info); err != nil {
			log.Errorf("addNodeFunc error %+v for node %+v", info)
		}
	}

	return nil
}

func (m *EtcdSubscriber) WatchNodes() {
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
				m.addNodeFunc(info)
			case clientv3.EventTypeDelete:
				log.Infof("[%s] %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
				info, err := ValueToServiceInfo(ev.Kv.Value)
				if err != nil {
					log.Errorf("EventToServiceInfo error %v for event %v", err, ev)
					continue
				}
				m.delNodeFunc(info)
			}
		}
	}
	log.Infof("EtcdSubscriber watch nodes exit")
}

func ValueToServiceInfo(value []byte) (*ServiceInfo, error) {
	info := &ServiceInfo{}
	err := json.Unmarshal(value, info)
	if err != nil {
		return nil, err
	}
	return info, nil
}
