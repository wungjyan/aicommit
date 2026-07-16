package main

import (
	"os"

	"github.com/wungjyan/aicommit/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(cmd.ExitCode(err))
	}
}
