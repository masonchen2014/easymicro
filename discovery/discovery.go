package discovery

type Discovery interface {
	Register(*ServiceInfo)
}

type DiscoveryMaster interface {
	GetAllNodes() []*ServiceInfo
}

//the detail of service
type ServiceInfo struct {
	Name    string
	Addr    string
	Network string
}
