package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
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

// Audit runs the repository gate tests, then the parent's specification-critic
// corpus-audit agent over the repository.
func Audit() error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	if err := runStreaming(root, "go", "test", "./magefiles/...", "-count=1"); err != nil {
		return fmt.Errorf("repository gate tests: %w", err)
	}
	return runCorpusAudit(root, runStreaming)
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
func pinnedRelease(root string) (string, error) {
	var pins struct {
		Pins map[string]string `yaml:"pins"`
	}
	if err := readYAML(filepath.Join(root, pinsFile), &pins); err != nil {
		return "", err
	}
	tag := pins.Pins[parentRepository]
	if tag == "" {
		return "", fmt.Errorf("%s pins no release of %s", pinsFile, parentRepository)
	}
	return tag, nil
}

func readYAML(path string, into any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, into); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

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
