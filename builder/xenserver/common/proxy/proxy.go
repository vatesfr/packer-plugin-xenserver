package proxy

import (
	"context"
	"net"
)

type XenProxy interface {
	Close() error
	Start() error
	ConnectWithAddr(address string) (net.Conn, error)
	Connect(host string, port int) (net.Conn, error)
	CreateForwarding(host string, port int) ProxyForwarding
	CreateWrapperForwarding(host string, port int, wrapper func(rawConn net.Conn) (net.Conn, error)) ProxyForwarding
}

func CreateProxy(skipNatForwarding bool, dialer func(ctx context.Context, network, addr string) (net.Conn, error)) XenProxy {
	if skipNatForwarding {
		return &noNatXenProxy{}
	}

	return &realXenProxy{
		Dialer: dialer,
	}
}
