package main

import (
	"os"

	"github.com/gunysa1/tgfs/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
