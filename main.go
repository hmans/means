package main

import (
	"os"

	"github.com/hmans/means/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.Execute(os.Args[1:], os.Stdout, os.Stderr, version); err != nil {
		os.Exit(1)
	}
}
