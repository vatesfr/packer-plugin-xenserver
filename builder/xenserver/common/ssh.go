package common

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/proxy"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func doExecuteSSHCmd(cmd string, client *ssh.Client) (stdout string, err error) {
	//Create session
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}

	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run(cmd); err != nil {
		return "", err
	}

	return strings.Trim(b.String(), "\n"), nil
}

func ExecuteApiHostSSHCmd(state multistep.StateBag, cmd string) (stdout string, err error) {
	config := state.Get("commonconfig").(CommonConfig)

	sshClient, err := connectSSH(config.HostIp, config.HostSSHPort, config.Username, config.Password)
	if err != nil {
		return "", fmt.Errorf("Could not connect to ssh")
	}

	defer sshClient.Close()

	return doExecuteSSHCmd(cmd, sshClient)
}

func ExecuteHostSSHCmd(state multistep.StateBag, cmd string) (stdout string, err error) {
	config := state.Get("commonconfig").(CommonConfig)
	xenProxy := state.Get("xen_proxy").(proxy.XenProxy)

	host := state.Get("vm_host_address").(string)

	sshClient, err := ConnectSSHWithProxy(xenProxy, host, 22, config.Username, config.Password)
	if err != nil {
		return "", fmt.Errorf("could not connect to ssh proxy: %w", err)
	}

	defer sshClient.Close()

	return doExecuteSSHCmd(cmd, sshClient)
}

func connectSSH(host string, port int, username string, password string) (*ssh.Client, error) {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	log.Printf("Connecting with ssh to %s", address)

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("could not connect to ssh server: %w", err)
	}
	return client, err
}

func ConnectSSHWithProxy(proxy proxy.XenProxy, host string, port int, username string, password string) (*ssh.Client, error) {
	connection, err := proxy.Connect(host, port)

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

	return sshClient, nil
}

func sshDialer(client *ssh.Client) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return client.Dial("tcp", addr)
	}
}
