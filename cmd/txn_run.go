package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/Unbabel/replicant/client"
	"github.com/Unbabel/replicant/emitter/stdout"
	"github.com/Unbabel/replicant/internal/cmdutil"
	"github.com/Unbabel/replicant/transaction"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// Run command
var Run = &cobra.Command{
	Use:   "run",
	Short: "Run provided or remote stored transactions on a replicant server",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var tx transaction.Config

		file := cmdutil.GetFlagString(cmd, "file")
		name := cmdutil.GetFlagString(cmd, "name")

		if file == "" && name == "" {
			die("Either a remote transaction name or a transaction definition file must be specified")
		}

		if file != "" {
			buf, err := ioutil.ReadFile(file)
			if err != nil {
				die("Error reading transaction: %s", err)
			}

			if err = yaml.Unmarshal(buf, &tx); err != nil {
				die("Error reading transaction: %s", err)
			}
		}

		var result transaction.Result
		em := stdout.New(stdout.Config{Pretty: true})

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

		switch {
		case file != "":
			result, err = c.Run(tx)
		case name != "":
			result, err = c.RunByName(name)
		}

		if err != nil {
			die("Error running transaction: %s", err)
		}

		em.Emit(result)
		fmt.Print()
	},
}
