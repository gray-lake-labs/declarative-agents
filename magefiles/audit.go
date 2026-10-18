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
const parentRepository = "github.com/Nokia-Bell-Labs/declarative-agents"

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

func runCorpusAudit(root string, run commandRunner) error {
	tag, err := pinnedRelease(root)
	if err != nil {
		return err
	}
	cache, err := parentCheckout(root)
	if err != nil {
		return err
	}
	src, binary, err := ensureAgent(cache, tag, run)
	if err != nil {
		return err
	}
	out, err := os.MkdirTemp("", "scrum-master-audit-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(out)
	err = run(root, binary,
		"--profile", filepath.Join(src, auditProfile),
		"--directory", root,
		"--core-root", filepath.Join(src, "agent-core"),
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

// parentCheckout resolves the clone cache: the override in
// docs/external-checkouts.yaml, else the sibling checkout named after the
// repository.
func parentCheckout(root string) (string, error) {
	var checkouts struct {
		Checkouts map[string]string `yaml:"checkouts"`
	}
	path := filepath.Join(root, checkoutsFile)
	if _, err := os.Stat(path); err == nil {
		if err := readYAML(path, &checkouts); err != nil {
			return "", err
		}
	}
	rel := checkouts.Checkouts[parentRepository]
	if rel == "" {
		rel = filepath.Join("..", filepath.Base(parentRepository))
	}
	dir := filepath.Clean(filepath.Join(root, rel))
	if filepath.IsAbs(rel) {
		dir = filepath.Clean(rel)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return "", fmt.Errorf("parent clone cache %s is missing; build it as docs/guides/corpus-audit.md describes", dir)
	}
	return dir, nil
}

// releaseDir is where the pinned release is extracted and its tools built,
// once per tag, under the user cache directory.
func releaseDir(tag string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "scrum-master", "agent", tag), nil
}

// extractRelease writes the pinned release's source from the clone cache to
// dir/src, unless it is there already.
func extractRelease(cache, tag, dir string, run commandRunner) (string, error) {
	src := filepath.Join(dir, "src")
	if _, err := os.Stat(filepath.Join(src, "agent-core", "go.mod")); err == nil {
		return src, nil
	}
	if err := run(cache, "git", "show-ref", "--verify", "--quiet", "refs/tags/"+tag); err != nil {
		return "", fmt.Errorf("tag %s is not in the clone cache %s; fetch it with git -C %s fetch --tags origin", tag, cache, cache)
	}
	if err := os.RemoveAll(src); err != nil {
		return "", err
	}
	if err := os.MkdirAll(src, 0o755); err != nil {
		return "", err
	}
	cache, err := filepath.Abs(cache)
	if err != nil {
		return "", err
	}
	archive := filepath.Join(dir, "release.tar")
	defer os.Remove(archive)
	if err := run(cache, "git", "archive", "-o", archive, tag); err != nil {
		return "", fmt.Errorf("archive %s from %s: %w", tag, cache, err)
	}
	if err := run(src, "tar", "-xf", archive); err != nil {
		return "", fmt.Errorf("extract %s: %w", tag, err)
	}
	if _, err := os.Stat(filepath.Join(src, "agent-core", "go.mod")); err != nil {
		return "", fmt.Errorf("the extracted release %s has no agent-core/go.mod: %w", tag, err)
	}
	return src, nil
}

// ensureAgent builds the pinned release's agent once per tag.
func ensureAgent(cache, tag string, run commandRunner) (src, binary string, err error) {
	dir, err := releaseDir(tag)
	if err != nil {
		return "", "", err
	}
	binary = filepath.Join(dir, "agent")
	if _, err := os.Stat(binary); err == nil {
		return filepath.Join(dir, "src"), binary, nil
	}
	if src, err = extractRelease(cache, tag, dir, run); err != nil {
		return "", "", err
	}
	if err := run(filepath.Join(src, "agent-core"), "go", "build", "-tags", "production", "-o", binary, "./cmd/agent"); err != nil {
		return "", "", fmt.Errorf("build the specification-critic agent at %s: %w", tag, err)
	}
	return src, binary, nil
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
