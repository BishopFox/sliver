package creds

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// SelectCredential - Interactive menu for the user to select a credentials from the database.
func SelectCredential(plaintext bool, hashType clientpb.HashType, con *console.SliverClient) (*clientpb.Credential, error) {
	var creds *clientpb.Credentials
	var err error
	if hashType == clientpb.HashType_INVALID {
		creds, err = con.Rpc.Creds(context.Background(), &commonpb.Empty{})
		if err != nil {
			return nil, con.UnwrapServerErr(err)
		}
	} else {
		creds, err = con.Rpc.GetCredsByHashType(context.Background(), &clientpb.Credential{HashType: hashType})
		if err != nil {
			return nil, con.UnwrapServerErr(err)
		}
	}
	if len(creds.Credentials) == 0 {
		return nil, errors.New("no credentials to select from")
	}

	// Render selection table
	buf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(buf, 0, 2, 2, ' ', 0)
	for _, cred := range creds.Credentials {
		if cred.Plaintext == "" {
			fmt.Fprintf(table, "%s\t%s\t%s\t[not cracked]\t\n", strings.Split(cred.ID, "-")[0], cred.Collection, cred.HashType)
		} else {
			plaintext := cred.Plaintext
			if len(plaintext) > 12 {
				plaintext = plaintext[:12] + "..."
			}
			fmt.Fprintf(table, "%s\t%s\t%s\tplaintext: %s\t\n", strings.Split(cred.ID, "-")[0], cred.Collection, cred.HashType, plaintext)
		}
	}
	table.Flush()
	options := strings.Split(buf.String(), "\n")
	options = options[:len(options)-1]
	if len(options) == 0 {
		return nil, errors.New("no credentials to select from")
	}

	selected := ""
	prompt := &survey.Select{
		Message: "Select a credential:",
		Options: options,
	}
	err = survey.AskOne(prompt, &selected)
	if err != nil {
		return nil, err
	}
	for index, value := range options {
		if value == selected {
			return creds.Credentials[index], nil
		}
	}
	return nil, nil
}
