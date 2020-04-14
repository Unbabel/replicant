/*Package cmd implements replicant commands*/
package cmd

import (
	"fmt"
	"os"

	"github.com/Unbabel/replicant/internal/cmdutil"
	"github.com/Unbabel/replicant/log"
	"github.com/spf13/cobra"
)

func init() {
	Root.PersistentFlags().String("log-level", "INFO", "log level")
	Root.AddCommand(Server)
	Root.AddCommand(Executor)
	Root.AddCommand(Txn)
}

// Root command for replicant
var Root = &cobra.Command{
	Use:   "replicant",
	Short: "replicant command line interface",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.Init(cmdutil.GetFlagString(cmd, "log-level"))

	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func die(format string, a ...interface{}) {
	fmt.Printf(format+"\n", a...)
	os.Exit(1)
}
