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

func TestPinnedReleaseReadsTheParentPin(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, pinsFile), "pins:\n  "+parentRepository+": v0.20261014.0\n")
	tag, err := pinnedRelease(root)
	if err != nil || tag != "v0.20261014.0" {
		t.Fatalf("tag=%q err=%v", tag, err)
	}
}

func TestPinnedReleaseRefusesAMissingPin(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, pinsFile), "pins: {}\n")
	if _, err := pinnedRelease(root); err == nil || !strings.Contains(err.Error(), parentRepository) {
		t.Fatalf("err = %v", err)
	}
}

// TestAuditFailsOnMissingIndexedDocument proves the acceptance criterion of
// GH-39: a corpus whose SPECIFICATIONS.md index names a document that does
// not exist fails the corpus audit with index-broken-path. It runs the
// pinned release's agent on a copy of this corpus with docs/VISION.md
// removed, so it downloads the release; it runs only when
// SCRUM_MASTER_AUDIT_FIXTURE_TAG names one.
func TestAuditFailsOnMissingIndexedDocument(t *testing.T) {
	tag := os.Getenv("SCRUM_MASTER_AUDIT_FIXTURE_TAG")
	if tag == "" {
		t.Skip("set SCRUM_MASTER_AUDIT_FIXTURE_TAG to a published release")
	}
	work := t.TempDir()
	if err := runStreaming(work, "cp", "-R", mustAbs(t, filepath.Join("..", "docs")), "."); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(work, pinsFile), "pins:\n  "+parentRepository+": "+tag+"\n")
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
