package discovery

func NewDirectDiscoveryMaster(endpoints []string) (*DirectDiscoveryMaster, error) {
	master := &DirectDiscoveryMaster{
		Nodes: make(map[string]struct{}),
	}
	for _, endpoint := range endpoints {
		master.Nodes[endpoint] = struct{}{}
	}
	return master, nil
}

type DirectDiscoveryMaster struct {
	Nodes map[string]struct{}
}

func (m *DirectDiscoveryMaster) GetAllNodes() []string {
	addrs := []string{}
	for k, _ := range m.Nodes {
		addrs = append(addrs, k)
	}
	return addrs
}
