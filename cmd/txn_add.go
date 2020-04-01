package cmd

import (
	"github.com/Unbabel/replicant/client"
	"github.com/Unbabel/replicant/internal/cmdutil"
	"github.com/Unbabel/replicant/transaction"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// Add command
var Add = &cobra.Command{
	Use:   "add",
	Short: "Add a transaction to a replicant server",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var tx transaction.Config

		buf, err := loadFile(cmdutil.GetFlagString(cmd, "file"))
		if err != nil {
			die("Error reading transaction file: %s", err)
		}

		if err = yaml.Unmarshal(buf, &tx); err != nil {
			die("Error reading transaction file: %s", err)
		}

		if tx.Driver == "go_binary" {
			buf, err := loadFile(cmdutil.GetFlagString(cmd, "binary"))
			if err != nil {
				die("Error reading transaction's binary: %s", err)
			}
			tx.Binary = buf
		}

		c, err := client.New(client.Config{
			URL:                cmdutil.GetFlagString(cmd, "server-url"),
			Username:           cmdutil.GetFlagString(cmd, "username"),
			Password:           cmdutil.GetFlagString(cmd, "password"),
			Timeout:            cmdutil.GetFlagDuration(cmd, "timeout"),
			InsecureSkipVerify: cmdutil.GetFlagBool(cmd, "insecure"),
		})

		if err != nil {
			die("Error creating client: %s", err)
		}

		err = c.Add(tx)
		if err != nil {
			die(err.Error())
		}

	},
}
