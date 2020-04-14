package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"text/tabwriter"

	"github.com/Unbabel/replicant/client"
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

		var result transaction.Result
		switch {
		case file != "":
			result, err = c.Run(tx)
		case name != "":
			result, err = c.RunByName(name)
		}

		if err != nil {
			die("Error running transaction: %s", err)
		}

		switch cmdutil.GetFlagString(cmd, "output") {
		case "":
			w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', tabwriter.TabIndent)
			fmt.Fprintf(w, "NAME\tDRIVER\tFAILED\tDURATION\tRETRIES\tTIME\n")
			t, _ := result.Time.MarshalText()
			fmt.Fprintf(w, "%s\t%s\t%t\t%.2f\t%d\t%s\n",
				result.Name, result.Driver, result.Failed, result.DurationSeconds, result.RetryCount, t)
			w.Flush()
		case "json":
			buf, err := json.MarshalIndent(&result, "", "  ")
			if err != nil {
				die(err.Error())
			}
			fmt.Printf("%s\n", buf)
		case "yaml":
			buf, err := yaml.Marshal(&result)
			if err != nil {
				die(err.Error())
			}
			fmt.Printf("%s\n", buf)
		}

		fmt.Print()
	},
}
