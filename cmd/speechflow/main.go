// Package main is the speechflow CLI entrypoint.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/camggould/speechflow/internal/cli"
)

func main() {
	err := cli.NewRootCommand().Execute()
	if err == nil {
		return
	}
	var ec *cli.ExitError
	if errors.As(err, &ec) {
		if ec.Message != "" {
			fmt.Fprintln(os.Stderr, "error:", ec.Message)
		}
		os.Exit(ec.Code)
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
