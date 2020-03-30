package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/Unbabel/replicant/emitter/stdout"
	"github.com/Unbabel/replicant/internal/cmdutil"
	"github.com/Unbabel/replicant/internal/executor"
	"github.com/Unbabel/replicant/internal/tmpl"
	"github.com/Unbabel/replicant/transaction"
	"github.com/segmentio/ksuid"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func init() {
	RunLocal.Flags().String("chrome-remote-url", "http://127.0.0.1:9222", "Chrome remote debugging protocol server")
}

// RunLocal command
var RunLocal = &cobra.Command{
	Use:   "run-local",
	Short: "Run transactions locally for development",
	Run: func(cmd *cobra.Command, args []string) {

		file := cmdutil.GetFlagString(cmd, "file")

		if file == "" {
			die("No transaction file specified")
		}

		buf, err := ioutil.ReadFile(file)
		if err != nil {
			die("Error reading transaction: %s", err)
		}

		tx := transaction.Config{}
		if err = yaml.Unmarshal(buf, &tx); err != nil {
			die("Error reading transaction: %s", err)
		}

		if tx.CallBack != nil {
			die("Callbacks still not supported in local runs")
		}

		tx, err = tmpl.Parse(tx)
		if err != nil {
			die("Error parsing transaction: %s", err)
		}

		config := executor.Config{}
		config.Web.ServerURL = cmdutil.GetFlagString(cmd, "chrome-remote-url")
		e, err := executor.New(config)
		if err != nil {
			die("Error creating local executor: %s", err)
		}

		em := stdout.New(stdout.Config{Pretty: true})
		result, err := e.Run(ksuid.New().String(), tx)
		if err != nil {
			die("Error running transaction: %s", err)
		}

		em.Emit(result)
		fmt.Print()
	},
}
