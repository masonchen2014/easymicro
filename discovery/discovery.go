package discovery

type Discovery interface {
	Register(*ServiceInfo)
}

type DiscoveryMaster interface {
	GetAllNodes() []string
}

//the detail of service
type ServiceInfo struct {
	Name    string
	Addr    string
	Network string
}
