package discovery

type Discovery interface {
	Register(*ServiceInfo)
}

//the detail of service
type ServiceInfo struct {
	Name string
	Addr string
}
