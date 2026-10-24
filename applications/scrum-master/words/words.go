// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0

// Package words runs tpm's commands, the words the Scrum Master profiles'
// exec declarations name, as one command line: from the tpm binary, or in
// the agent's own process when an agent build links this package in and
// serves Binary with Run (GH-27).
package words

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/cli"
	_ "github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/commands"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/install"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/routing"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/tracker"
)

// Binary is the name the profiles' exec declarations give the words.
const Binary = "tpm"

// Exit codes, as the tpm binary has always had them.
const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
)

// Run runs one command line of words with dir as the working directory,
// the tracker routed from it, writing the output and any error to out, and
// returns the exit code. An empty dir is the current one. The working
// directory is the process's: a caller running words in its own process
// serializes calls and gets its directory back afterwards.
func Run(dir string, args []string, out io.Writer) int {
	if dir != "" {
		prev, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(out, "tpm:", err)
			return ExitError
		}
		if err := os.Chdir(dir); err != nil {
			fmt.Fprintln(out, "tpm:", err)
			return ExitError
		}
		defer os.Chdir(prev)
	}
	return Main(args, out, out)
}

// Main is the tpm command line from the current directory: the output to
// out, an error to errOut, and the exit code.
func Main(args []string, out, errOut io.Writer) int {
	if len(args) == 1 && (args[0] == "--version" || args[0] == "version") {
		fmt.Fprintln(out, "tpm", install.Version())
		return ExitOK
	}
	open := func() (tracker.Tracker, error) {
		dir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		return routing.Open(dir)
	}
	err := cli.Default.Run(args, out, open)
	switch {
	case err == nil:
		return ExitOK
	case errors.Is(err, cli.ErrUsage):
		fmt.Fprintln(errOut, err)
		return ExitUsage
	default:
		fmt.Fprintln(errOut, "tpm:", err)
		return ExitError
	}
}
