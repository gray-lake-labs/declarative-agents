package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/spectool"
)

// parentRepository is the parent corpus whose specification-critic agent
// audits this repository (docs/guides/corpus-audit.md).
const parentRepository = "github.com/gray-lake-labs/declarative-agents"

const (
	pinsFile      = "docs/external-pins.yaml"
	checkoutsFile = "docs/external-checkouts.yaml"
	auditProfile  = "applications/catalog/agents/specification-critic/corpus-audit-profile.yaml"
)

type commandRunner func(dir string, name string, args ...string) error

// Audit runs the repository gate tests, the parent's specification-critic
// corpus-audit agent over the repository, and the runtime's --validate-config
// over every shipped profile (GH-135).
func Audit() error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	if err := runStreaming(root, "go", "test", "./magefiles/...", "-count=1"); err != nil {
		return fmt.Errorf("repository gate tests: %w", err)
	}
	if err := runCorpusAudit(root, runStreaming); err != nil {
		return err
	}
	return validateProfiles(root, runStreaming)
}

// validateProfiles holds every shipped declarative-agents profile under
// plugin/agents to the pinned runtime: the same boot smoke a rollout runs,
// so a profile the runtime would refuse fails the gate (GH-135, srd036 R6.1).
func validateProfiles(root string, run commandRunner) error {
	profiles, err := filepath.Glob(filepath.Join(root, "plugin", "agents", "*", "profile.yaml"))
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		return nil
	}
	tag, err := pinnedRelease(root)
	if err != nil {
		return err
	}
	rel, err := fetchRelease(tag)
	if err != nil {
		return err
	}
	for _, p := range profiles {
		if err := run(root, rel.Agent, "--profile", p, "--validate-config", "--core-root", rel.CoreRoot()); err != nil {
			return fmt.Errorf("profile %s fails the pinned runtime's validation: %w", p, err)
		}
	}
	return nil
}

// runCorpusAudit runs the pinned release's agent over root, with derived_from
// entries resolving against the release's own source.
func runCorpusAudit(root string, run commandRunner) error {
	tag, err := pinnedRelease(root)
	if err != nil {
		return err
	}
	rel, err := fetchRelease(tag)
	if err != nil {
		return err
	}
	restore, err := withCheckouts(root, rel.Src, parentRepository)
	if err != nil {
		return err
	}
	defer restore()
	out, err := os.MkdirTemp("", "scrum-master-audit-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(out)
	err = run(root, rel.Agent,
		"--profile", filepath.Join(rel.Src, auditProfile),
		"--directory", root,
		"--core-root", filepath.Join(rel.Src, "agent-core"),
		"--output", out,
	)
	if err != nil {
		return fmt.Errorf("specification corpus audit failed: %w", err)
	}
	return nil
}

// pinnedRelease reads the parent release docs/external-pins.yaml tracks.
func pinnedRelease(root string) (string, error) { return spectool.Pinned(root) }

func repositoryRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", errors.New("not inside a git checkout")
	}
	return strings.TrimSpace(string(out)), nil
}

func runStreaming(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
