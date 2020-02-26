/*Package server implements utilities for working with spf13/cobra commands*/
package cmdutil

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// GetFlagString fetches the named flag from cmd and fails on error
func GetFlagString(cmd *cobra.Command, name string) (v string) {
	v, err := cmd.Flags().GetString(name)
	if err != nil {
		fmt.Printf("error accessing flag %s for command %s: %s", name, cmd.Name, err)
		os.Exit(1)
	}

	return v
}

// GetFlagBool fetches the named flag from cmd and fails on error
func GetFlagBool(cmd *cobra.Command, name string) (v bool) {
	v, err := cmd.Flags().GetBool(name)
	if err != nil {
		fmt.Printf("error accessing flag %s for command %s: %s", name, cmd.Name, err)
		os.Exit(1)
	}

	return v
}

// GetFlagDuration fetches the named flag from cmd and fails on error
func GetFlagDuration(cmd *cobra.Command, name string) (v time.Duration) {
	v, err := cmd.Flags().GetDuration(name)
	if err != nil {
		fmt.Printf("error accessing flag %s for command %s: %s", name, cmd.Name, err)
		os.Exit(1)
	}

	return v
}
