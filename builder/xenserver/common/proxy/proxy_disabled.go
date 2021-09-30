package proxy

import (
	"net"
	"strconv"
)

type noNatXenProxy struct{}

func (proxy *noNatXenProxy) Close() error {
	return nil
}

func (proxy *noNatXenProxy) Start() error {
	return nil
}

func (proxy *noNatXenProxy) ConnectWithAddr(address string) (net.Conn, error) {
	c, err := net.Dial("tcp", address)

	if err != nil {
		return nil, err
	}

	return c, nil
}

func (proxy *noNatXenProxy) Connect(host string, port int) (net.Conn, error) {
	return proxy.ConnectWithAddr(net.JoinHostPort(host, strconv.Itoa(port)))
}

func (proxy *noNatXenProxy) CreateForwarding(host string, port int) ProxyForwarding {
	return &noNatProxyForwarding{
		TargetHost: host,
		TargetPort: port,
	}
}

func (proxy *noNatXenProxy) CreateWrapperForwarding(host string, port int,
	_ func(rawConn net.Conn) (net.Conn, error)) ProxyForwarding {
	return &noNatProxyForwarding{
		TargetHost: host,
		TargetPort: port,
	}
}
