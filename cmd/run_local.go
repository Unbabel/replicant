package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Unbabel/replicant/emitter/stdout"
	"github.com/Unbabel/replicant/internal/cmdutil"
	"github.com/Unbabel/replicant/internal/executor"
	"github.com/Unbabel/replicant/internal/tmpl"
	"github.com/Unbabel/replicant/log"
	"github.com/Unbabel/replicant/transaction"
	"github.com/segmentio/ksuid"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func init() {
	Run.Flags().String("chrome-remote-url", "http://127.0.0.1:9222", "Chrome remote debugging protocol server")
	Run.Flags().String("file", "", "Path to transaction definition file")
}

// Run command
var Run = &cobra.Command{
	Use:   "run",
	Short: "Run transactions for local development",
	Run: func(cmd *cobra.Command, args []string) {
		buf, err := ioutil.ReadFile(cmdutil.GetFlagString(cmd, "file"))
		if err != nil {
			log.Error("error reading transaction").String("file", cmdutil.GetFlagString(cmd, "file")).Error("error", err).Log()
			os.Exit(1)
		}

		tx := transaction.Config{}
		if err = yaml.Unmarshal(buf, &tx); err != nil {
			log.Error("error reading transaction").String("file", cmdutil.GetFlagString(cmd, "file")).Error("error", err).Log()
			os.Exit(1)
		}

		if tx.CallBack != nil {
			log.Error("callbacks not supported in local runs").Log()
			os.Exit(1)
		}

		tx, err = tmpl.Parse(tx)
		if err != nil {
			log.Error("error parsing transaction").String("file", cmdutil.GetFlagString(cmd, "file")).Error("error", err).Log()
			os.Exit(1)
		}

		config := executor.Config{}
		config.Web.ServerURL = cmdutil.GetFlagString(cmd, "chrome-remote-url")
		e, err := executor.New(config)
		if err != nil {
			log.Error("error creating local replicant-executor").Error("error", err).Log()
			os.Exit(1)
		}

		em := stdout.New(stdout.Config{Pretty: true})
		result, err := e.Run(ksuid.New().String(), tx)
		if err != nil {
			log.Error("error running transaction").Error("error", err).Log()
			os.Exit(1)
		}

		em.Emit(result)
		fmt.Print()
	},
}
