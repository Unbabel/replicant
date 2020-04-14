package cmd

import (
	"github.com/Unbabel/replicant/client"
	"github.com/Unbabel/replicant/internal/cmdutil"
	"github.com/spf13/cobra"
)

// Delete command
var Delete = &cobra.Command{
	Use:   "delete",
	Short: "Delete a transaction from a replicant server",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		name := cmdutil.GetFlagString(cmd, "name")
		if name == "" {
			die("Transaction name must be specified")
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

		err = c.Delete(name)
		if err != nil {
			die(err.Error())
		}

	},
}
