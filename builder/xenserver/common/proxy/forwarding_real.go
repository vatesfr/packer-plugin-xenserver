package proxy

import (
	"fmt"
	"io"
	"log"
	"net"
)

type realProxyForwarding struct {
	Proxy             XenProxy
	TargetHost        string
	TargetPort        int
	ConnectionWrapper func(net.Conn) (net.Conn, error)

	listener net.Listener
}

func (self *realProxyForwarding) GetServiceHost() string {
	return self.listener.Addr().(*net.TCPAddr).IP.String()
}

func (self *realProxyForwarding) GetServicePort() int {
	return self.listener.Addr().(*net.TCPAddr).Port
}

func (self *realProxyForwarding) Close() error {
	if self.listener != nil {
		return self.listener.Close()
	}
	return nil
}

func identityWrapper(conn net.Conn) (net.Conn, error) {
	return conn, nil
}

func (self *realProxyForwarding) Start() error {
	if self.ConnectionWrapper == nil {
		self.ConnectionWrapper = identityWrapper
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")

	if err != nil {
		return fmt.Errorf("could not create port forward listener: %w", err)
	}

	go func() {
		for {
			accept, err := listener.Accept()
			if err != nil {
				return
			}

			go self.handleConnection(accept)
		}
	}()

	return nil
}

func (self *realProxyForwarding) serviceForwardedConnection(clientConn net.Conn, targetConn net.Conn) {
	txDone := make(chan struct{})
	rxDone := make(chan struct{})

	go func() {
		_, err := io.Copy(targetConn, clientConn)

		log.Printf("[FORWARD] proxy client closed connection")

		// Close conn so that other copy operation unblocks
		targetConn.Close()
		close(txDone)

		if err != nil {
			log.Printf("[FORWARD] Error conn <- accept: %v", err)
			return
		}
	}()

	go func() {
		_, err := io.Copy(clientConn, targetConn)

		log.Printf("[FORWARD] proxy target closed connection")

		// Close accept so that other copy operation unblocks
		clientConn.Close()
		close(rxDone)

		if err != nil {
			log.Printf("[FORWARD] Error accept <- conn: %v", err)
			return
		}
	}()

	<-txDone
	<-rxDone
}

func (self *realProxyForwarding) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	rawTargetConnection, err := self.Proxy.Connect(self.TargetHost, self.TargetPort)
	if err != nil {
		log.Printf("[FORWARD] Connect proxy Error: %v", err)
		return
	}
	defer rawTargetConnection.Close()

	targetConn, err := self.ConnectionWrapper(rawTargetConnection)

	if err != nil {
		log.Printf("[FORWARD] wrap proxy connection error: %v", err)
		return
	}

	defer targetConn.Close()

	self.serviceForwardedConnection(clientConn, targetConn)
}
