// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// upstream is the organization the parent was forked from. It is assembled
// here so this file does not name it.
var upstream = "Nokia" + "-Bell-Labs"

// TestNoTrackedFileNamesTheUpstream holds the repository to the fork it
// consumes: no tracked file names the upstream organization (GH-29).
func TestNoTrackedFileNamesTheUpstream(t *testing.T) {
	out, err := exec.Command("git", "-C", "..", "ls-files").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range strings.Fields(string(out)) {
		b, err := os.ReadFile(filepath.Join("..", f))
		if err != nil {
			continue
		}
		if bytes.Contains(bytes.ToLower(b), []byte(strings.ToLower(upstream))) {
			t.Errorf("%s names %s; name the parent as gray-lake-labs/declarative-agents", f, upstream)
		}
	}
}
