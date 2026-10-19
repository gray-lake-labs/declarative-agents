package main

import (
	"os"
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
