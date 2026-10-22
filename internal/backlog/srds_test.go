package backlog

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const srdUntraced = `---
id: srd010-delivery
title: "Subsystem: delivery"
---

<!-- req: R1 -->
## R1 - Fleet

<!-- req: R1.1 -->
### R1.1

The fleet runs promoted agents.

<!-- req: R1.2 -->
### R1.2

The fleet reports its health.

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 -->
### AC1

A promoted agent runs.

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
`

// TestCheckReadsSRDs counts the SRDs as documents and refuses a
// requirement item no acceptance criterion traces (#149).
func TestCheckReadsSRDs(t *testing.T) {
	stubSpec(t, nil, errors.New("no pinned release"))
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, SRDDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, SRDDir, "srd010-delivery.md"), []byte(srdUntraced), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if r.Documents != 1 || len(r.Findings) != 1 || r.Findings[0].Field != "R1.2" ||
		r.Findings[0].File != filepath.Join(SRDDir, "srd010-delivery.md") {
		t.Fatalf("report = %+v, want one SRD and R1.2 untraced", r)
	}
	if r.SpecSkipped != "no pinned release" {
		t.Fatalf("spec check skipped = %q", r.SpecSkipped)
	}
}

// TestCheckRunsTheSpecToolsShapeCheck reports the parent spec tool's shape
// findings, such as a VISION field holding the wrong form, and leaves its
// formatting findings to tpm doctor (#149).
func TestCheckRunsTheSpecToolsShapeCheck(t *testing.T) {
	stubSpec(t, []string{
		"docs/VISION.md: field-shape-mismatch: what_this_does holds a list, want prose",
		"docs/ARCHITECTURE.md: not-canonical: run mage fmt",
	}, errors.New("exit status 1"))
	r, err := Check(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Findings) != 1 || r.Findings[0].File != "docs/VISION.md" || r.Findings[0].Field != "field-shape-mismatch" ||
		!strings.Contains(r.Findings[0].Message, "what_this_does") || r.SpecSkipped != "" {
		t.Fatalf("report = %+v, want the VISION shape finding alone", r)
	}
}

func stubSpec(t *testing.T, lines []string, err error) {
	prev := specCheck
	specCheck = func(string, string, string) ([]string, error) { return lines, err }
	t.Cleanup(func() { specCheck = prev })
}

// TestBacklogCheckNamesUntrackedDocuments is srd045 R8.2 for a document git
// does not track: the shape check cannot read it, so the gate names it
// until it is staged.
func TestBacklogCheckNamesUntrackedDocuments(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	epic := filepath.Join(dir, "docs", "specs", "epics", "ep001-scoring.md")
	if err := os.MkdirAll(filepath.Dir(epic), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(epic, []byte("not an epic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if f := untracked(dir); len(f) != 1 || f[0].File != epic || f[0].Field != "untracked" {
		t.Fatalf("untracked epic = %+v, want one finding naming it", f)
	}
	git("add", "-A")
	if f := untracked(dir); len(f) != 0 {
		t.Fatalf("a staged epic = %+v, want none", f)
	}
	if f := untracked(t.TempDir()); f != nil {
		t.Fatalf("a directory git does not manage = %+v, want none", f)
	}
}
