// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The application roots at its go.mod, not the monorepo's git toplevel, so
// every gate and rig path stays inside applications/scrum-master (GH-190).
func TestRepositoryRootIsTheApplication(t *testing.T) {
	root, err := repositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	if _, serr := os.Stat(filepath.Join(root, "go.mod")); serr != nil || filepath.Base(root) == "magefiles" {
		t.Fatalf("root = %q, want the directory holding go.mod: %v", root, serr)
	}
	if _, serr := os.Stat(filepath.Join(root, "plugin", "agents")); serr != nil {
		t.Fatalf("root %q does not hold the application's profiles: %v", root, serr)
	}
}

// TestAuditFailsOnMissingIndexedDocument proves the acceptance criterion of
// GH-39: a corpus whose SPECIFICATIONS.md index names a document that does
// not exist fails the corpus audit with index-broken-path. It runs the
// sibling agent on a copy of this corpus with docs/VISION.md removed; it
// builds the agent, so it runs only when SCRUM_MASTER_AUDIT_FIXTURE is set.
func TestAuditFailsOnMissingIndexedDocument(t *testing.T) {
	if os.Getenv("SCRUM_MASTER_AUDIT_FIXTURE") == "" {
		t.Skip("set SCRUM_MASTER_AUDIT_FIXTURE to build the sibling agent and run the fixture")
	}
	work := t.TempDir()
	if err := runStreaming(work, "cp", "-R", mustAbs(t, filepath.Join("..", "docs")), "."); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(work, "docs", "VISION.md")); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	capture := func(dir, name string, args ...string) error {
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		cmd.Stdout = &out
		cmd.Stderr = &out
		return cmd.Run()
	}
	err := runCorpusAudit(work, capture)
	if err == nil {
		t.Fatal("the audit passed a corpus whose index names a missing document")
	}
	if !strings.Contains(out.String(), "index-broken-path") {
		t.Fatalf("the audit failed without an index-broken-path finding:\n%s", out.String())
	}
}
