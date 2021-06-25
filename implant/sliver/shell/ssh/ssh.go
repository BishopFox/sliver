package ssh

import (
	"bytes"
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func RunSSHCommand(host string, port uint16, username string, command string) (string, string, error) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	// ssh-agent(1) provides a UNIX socket at $SSH_AUTH_SOCK.
	socket := os.Getenv("SSH_AUTH_SOCK")
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return "", "", err
	}

	agentClient := agent.NewClient(conn)
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			// Use a callback rather than PublicKeys so we only consult the
			// agent once the remote server wants it.
			ssh.PublicKeysCallback(agentClient.Signers),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshc, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		return "", "", (err)
	}
	// Use sshc...
	session, err := sshc.NewSession()
	if err != nil {
		return "", "", err
	}
	session.Stderr = &stderr
	session.Stdout = &stdout
	err = session.Run(command)
	if err != nil {
		return "", stderr.String(), err
	}
	sshc.Close()
	return stdout.String(), stderr.String(), nil
}
