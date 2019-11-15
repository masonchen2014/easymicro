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
