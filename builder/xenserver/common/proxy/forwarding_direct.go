package proxy

type noNatProxyForwarding struct {
	TargetHost string
	TargetPort int
}

func (self *noNatProxyForwarding) GetServiceHost() string {
	return self.TargetHost
}

func (self *noNatProxyForwarding) GetServicePort() int {
	return self.TargetPort
}

func (self *noNatProxyForwarding) Close() error {
	return nil
}

func (self *noNatProxyForwarding) Start() error {
	return nil
}
