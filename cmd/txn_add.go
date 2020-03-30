package cmd

import (
	"io/ioutil"

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

		file := cmdutil.GetFlagString(cmd, "file")
		if file == "" {
			die("Transaction file must be specified")
		}

		buf, err := ioutil.ReadFile(file)
		if err != nil {
			die("Error reading transaction: %s", err)
		}

		if err = yaml.Unmarshal(buf, &tx); err != nil {
			die("Error reading transaction: %s", err)
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
