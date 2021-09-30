package proxy

import (
	"context"
	"fmt"
	"github.com/armon/go-socks5"
	proxyClient "golang.org/x/net/proxy"
	"log"
	"net"
	"strconv"
)

type realXenProxy struct {
	Dialer func(ctx context.Context, network, addr string) (net.Conn, error)

	socksListener net.Listener
}

func (proxy *realXenProxy) Addr() string {
	return proxy.socksListener.Addr().String()
}

func (proxy *realXenProxy) Close() error {
	if proxy.socksListener != nil {
		return proxy.socksListener.Close()
	}
	return nil
}

func (proxy *realXenProxy) Start() error {
	socksConfig := &socks5.Config{
		Dial: proxy.Dialer,
	}
	server, err := socks5.New(socksConfig)
	if err != nil {
		return fmt.Errorf("could not setup socks server: %w", err)
	}

	proxy.socksListener, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("error creating socks listener: %s", err)
	}

	go func() {
		err := server.Serve(proxy.socksListener)
		if err != nil {
			log.Printf("error in proxy server: %v", err)
		}
	}()

	return nil
}

func (proxy *realXenProxy) ConnectWithAddr(address string) (net.Conn, error) {
	dialer, err := proxyClient.SOCKS5("tcp", proxy.Addr(), nil, proxyClient.Direct)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to proxy: %s", err)
	}

	c, err := dialer.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (proxy *realXenProxy) Connect(host string, port int) (net.Conn, error) {
	return proxy.ConnectWithAddr(net.JoinHostPort(host, strconv.Itoa(port)))
}

func (proxy *realXenProxy) CreateForwarding(host string, port int) ProxyForwarding {
	return &realProxyForwarding{
		Proxy:      proxy,
		TargetHost: host,
		TargetPort: port,
	}
}

func (proxy *realXenProxy) CreateWrapperForwarding(host string, port int,
	wrapper func(rawConn net.Conn) (net.Conn, error)) ProxyForwarding {
	return &realProxyForwarding{
		Proxy:             proxy,
		TargetHost:        host,
		TargetPort:        port,
		ConnectionWrapper: wrapper,
	}
}
