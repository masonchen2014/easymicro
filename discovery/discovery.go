package discovery

//the detail of service
type ServiceInfo struct {
	Name    string
	Addr    string
	Network string
}
type Publisher interface {
	Register(*ServiceInfo)
	UnRegister()
}

type Subscriber interface {
	GetAllNodes() []*ServiceInfo
	Watch()
}
