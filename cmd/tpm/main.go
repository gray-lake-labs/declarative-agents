// Command tpm is the process's deterministic half: every step a program can
// do (design.md D10). Skills call it with --json.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/gray-lake-labs/scrum-master/internal/beads"
	"github.com/gray-lake-labs/scrum-master/internal/cli"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"

	_ "github.com/gray-lake-labs/scrum-master/internal/commands"
)

func main() {
	tracker := func() (tracker.Tracker, error) {
		if !beads.Configured() {
			return nil, errors.New("no tracker: set TPM_TRACKER to the tracker repository (eng04-beads-sync)")
		}
		return beads.FromEnv()
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
