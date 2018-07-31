package main

import (
	"os"

	"github.com/yyff/cmqbeat/cmd"

	_ "github.com/yyff/cmqbeat/include"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
