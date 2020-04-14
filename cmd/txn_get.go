package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Unbabel/replicant/client"
	"github.com/Unbabel/replicant/internal/cmdutil"
	"github.com/Unbabel/replicant/transaction"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func init() {
	Get.Flags().Bool("results", false, "Get transaction results")
}

// Get command
var Get = &cobra.Command{
	Use:   "get",
	Short: "Get transaction definitions and result data",
	Run: func(cmd *cobra.Command, args []string) {

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

		if cmdutil.GetFlagBool(cmd, "results") {
			getResults(cmd, c)
			return
		}

		getTransactions(cmd, c)
	},
}

func getTransactions(cmd *cobra.Command, c *client.Client) {
	var err error
	name := cmdutil.GetFlagString(cmd, "name")
	output := cmdutil.GetFlagString(cmd, "output")

	var ts []transaction.Config
	switch name != "" {
	case true:
		t, err := c.GetTransaction(name)
		if err != nil {
			die(err.Error())
		}
		ts = append(ts, t)

	case false:
		ts, err = c.GetTransactions()
		if err != nil {
			die(err.Error())
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', tabwriter.TabIndent)
	fmt.Fprintf(w, "NAME\tDRIVER\tSCHEDULE\tTIMEOUT\tRETRIES\tCALLBACK\n")
	for x := 0; x < len(ts); x++ {

		if output == "yaml" {
			buf, err := yaml.Marshal(&ts[x])
			if err != nil {
				die(err.Error())
			}
			fmt.Printf("%s\n", buf)
		}

		if output == "json" {
			buf, err := json.MarshalIndent(&ts[x], "", "  ")
			if err != nil {
				die(err.Error())
			}
			fmt.Printf("%s\n", buf)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%t\n",
			ts[x].Name, ts[x].Driver, ts[x].Schedule, ts[x].Timeout, ts[x].RetryCount, ts[x].CallBack != nil)
	}

	if output == "" {
		w.Flush()
	}
}

func getResults(cmd *cobra.Command, c *client.Client) {
	var err error
	name := cmdutil.GetFlagString(cmd, "name")
	output := cmdutil.GetFlagString(cmd, "output")

	var rs []transaction.Result
	switch name != "" {
	case true:
		r, err := c.GetResult(name)
		if err != nil {
			die(err.Error())
		}
		rs = append(rs, r)

	case false:
		rs, err = c.GetResults()
		if err != nil {
			die(err.Error())
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', tabwriter.TabIndent)
	fmt.Fprintf(w, "NAME\tDRIVER\tFAILED\tDURATION\tRETRIES\tTIME\n")
	for x := 0; x < len(rs); x++ {

		if output == "yaml" {
			buf, err := yaml.Marshal(&rs[x])
			if err != nil {
				die(err.Error())
			}
			fmt.Printf("%s\n", buf)
		}

		if output == "json" {
			buf, err := json.MarshalIndent(&rs[x], "", "  ")
			if err != nil {
				die(err.Error())
			}
			fmt.Printf("%s\n", buf)
		}

		t, _ := rs[x].Time.MarshalText()
		fmt.Fprintf(w, "%s\t%s\t%t\t%.2f\t%d\t%s\n",
			rs[x].Name, rs[x].Driver, rs[x].Failed, rs[x].DurationSeconds, rs[x].RetryCount, t)
	}

	if output == "" {
		w.Flush()
	}
}
