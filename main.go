package main

import (
	"fmt"
	"os"

	"github.com/tu-graz/kanboard-cli/internal/cmd"
)

func main() {
	root := cmd.NewRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
