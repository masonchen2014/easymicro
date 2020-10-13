package server

import "github.com/masonchen2014/easymicro/discovery"

type ServerOption func(*Server) error

func SetEtcdDiscovery(endpoints []string, advertiseUrl string) ServerOption {
	return func(s *Server) error {
		dis, err := discovery.NewEtcdDiscovery(endpoints)
		if err != nil {
			return err
		}
		s.discovery = dis
		s.advertiseUrl = advertiseUrl
		return nil
	}
}

func SetBasicOption(opt Option) ServerOption {
	return func(s *Server) error {
		if opt.MaxConnIdleTime > 0 {
			s.maxConnIdleTime = opt.MaxConnIdleTime
		}
		return nil
	}
}

func SetGateWayMode() ServerOption {
	return func(s *Server) error {
		s.useGateWay = true
		return nil
	}
}

type Option struct {
	MaxConnIdleTime int64
	JobChanSize     int64
	WorkerNum       int64
}

var defaultOption = &Option{
	MaxConnIdleTime: 10,
	JobChanSize:     50000,
	WorkerNum:       1000,
}
