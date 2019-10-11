package server

import "github.com/easymicro/discovery"

type ServerOption func(*Server) error

func SetEtcdDiscovery(endpoints []string) ServerOption {
	return func(s *Server) error {
		dis, err := discovery.NewEtcdDiscovery(endpoints)
		if err != nil {
			return err
		}
		s.discovery = dis
		return nil
	}
}
