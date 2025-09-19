package main

import (
	"os"

	"github.com/Additional-Code/atlas/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
