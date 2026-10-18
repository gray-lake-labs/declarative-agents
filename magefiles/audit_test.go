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

func TestParentCheckoutUsesOverrideThenSibling(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "repo")
	writeFile(t, filepath.Join(base, "cache", ".git", "HEAD"), "ref\n")
	writeFile(t, filepath.Join(base, "declarative-agents", ".git", "HEAD"), "ref\n")

	got, err := parentCheckout(root)
	if err != nil || got != filepath.Join(base, "declarative-agents") {
		t.Fatalf("sibling: got=%q err=%v", got, err)
	}

	writeFile(t, filepath.Join(root, checkoutsFile), "checkouts:\n  "+parentRepository+": ../cache\n")
	got, err = parentCheckout(root)
	if err != nil || got != filepath.Join(base, "cache") {
		t.Fatalf("override: got=%q err=%v", got, err)
	}
}

func TestParentCheckoutNamesAMissingCache(t *testing.T) {
	root := t.TempDir()
	if _, err := parentCheckout(root); err == nil || !strings.Contains(err.Error(), "corpus-audit.md") {
		t.Fatalf("err = %v", err)
	}
}

func TestEnsureAgentNamesAMissingTag(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	fail := func(dir, name string, args ...string) error {
		if name == "git" && args[0] == "show-ref" {
			return os.ErrNotExist
		}
		return nil
	}
	if _, _, err := ensureAgent("/cache", "v9", fail); err == nil || !strings.Contains(err.Error(), "fetch --tags") {
		t.Fatalf("err = %v", err)
	}
}
