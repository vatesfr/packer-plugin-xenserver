package common

import (
	"bytes"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	gossh "golang.org/x/crypto/ssh"
)

// SSHConfig contains SSH-specific configuration
type SSHConfig struct {
	Comm communicator.Config `mapstructure:",squash"`

	SSHHostPortMin    uint `mapstructure:"ssh_host_port_min"`
	SSHHostPortMax    uint `mapstructure:"ssh_host_port_max"`
	SSHSkipNatMapping bool `mapstructure:"ssh_skip_nat_mapping"`

	// These are deprecated, but we keep them around for BC
	// TODO(@mitchellh): remove
	SSHKeyPath     string        `mapstructure:"ssh_key_path"`
	SSHWaitTimeout time.Duration `mapstructure:"ssh_wait_timeout"`
}

// Prepare validates and prepares the SSH configuration
func (c *SSHConfig) Prepare(ctx *interpolate.Context) []error {
	if c.SSHHostPortMin == 0 {
		c.SSHHostPortMin = 2222
	}

	if c.SSHHostPortMax == 0 {
		c.SSHHostPortMax = 4444
	}

	// TODO: backwards compatibility, write fixer instead
	if c.SSHKeyPath != "" {
		c.Comm.SSHPrivateKeyFile = c.SSHKeyPath
	}
	if c.SSHWaitTimeout != 0 {
		c.Comm.SSHTimeout = c.SSHWaitTimeout
	}

	errs := c.Comm.Prepare(ctx)
	if c.SSHHostPortMin > c.SSHHostPortMax {
		errs = append(errs,
			errors.New("ssh_host_port_min must be less than ssh_host_port_max"))
	}

	return errs
}

// SSH Address Resolution Functions

// SSHAddress returns the SSH connection address for the instance
func SSHAddress(state multistep.StateBag) (string, error) {
	sshIP := state.Get("ssh_address").(string)
	sshHostPort := state.Get("ssh_port").(uint)
	return fmt.Sprintf("%s:%d", sshIP, sshHostPort), nil
}

// SSHLocalAddress returns the local SSH forwarding address
func SSHLocalAddress(state multistep.StateBag) (string, error) {
	sshLocalPort, ok := state.Get("local_ssh_port").(uint)
	if !ok {
		return "", fmt.Errorf("SSH port forwarding hasn't been set up yet")
	}
	return fmt.Sprintf("%s:%d", "127.0.0.1", sshLocalPort), nil
}

// SSHPort returns the SSH port for the instance
func SSHPort(state multistep.StateBag) (int, error) {
	sshHostPort := state.Get("local_ssh_port").(uint)
	return int(sshHostPort), nil
}

// CommHost returns the communication host address
func CommHost(state multistep.StateBag) (string, error) {
	return "127.0.0.1", nil
}

// SSH Configuration and Authentication

// SSHConfigFunc returns a function that creates an SSH client configuration
func SSHConfigFunc(config SSHConfig) func(multistep.StateBag) (*gossh.ClientConfig, error) {
	return func(state multistep.StateBag) (*gossh.ClientConfig, error) {
		config := state.Get("config").(Config)
		auth := []gossh.AuthMethod{
			gossh.Password(config.SSHPassword),
		}

		if config.SSHKeyPath != "" {
			signer, err := FileSigner(config.SSHKeyPath)
			if err != nil {
				return nil, err
			}
			auth = append(auth, gossh.PublicKeys(signer))
		}

		return &gossh.ClientConfig{
			User:            config.SSHUser,
			Auth:            auth,
			HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		}, nil
	}
}

// SSH Command Execution

// doExecuteSSHCmd executes a command over SSH
func doExecuteSSHCmd(cmd, target string, config *gossh.ClientConfig) (stdout string, err error) {
	client, err := gossh.Dial("tcp", target, config)
	if err != nil {
		return "", err
	}

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

// ExecuteHostSSHCmd executes a command on the host over SSH
func ExecuteHostSSHCmd(state multistep.StateBag, cmd string) (stdout string, err error) {
	config := state.Get("config").(Config)
	sshAddress, _ := SSHAddress(state)
	sshConfig := &gossh.ClientConfig{
		User: config.Username,
		Auth: []gossh.AuthMethod{
			gossh.Password(config.Password),
		},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}
	return doExecuteSSHCmd(cmd, sshAddress, sshConfig)
}

// ExecuteGuestSSHCmd executes a command on the guest over SSH
func ExecuteGuestSSHCmd(state multistep.StateBag, cmd string) (stdout string, err error) {
	config := state.Get("config").(Config)
	localAddress, err := SSHLocalAddress(state)
	if err != nil {
		return
	}
	sshConfig, err := SSHConfigFunc(config.SSHConfig)(state)
	if err != nil {
		return
	}

	return doExecuteSSHCmd(cmd, localAddress, sshConfig)
}

// SSH Port Forwarding

// forward handles the SSH port forwarding for a single connection
func forward(local_conn net.Conn, config *gossh.ClientConfig, server string, server_ssh_port int, remote_dest string, remote_port uint) error {
	defer local_conn.Close()

	ssh_client_conn, err := gossh.Dial("tcp", fmt.Sprintf("%s:%d", server, server_ssh_port), config)
	if err != nil {
		log.Printf("local ssh.Dial error: %s", err)
		return err
	}
	defer ssh_client_conn.Close()

	remote_loc := fmt.Sprintf("%s:%d", remote_dest, remote_port)
	ssh_conn, err := ssh_client_conn.Dial("tcp", remote_loc)
	if err != nil {
		log.Printf("ssh.Dial error: %s", err)
		return err
	}
	defer ssh_conn.Close()

	txDone := make(chan struct{})
	rxDone := make(chan struct{})

	go func() {
		_, err = io.Copy(ssh_conn, local_conn)
		if err != nil {
			log.Printf("io.copy failed: %v", err)
		}
		close(txDone)
	}()

	go func() {
		_, err = io.Copy(local_conn, ssh_conn)
		if err != nil {
			log.Printf("io.copy failed: %v", err)
		}
		close(rxDone)
	}()

	<-txDone
	<-rxDone

	return nil
}

// ssh_port_forward sets up SSH port forwarding
func ssh_port_forward(local_listener net.Listener, remote_port int, remote_dest, host string, host_ssh_port int, username, password string) error {
	config := &gossh.ClientConfig{
		User: username,
		Auth: []gossh.AuthMethod{
			gossh.Password(password),
		},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}

	for {
		local_connection, err := local_listener.Accept()
		if err != nil {
			log.Printf("Local accept failed: %s", err)
			return err
		}

		go forward(local_connection, config, host, host_ssh_port, remote_dest, uint(remote_port))
	}
}

// SSH Key Management

// FileSigner creates an SSH signer from a private key file
func FileSigner(path string) (gossh.Signer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	keyBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, fmt.Errorf("Failed to read key '%s': no key found", path)
	}
	if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
		return nil, fmt.Errorf(
			"Failed to read key '%s': password protected keys are\n"+
				"not supported. Please decrypt the key prior to use.", path)
	}

	signer, err := gossh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("Error setting up SSH config: %s", err)
	}

	return signer, nil
}