package common

import (
	"bytes"
	"context"
	"fmt"
	"github.com/armon/go-socks5"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"strings"
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

	proxyAddress, err := GetXenProxyAddress(state)

	if err != nil {
		return "", err
	}

	host := state.Get("vm_host_address").(string)

	sshClient, err := ConnectSSHWithProxy(proxyAddress, host, 22, config.Username, config.Password)
	if err != nil {
		return "", fmt.Errorf("Could not connect to ssh proxy")
	}

	defer sshClient.Close()

	return doExecuteSSHCmd(cmd, sshClient.Client)
}

func connectSSH(host string, port int, username string, password string) (*ssh.Client, error) {
	log.Printf("Connecting with ssh to %s:%d", host, port)

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		return nil, fmt.Errorf("could not connect to ssh server: %w", err)
	}
	return client, err
}

func sshDialer(client *ssh.Client) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return client.Dial("tcp", addr)
	}
}

func setupProxyServer(dialer func(ctx context.Context, network, addr string) (net.Conn, error)) (*socks5.Server, error) {
	socksConfig := &socks5.Config{
		Dial: dialer,
	}
	server, err := socks5.New(socksConfig)
	if err != nil {
		return nil, fmt.Errorf("could not setup socks server: %w", err)
	}

	return server, nil
}
