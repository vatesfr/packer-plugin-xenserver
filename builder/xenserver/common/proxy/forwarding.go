package proxy

type ProxyForwarding interface {
	GetServiceHost() string
	GetServicePort() int
	Start() error
	Close() error
}
