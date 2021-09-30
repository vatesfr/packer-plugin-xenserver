package common

import (
	"errors"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"time"
)

type ProxyWrapper struct {
	Client *ssh.Client

	rawConnection net.Conn
	sshConnection ssh.Conn
}

func (w ProxyWrapper) Close() {
	if w.Client != nil {
		w.Client.Close()
	}

	if w.sshConnection != nil {
		w.sshConnection.Close()
	}

	if w.rawConnection != nil {
		w.rawConnection.Close()
	}
}

func GetXenProxyAddress(state multistep.StateBag) (string, error) {
	proxyAddress, ok := state.GetOk("xen_proxy_address")

	if !ok {
		return "", errors.New("Proxy address not set. Did the create proxy step run?")
	}

	return proxyAddress.(string), nil
}

func ConnectViaXenProxy(state multistep.StateBag, address string) (net.Conn, error) {
	proxyAddress, err := GetXenProxyAddress(state)

	if err != nil {
		return nil, err
	}

	return ConnectViaProxy(proxyAddress, address)
}

func ConnectViaProxy(proxyAddress, address string) (net.Conn, error) {
	dialer, err := proxy.SOCKS5("tcp", proxyAddress, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to proxy: %s", err)
	}

	c, err := dialer.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func ConnectSSHWithProxy(proxyAddress, host string, port int, username string, password string) (*ProxyWrapper, error) {
	connection, err := ConnectViaProxy(proxyAddress, fmt.Sprintf("%s:%d", host, port))

	if err != nil {
		return nil, fmt.Errorf("could not connect to target server: %w", err)
	}

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	connectionEstablished := make(chan struct{}, 1)
	var sshConn ssh.Conn
	var sshChan <-chan ssh.NewChannel
	var req <-chan *ssh.Request

	go func() {
		sshConn, sshChan, req, err = ssh.NewClientConn(connection, host, config)
		close(connectionEstablished)
	}()

	timeout := time.Minute

	select {
	case <-connectionEstablished:
		// We don't need to do anything here. We just want select to block until
		// we connect or timeout.
	case <-time.After(timeout):
		if sshConn != nil {
			sshConn.Close()
		}

		if connection != nil {
			connection.Close()
		}

		return nil, fmt.Errorf("timeout connecting to ssh server: %w", err)
	}

	sshClient := ssh.NewClient(sshConn, sshChan, req)

	wrapper := ProxyWrapper{
		Client:        sshClient,
		rawConnection: connection,
		sshConnection: sshConn,
	}

	return &wrapper, nil
}

func CreatePortForwarding(proxyAddress, targetAddress string) (net.Listener, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")

	if err != nil {
		return nil, fmt.Errorf("could not create port forward listener: %w", err)
	}

	go func() {
		for {
			accept, err := listener.Accept()
			if err != nil {
				fmt.Printf("error accepting: %v", err)
				continue
			}

			go handleConnection(proxyAddress, targetAddress, accept)
		}
	}()

	return listener, nil
}

func handleConnection(proxyAddress, targetAddress string, accept net.Conn) {
	defer accept.Close()
	conn, err := ConnectViaProxy(proxyAddress, targetAddress)

	if err != nil {
		fmt.Printf("[FORWARD] Connect proxy Error: %v", err)
		return
	}

	defer conn.Close()

	txDone := make(chan struct{})
	rxDone := make(chan struct{})

	go func() {
		_, err := io.Copy(conn, accept)

		// Close conn so that other copy operation unblocks
		conn.Close()
		close(txDone)

		if err != nil {
			fmt.Printf("[FORWARD] Error conn <- accept: %v", err)
			return
		}
	}()

	go func() {
		_, err := io.Copy(accept, conn)

		// Close accept so that other copy operation unblocks
		accept.Close()
		close(rxDone)

		if err != nil {
			fmt.Printf("[FORWARD] Error accept <- conn: %v", err)
			return
		}
	}()

	<-txDone
	<-rxDone
}
