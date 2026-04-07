package main

import (
	"fmt"
	"os"

	"github.com/els0r/gw/cmd/gw-log/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "gw-log: %v\n", err)
		os.Exit(1)
	}
}
