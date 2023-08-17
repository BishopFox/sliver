package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/assets"
	"github.com/reeflective/team/internal/command"
	"github.com/reeflective/team/server"
)

func createUserCmd(serv *server.Server, cli *client.Client) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, _ []string) {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.WarnLevel))
			}
		}

		name, _ := cmd.Flags().GetString("name")
		lhost, _ := cmd.Flags().GetString("host")
		lport, _ := cmd.Flags().GetUint16("port")
		save, _ := cmd.Flags().GetString("save")
		system, _ := cmd.Flags().GetBool("system")

		if save == "" {
			save, _ = os.Getwd()
		}

		var filename string
		var saveTo string

		if system {
			user, err := user.Current()
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"Failed to get current OS user: %s\n", err)
				return
			}

			name = user.Username
			filename = fmt.Sprintf("%s_%s_default", serv.Name(), user.Username)
			saveTo = cli.ConfigsDir()

			err = os.MkdirAll(saveTo, assets.DirPerm)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"cannot write to %s root dir: %s\n", saveTo, err)
				return
			}
		} else {
			saveTo, _ = filepath.Abs(save)
			userFile, err := os.Stat(saveTo)
			if !os.IsNotExist(err) && !userFile.IsDir() {
				fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"File already exists %s\n", err)
				return
			}

			if !os.IsNotExist(err) && userFile.IsDir() {
				filename = fmt.Sprintf("%s_%s", filepath.Base(name), filepath.Base(lhost))
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), command.Info+"Generating new client certificate, please wait ... \n")

		config, err := serv.UserCreate(name, lhost, lport)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"%s\n", err)
			return
		}

		configJSON, err := json.Marshal(config)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"JSON marshaling error: %s\n", err)
			return
		}

		saveTo = filepath.Join(saveTo, filename+".teamclient.cfg")

		err = os.WriteFile(saveTo, configJSON, assets.FileReadPerm)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"Failed to write config to %s: %s\n", saveTo, err)
			return
		}

		fmt.Fprintf(cmd.OutOrStdout(), command.Info+"Saved new client config to: %s\n", saveTo)
	}
}

func rmUserCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.WarnLevel))
			}
		}

		user := args[0]

		fmt.Fprintf(cmd.OutOrStdout(), command.Info+"Removing client certificate(s)/token(s) for %s, please wait ... \n", user)

		err := serv.UserDelete(user)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"Failed to remove the user certificate: %v\n", err)
			return
		}

		fmt.Fprintf(cmd.OutOrStdout(), command.Info+"User %s has been deleted from the teamserver, and kicked out.\n", user)
	}
}

func importCACmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.WarnLevel))
			}
		}

		load := args[0]

		fi, err := os.Stat(load)
		if os.IsNotExist(err) || fi.IsDir() {
			fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"Cannot load file %s\n", load)
		}

		data, err := os.ReadFile(load)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"Cannot read file: %v\n", err)
		}

		// CA - Exported CA format
		type CA struct {
			Certificate string `json:"certificate"`
			PrivateKey  string `json:"private_key"`
		}

		importCA := &CA{}
		err = json.Unmarshal(data, importCA)

		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"Failed to parse file: %s\n", err)
		}

		cert := []byte(importCA.Certificate)
		key := []byte(importCA.PrivateKey)
		serv.UsersSaveCA(cert, key)
	}
}

func exportCACmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.WarnLevel))
			}
		}

		var save string
		if len(args) == 1 {
			save = args[0]
		}

		if strings.TrimSpace(save) == "" {
			save, _ = os.Getwd()
		}

		certificateData, privateKeyData, err := serv.UsersGetCA()
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"Error reading CA %s\n", err)
			return
		}

		// CA - Exported CA format
		type CA struct {
			Certificate string `json:"certificate"`
			PrivateKey  string `json:"private_key"`
		}

		exportedCA := &CA{
			Certificate: string(certificateData),
			PrivateKey:  string(privateKeyData),
		}

		saveTo, _ := filepath.Abs(save)

		caFile, err := os.Stat(saveTo)
		if !os.IsNotExist(err) && !caFile.IsDir() {
			fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"File already exists: %s\n", err)
			return
		}

		if !os.IsNotExist(err) && caFile.IsDir() {
			filename := fmt.Sprintf("%s-%s.teamserver.ca", serv.Name(), "users")
			saveTo = filepath.Join(saveTo, filename)
		}

		data, _ := json.Marshal(exportedCA)

		err = os.WriteFile(saveTo, data, assets.FileWritePerm)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"Write failed: %s (%s)\n", saveTo, err)
			return
		}
	}
}
