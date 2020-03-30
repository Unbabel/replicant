package cmd

import (
	"time"

	"github.com/spf13/cobra"
)

func init() {
	Txn.PersistentFlags().StringP("server-url", "s", "http://127.0.0.1:8080", "Replicant server URL")
	Txn.PersistentFlags().String("name", "", "Name of managed server transaction")
	Txn.PersistentFlags().StringP("username", "u", "", "Replicant server username")
	Txn.PersistentFlags().StringP("password", "p", "", "Replicant server password")
	Txn.PersistentFlags().StringP("file", "f", "", "Path to transaction definition file")
	Txn.PersistentFlags().Bool("insecure", false, "Skip server certificate verification")
	Txn.PersistentFlags().DurationP("timeout", "t", 5*time.Minute, "Replicant server timeout for running transactions")
	Txn.AddCommand(Add)
	Txn.AddCommand(Get)
	Txn.AddCommand(Run)
	Txn.AddCommand(Delete)
	Txn.AddCommand(RunLocal)
}

// Txn command
var Txn = &cobra.Command{
	Use:   "txn",
	Short: "Manage and run transactions",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}
