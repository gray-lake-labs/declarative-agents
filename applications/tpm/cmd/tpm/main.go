// Command tpm is the process's deterministic half: every step a program can
// do (design.md D10). Skills call it with --json.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/gray-lake-labs/scrum-master/internal/cli"
	"github.com/gray-lake-labs/scrum-master/internal/install"
	"github.com/gray-lake-labs/scrum-master/internal/routing"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"

	_ "github.com/gray-lake-labs/scrum-master/internal/commands"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Println("tpm", install.Version())
		return
	}
	tracker := func() (tracker.Tracker, error) {
		dir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		return routing.Open(dir)
	}
	err := cli.Default.Run(os.Args[1:], os.Stdout, tracker)
	switch {
	case err == nil:
	case errors.Is(err, cli.ErrUsage):
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	default:
		fmt.Fprintln(os.Stderr, "tpm:", err)
		os.Exit(1)
	}
}
