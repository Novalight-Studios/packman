package main

import (
	"os"

	"github.com/Novalight-Studios/packman/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
