package main

import (
	"os"

	"github.com/opheelia/blockchainbeat/cmd"

	_ "github.com/opheelia/blockchainbeat/include"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
