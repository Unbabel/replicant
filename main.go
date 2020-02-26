package main

import (
	"fmt"
	"os"

	"github.com/Unbabel/replicant/cmd"
)

var (
	Version   string
	GitCommit string
	BuildTime string
)

func main() {
	cmd.Root.Version = fmt.Sprintf("%s %s %s", Version, BuildTime, GitCommit)
	if err := cmd.Root.Execute(); err != nil {
		fmt.Printf("error running command:\n%s\n", err)
		os.Exit(1)
	}
}
