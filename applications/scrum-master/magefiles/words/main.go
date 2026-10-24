// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0

// Command words is the rig's harness: the words as one command line, which
// the integration targets drive in place of a person's skill or a profile's
// run. It is never shipped; a release ships the agent, which serves the
// words in its own process (GH-37).
package main

import (
	"os"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/words"
)

func main() {
	os.Exit(words.Main(os.Args[1:], os.Stdout, os.Stderr))
}
