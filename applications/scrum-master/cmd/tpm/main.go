// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
// Command tpm is the process's deterministic half: every step a program can
// do (design.md D10). Skills call it with --json; an agent build that links
// the words package runs the same commands in its own process (GH-27).
package main

import (
	"os"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/words"
)

func main() {
	os.Exit(words.Main(os.Args[1:], os.Stdout, os.Stderr))
}
