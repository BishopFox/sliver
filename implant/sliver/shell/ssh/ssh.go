package ssh

import (
	"bytes"
	"fmt"
	"io/ioutil"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"net"
	"os"

	"github.com/yiya1989/sshkrb5/krb5forssh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func getClient(host string, port uint16, username string, password string, privKey []byte, signedUserCert []byte, krb5conf string, keytab []byte, realm string) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod
	if password != "" {
		// Try password auth first
		authMethods = append(authMethods, ssh.Password(password))
	} else if len(privKey) != 0 {
		// Then try private key
		signer, err := ssh.ParsePrivateKey(privKey)
		if err != nil {
			return nil, err
		}
		// If certificate based auth add signed public key
		if len(signedUserCert) != 0 {
			cert, _, _, _, err := ssh.ParseAuthorizedKey(signedUserCert)
			if err != nil {
				return nil, err
			}

			// create a signer using both the certificate and the private key:
			certSigner, err := ssh.NewCertSigner(cert.(*ssh.Certificate), signer)
			if err != nil {
				return nil, err
			}
			//Update signer
			signer = certSigner
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else if krb5conf != "" && keytab != nil && realm != "" {
		// Then try kerberos auth
		krb5ConfData, err := ioutil.ReadFile(krb5conf)
		if err != nil {
			return nil, err
		}
		sshGSSAPIClient, err := krb5forssh.NewKrb5InitiatorClient(string(krb5ConfData), username, realm, keytab)
		if err != nil {
			return nil, err
		}
		authMethods = append(authMethods, ssh.GSSAPIWithMICAuthMethod(&sshGSSAPIClient, host))
	} else {
		// Use ssh-agent if neither password nor private key has been provided
		// ssh-agent(1) provides a UNIX socket at $SSH_AUTH_SOCK.
		socket := os.Getenv("SSH_AUTH_SOCK")
		conn, err := net.Dial("unix", socket)
		if err != nil {
			return nil, err
		}
		agentClient := agent.NewClient(conn)
		authMethods = append(authMethods, ssh.PublicKeysCallback(agentClient.Signers))
	}

	// {{if .Config.Debug}}
	log.Printf("Auth methods: %+v\n", authMethods)
	// {{end}}

	config := &ssh.ClientConfig{
		User: username,
		Auth: authMethods,
		// This setting is insecure, but we need to be able
		// to connect to any host, not only those in the target's
		// known_hosts file
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshc, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		return nil, err
	}
	return sshc, nil

}

// RunSSHCommand - SSH to a host and execute a command
func RunSSHCommand(host string, port uint16, username string, password string, privKey []byte, signedUserCert []byte, krb5conf string, keytab []byte, realm string, command string) (string, string, error) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	sshc, err := getClient(host, port, username, password, privKey, signedUserCert, krb5conf, keytab, realm)
	if err != nil {
		return "", "", err
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
