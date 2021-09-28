package common

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	gossh "golang.org/x/crypto/ssh"
)

func SSHAddress(state multistep.StateBag) (string, error) {
	sshIP := state.Get("ssh_address").(string)
	sshHostPort := 22
	return fmt.Sprintf("%s:%d", sshIP, sshHostPort), nil
}

func doExecuteSSHCmd(cmd, target string, config *gossh.ClientConfig) (stdout string, err error) {
	client, err := gossh.Dial("tcp", target, config)
	if err != nil {
		return "", err
	}

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

func ExecuteHostSSHCmd(state multistep.StateBag, cmd string) (stdout string, err error) {
	config := state.Get("commonconfig").(CommonConfig)
	sshAddress, _ := SSHAddress(state)
	// Setup connection config
	sshConfig := &gossh.ClientConfig{
		User: config.Username,
		Auth: []gossh.AuthMethod{
			gossh.Password(config.Password),
		},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}
	return doExecuteSSHCmd(cmd, sshAddress, sshConfig)
}
