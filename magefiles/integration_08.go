package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/hierarchy"
	"github.com/gray-lake-labs/scrum-master/internal/objectives"
	"github.com/gray-lake-labs/scrum-master/internal/scaffold"
)

func init() {
	integrationTargets = append(integrationTargets, Integration{}.Init)
}

// Init builds tpm, inits an empty git repository with an empty tracker repository beside it, and proves the parent's corpus audit, the objective gate, and the hierarchy gate pass on the result as written, the orientation files are written for the system level, the tracker got its first commit and labels, and a second init changes nothing (test-rel01.4-agent-legible-cli, test-rel04.1-orientation-doctor). It needs no rig.
func (Integration) Init() error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "tpm-init-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if tmp, err = filepath.EvalSymlinks(tmp); err != nil {
		return err
	}
	bin, repo, tracker := filepath.Join(tmp, "tpm"), filepath.Join(tmp, "system"), filepath.Join(tmp, "tracker.git")
	if err := runStreaming(root, "go", "build", "-o", bin, "./cmd/tpm"); err != nil {
		return fmt.Errorf("build tpm: %w", err)
	}
	env := append(os.Environ(), "TPM_USER=operator",
		"GIT_AUTHOR_NAME=operator", "GIT_AUTHOR_EMAIL=operator@example.invalid",
		"GIT_COMMITTER_NAME=operator", "GIT_COMMITTER_EMAIL=operator@example.invalid")
	sh := func(dir string, name string, args ...string) error {
		cmd := exec.Command(name, args...)
		cmd.Dir, cmd.Env = dir, env
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, out)
		}
		return nil
	}
	if err := sh(tmp, "git", "init", "-q", "-b", "main", repo); err != nil {
		return err
	}
	if err := sh(tmp, "git", "init", "-q", "--bare", "-b", "main", tracker); err != nil {
		return err
	}
	initRepo := func() (scaffold.InitResult, error) {
		var r scaffold.InitResult
		cmd := exec.Command(bin, "init", "--json", "--dir", repo, "--agent", "claude", "--tracker", tracker, "--group", "rig")
		cmd.Dir, cmd.Env = repo, env
		out, err := cmd.Output()
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return r, fmt.Errorf("tpm init: %w: %s", err, exit.Stderr)
		}
		if err != nil {
			return r, err
		}
		return r, json.Unmarshal(out, &r)
	}
	c := &check{}
	first, err := initRepo()
	if err != nil {
		return err
	}
	created := 0
	for _, f := range first.Files {
		if f.Action == "created" {
			created++
		}
	}
	c.expect(created == len(first.Files) && created > 0, "first init: %+v", first.Files)
	c.expect(len(first.Skills) > 0, "first init installed no skills")
	c.expect(first.Tracker != nil && first.Tracker.Seeded && len(first.Tracker.Labels) > 0, "tracker not seeded: %+v", first.Tracker)
	expectOrientation(c, repo, []string{"themes", "initiatives", "epics"}, []string{"features", "stories"})

	if err := sh(repo, "git", "add", "-A"); err != nil {
		return err
	}
	c.expect(runCorpusAudit(repo, runStreaming) == nil, "the parent's corpus audit fails on the corpus tpm init wrote")
	c.expect(specCheckAt(repo) == nil, "the spec tool's check fails on the corpus tpm init wrote")
	docs, err := objectives.LoadDir(filepath.Join(repo, objectives.Dir))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	for _, d := range docs {
		c.expect(len(objectives.Validate(d)) == 0, "objective gate fails on %s", d.Path)
	}

	hdocs, err := hierarchy.Load(repo)
	c.expect(err == nil && len(hierarchy.Validate(hdocs)) == 0, "the hierarchy gate fails on the corpus tpm init wrote: %v", err)

	second, err := initRepo()
	if err != nil {
		return err
	}
	for _, f := range second.Files {
		c.expect(f.Action == "kept", "second init: %s %s", f.Path, f.Action)
	}
	c.expect(second.Tracker != nil && !second.Tracker.Seeded, "second init seeded the tracker again: %+v", second.Tracker)
	return c.err("init")
}

// expectOrientation holds a repository tpm laid down to its orientation
// files: README.md and AGENTS.md at the root, a README in each hierarchy
// folder of has and in none of lacks, each one tpm block, and AGENTS.md
// carrying the Mermaid figure rule (srd043 R1.6, R2.4).
func expectOrientation(c *check, dir string, has, lacks []string) {
	paths := []string{"README.md", scaffold.AgentsFile}
	for _, f := range has {
		paths = append(paths, filepath.Join("docs", "specs", f, "README.md"))
	}
	for _, p := range paths {
		b, err := os.ReadFile(filepath.Join(dir, p))
		c.expect(err == nil && strings.Contains(string(b), "<!-- tpm:begin") && strings.Contains(string(b), scaffold.BlockEnd),
			"%s: no orientation file with a tpm block: %v", p, err)
	}
	for _, f := range lacks {
		_, err := os.Stat(filepath.Join(dir, "docs", "specs", f, "README.md"))
		c.expect(errors.Is(err, fs.ErrNotExist), "a %s README was written where the level holds no %s", f, f)
	}
	agents, _ := os.ReadFile(filepath.Join(dir, scaffold.AgentsFile))
	c.expect(strings.Contains(string(agents), "Mermaid in a fenced code block") && strings.Contains(string(agents), "- tpm doctor: "),
		"AGENTS.md lacks the figure rule or tpm doctor")
}
