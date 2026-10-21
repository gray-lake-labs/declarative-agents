package scaffold

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gray-lake-labs/scrum-master"
	"github.com/gray-lake-labs/scrum-master/internal/beads"
	"github.com/gray-lake-labs/scrum-master/internal/install"
	"github.com/gray-lake-labs/scrum-master/internal/program"
)

// InitOptions are what tpm init is given.
type InitOptions struct {
	Dir     string
	Agents  []string // agents to install the skills for; none installs none
	Version string   // tpm version stamped into installed skills
	// Tracker is a tracker repository to seed; Group is the program group
	// whose labels are created there.
	Tracker, Group string
	// Prefix is the tracker's issue ID prefix; empty derives it from Group.
	Prefix        string
	User          string // the person the tracker's writes are made as
	Force, DryRun bool
}

// InitResult is what tpm init reports.
type InitResult struct {
	Dir     string           `json:"dir"`
	Files   []Result         `json:"files"`
	Skills  []install.Result `json:"skills,omitempty"`
	Tracker *TrackerResult   `json:"tracker,omitempty"`
	DryRun  bool             `json:"dry_run"`
}

// TrackerResult is what init did to the tracker repository (srd043 R1.5).
type TrackerResult struct {
	Remote string                `json:"remote"`
	Seeded bool                  `json:"seeded"`
	Prefix string                `json:"prefix,omitempty"` // the issue ID prefix the first commit set
	Reason string                `json:"reason,omitempty"`
	Labels []program.LabelResult `json:"labels,omitempty"`
}

// Init lays down a program's system repository in o.Dir: the corpus
// skeleton, the control files, the skills for o.Agents, and, with o.Tracker,
// the tracker repository's first commit and the program's labels (srd043 R1).
func Init(o InitOptions) (InitResult, error) {
	r := InitResult{Dir: o.Dir, DryRun: o.DryRun}
	if err := requireRepository(o.Dir); err != nil {
		return r, err
	}
	if o.Tracker != "" && o.Group == "" {
		return r, errors.New("--tracker needs --group, the program group whose labels it creates")
	}
	files, err := corpusFiles("system", o.Dir)
	if err != nil {
		return r, err
	}
	for n, b := range controlFiles() {
		files[n] = b
	}
	if r.Files, err = write(o.Dir, files, o.Force, o.DryRun); err != nil {
		return r, err
	}
	if len(o.Agents) > 0 {
		skills, err := install.Skills(scrummaster.Plugin, scrummaster.SkillsDir)
		if err != nil {
			return r, err
		}
		if r.Skills, err = install.Install(o.Dir, o.Agents, skills, o.Version, o.Force, o.DryRun); err != nil {
			return r, err
		}
	}
	switch {
	case o.Tracker != "":
		tr, err := seedTracker(o)
		r.Tracker = &tr
		if err != nil {
			return r, err
		}
	case !beads.Configured():
		r.Tracker = &TrackerResult{Reason: NoTracker}
	}
	return r, nil
}

// NoTracker is what init reports when it is given no tracker repository and
// TPM_TRACKER names none: the issue workflow cannot run until one exists.
const NoTracker = "no tracker repository seeded; issues cannot be filed until one exists: rerun with --tracker and --group"

// seedTracker makes an empty tracker repository's first commit, holding the
// config file with the issue prefix and the files layout, and creates the program's labels there. A tracker
// repository that already has commits is left as it is.
func seedTracker(o InitOptions) (TrackerResult, error) {
	r := TrackerResult{Remote: o.Tracker}
	tmp, err := os.MkdirTemp("", "tpm-init-tracker-*")
	if err != nil {
		return r, err
	}
	defer os.RemoveAll(tmp)
	clone := filepath.Join(tmp, "tracker")
	if err := git(tmp, "clone", "-q", o.Tracker, clone); err != nil {
		return r, err
	}
	if git(clone, "rev-parse", "--verify", "-q", "HEAD") == nil {
		r.Reason = "the tracker repository has commits; init left it as it is"
		return r, nil
	}
	if o.DryRun {
		r.Seeded, r.Reason = true, "dry run: would make the first commit and create the labels"
		return r, nil
	}
	if err := os.MkdirAll(filepath.Join(clone, filepath.Dir(beads.ConfigFile)), 0o755); err != nil {
		return r, err
	}
	r.Prefix = o.Prefix
	if r.Prefix == "" {
		r.Prefix = beads.PrefixFor(o.Group)
	}
	// A new tracker keeps one file per issue (srd042 R3.1, eng04).
	if err := beads.WriteConfig(clone, beads.Config{IssuePrefix: r.Prefix, Layout: beads.LayoutFiles}); err != nil {
		return r, err
	}
	for _, args := range [][]string{
		{"add", beads.ConfigFile},
		{"commit", "-q", "-m", "tpm init: the tracker repository's first commit"},
		{"push", "-q", "origin", "HEAD:main"},
	} {
		if err := git(clone, args...); err != nil {
			return r, err
		}
	}
	r.Seeded = true
	r.Labels, err = program.SyncLabels(beads.NewTracker(clone, o.User, o.Dir), o.Group, program.DefaultLabels(), false, false)
	return r, err
}

func git(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
