// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestGoSourcesAreFormatted(t *testing.T) {
	out, err := exec.Command("gofmt", "-l", "..").CombinedOutput()
	if err != nil {
		t.Fatalf("gofmt: %v\n%s", err, out)
	}
	if files := strings.TrimSpace(string(out)); files != "" {
		t.Fatalf("not gofmt-formatted:\n%s", files)
	}
}
