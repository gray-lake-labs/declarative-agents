package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/scaffold"
)

func init() {
	integrationTargets = append(integrationTargets, Integration{}.Doctor)
}

// Doctor builds tpm, inits a system repository, turns it into one an older tpm laid down (README.md deleted, AGENTS.md the owner's without a block, a block rewritten around the owner's notes, a document out of canonical form), and proves tpm doctor's dry run writes nothing, its run restores README.md, appends the block to AGENTS.md, refreshes the block and keeps the notes, and repairs the document so the parent's corpus audit and the spec tool's check pass, and a second run changes nothing (test-rel04.1-orientation-doctor). It needs no rig.
func (Integration) Doctor() error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "tpm-doctor-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if tmp, err = filepath.EvalSymlinks(tmp); err != nil {
		return err
	}
	bin, repo := filepath.Join(tmp, "tpm"), filepath.Join(tmp, "system")
	if err := runStreaming(root, "go", "build", "-o", bin, "./cmd/tpm"); err != nil {
		return fmt.Errorf("build tpm: %w", err)
	}
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME=operator", "GIT_AUTHOR_EMAIL=operator@example.invalid",
		"GIT_COMMITTER_NAME=operator", "GIT_COMMITTER_EMAIL=operator@example.invalid")
	sh := func(name string, args ...string) error {
		cmd := exec.Command(name, args...)
		cmd.Dir, cmd.Env = repo, env
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, out)
		}
		return nil
	}
	if err := os.MkdirAll(repo, 0o755); err != nil {
		return err
	}
	for _, args := range [][]string{{"init", "-q", "-b", "main"}} {
		if err := sh("git", args...); err != nil {
			return err
		}
	}
	if err := sh(bin, "init", "--dir", repo); err != nil {
		return err
	}
	if err := sh("git", "add", "-A"); err != nil {
		return err
	}
	if err := sh("git", "commit", "-q", "-m", "init"); err != nil {
		return err
	}

	// What an older tpm left: no README.md, an AGENTS.md the owner wrote
	// without tpm's block, an epics README whose block is an older text
	// between the owner's notes, and a VISION out of canonical form.
	if err := os.Remove(filepath.Join(repo, "README.md")); err != nil {
		return err
	}
	ownAgents := "Run the linter before committing.\n"
	if err := os.WriteFile(filepath.Join(repo, scaffold.AgentsFile), []byte(ownAgents), 0o644); err != nil {
		return err
	}
	epics := filepath.Join(repo, "docs", "specs", "epics", "README.md")
	older := "owner's note above\n<!-- tpm:begin: an older note -->\nan older tpm's text\n" + scaffold.BlockEnd + "\nowner's note below\n"
	if err := os.WriteFile(epics, []byte(older), 0o644); err != nil {
		return err
	}
	vision := filepath.Join(repo, "docs", "VISION.md")
	canonical, err := os.ReadFile(vision)
	if err != nil {
		return err
	}
	if err := os.WriteFile(vision, []byte(strings.Replace(string(canonical), "## Problem\n\n", "## Problem\n\n\n\n", 1)), 0o644); err != nil {
		return err
	}
	// A control/labels.md from before the op axis, with a value recolored.
	labelsPath := filepath.Join(repo, "control", "labels.md")
	labels, err := os.ReadFile(labelsPath)
	if err != nil {
		return err
	}
	start, end := strings.Index(string(labels), "\n<!-- field: op -->"), strings.Index(string(labels), "\n<!-- field: state -->")
	if start < 0 || end < start {
		return fmt.Errorf("tpm init's control/labels.md has no op axis before state")
	}
	olderLabels := strings.Replace(string(labels[:start])+string(labels[end:]), "| build | #428bca |", "| build | #000000 |", 1)
	if err := os.WriteFile(labelsPath, []byte(olderLabels), 0o644); err != nil {
		return err
	}

	doctor := func(args ...string) (scaffold.DoctorResult, error) {
		var r scaffold.DoctorResult
		cmd := exec.Command(bin, append([]string{"doctor", "--json", "--dir", repo}, args...)...)
		cmd.Dir, cmd.Env = repo, env
		out, err := cmd.Output()
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return r, fmt.Errorf("tpm doctor: %w: %s", err, exit.Stderr)
		}
		if err != nil {
			return r, err
		}
		return r, json.Unmarshal(out, &r)
	}
	expectChanges := func(c *check, r scaffold.DoctorResult, run string) {
		got := map[string]string{}
		for _, f := range r.Files {
			got[f.Path] = f.Action
		}
		c.expect(r.Level == "system", "%s: level %q", run, r.Level)
		c.expect(got["README.md"] == "created" && got[scaffold.AgentsFile] == "appended", "%s: root files %v", run, got)
		c.expect(got["docs/specs/epics/README.md"] == "refreshed", "%s: epics README %q", run, got["docs/specs/epics/README.md"])
		c.expect(slices.Equal(r.Repair.Repaired, []string{"docs/VISION.md"}) && r.Repair.Skipped == "", "%s: repair %+v", run, r.Repair)
		c.expect(got["control/labels.md"] == "extended", "%s: control/labels.md %q", run, got["control/labels.md"])
	}

	c := &check{}
	dry, err := doctor("--dry-run")
	if err != nil {
		return err
	}
	expectChanges(c, dry, "dry run")
	_, statErr := os.Stat(filepath.Join(repo, "README.md"))
	b, _ := os.ReadFile(epics)
	v, _ := os.ReadFile(vision)
	ag, _ := os.ReadFile(filepath.Join(repo, scaffold.AgentsFile))
	lb, _ := os.ReadFile(labelsPath)
	c.expect(errors.Is(statErr, os.ErrNotExist) && string(b) == older && string(v) != string(canonical) && string(ag) == ownAgents && string(lb) == olderLabels, "the dry run wrote")

	first, err := doctor()
	if err != nil {
		return err
	}
	expectChanges(c, first, "first run")
	expectOrientation(c, repo, []string{"themes", "initiatives", "epics"}, []string{"features", "stories"})
	b, _ = os.ReadFile(epics)
	c.expect(strings.HasPrefix(string(b), "owner's note above\n") && strings.HasSuffix(string(b), "\nowner's note below\n") &&
		!strings.Contains(string(b), "an older tpm's text") && strings.Contains(string(b), "`epNNN-short-name.md`"),
		"refreshed epics README:\n%s", b)
	v, _ = os.ReadFile(vision)
	c.expect(string(v) == string(canonical), "VISION not repaired to its canonical form")
	a, _ := os.ReadFile(filepath.Join(repo, scaffold.AgentsFile))
	c.expect(strings.HasPrefix(string(a), ownAgents+"\n<!-- tpm:begin"), "AGENTS.md: the owner's text not first:\n%s", a)
	lb, _ = os.ReadFile(labelsPath)
	c.expect(strings.Contains(string(lb), "<!-- field: op -->") && strings.Contains(string(lb), "| recur |") && strings.Contains(string(lb), "| build | #000000 |"),
		"control/labels.md not extended, or the recolor lost:\n%s", lb)
	if err := sh("git", "add", "-A"); err != nil {
		return err
	}
	c.expect(runCorpusAudit(repo, runStreaming) == nil, "the parent's corpus audit fails after tpm doctor")
	c.expect(specCheckAt(repo) == nil, "the spec tool's check fails after tpm doctor")

	second, err := doctor()
	if err != nil {
		return err
	}
	for _, f := range second.Files {
		c.expect(f.Action == "kept", "second run: %s %s", f.Path, f.Action)
	}
	c.expect(len(second.Repair.Repaired) == 0 && len(second.Repair.Findings) == 0, "second run repaired: %+v", second.Repair)
	return c.err("doctor")
}
