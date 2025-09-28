package main

import (
	"os"

	"github.com/LoriKarikari/compak/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}