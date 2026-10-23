package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/runtime"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

func init() {
	integrationTargets = append(integrationTargets, Integration{}.PopProfile)
}

// PopProfile runs the pinned runtime over the pop profile in the rig
// (GH-204, srd013 R7.5) — a model-free machine: an admitted issue claims,
// opens its branch and worktree, and installs the skills through the four
// tpm words, ending in Done; a closed issue ends in Refused with tpm's
// reason (test: pop profile run, pop profile refusal; environment: local).
func (Integration) PopProfile() error {
	r, err := newRigEnv()
	if err != nil {
		return err
	}
	po1, err := r.project("product-owners/po-1")
	if err != nil {
		return err
	}
	if err := r.milestone(rigMilestone, rigDue); err != nil {
		return err
	}
	if _, err := r.runTPM("issue", "labels", "--group", r.group); err != nil {
		return err
	}
	tag, err := pinnedRelease(r.root)
	if err != nil {
		return err
	}
	rel, err := fetchRelease(tag)
	if err != nil {
		return err
	}
	clone, err := r.clone(po1, "po-1")
	if err != nil {
		return err
	}
	if _, err := r.runTPM("install", "--agent", "claude", "--dir", clone); err != nil {
		return err
	}
	file := func(title string) (program.FileResult, error) {
		path := filepath.Join(r.run.Dir, "pop-"+program.Slug(title)+".md")
		writeFileTo(path, "## Requirements\n\n1. Add the note.\n\n## Acceptance Criteria\n\n- The note exists (test: note_exists; environment: local)\n")
		var res program.FileResult
		return res, r.tpmJSON(&res, "issue", "file", "--project", po1, "--title", title, "--body-file", path,
			"--plane", "data", "--kind", "build", "--component", "po-1", "--milestone", rigMilestone)
	}
	c := &check{}

	// The full pop: Claim → Branch → Worktree → Install → Done.
	work, err := file("Pop profile run")
	if err != nil {
		return err
	}
	out, runErr := r.runPopAgent(rel, po1, clone, work.IID)
	c.expect(runErr == nil, "the pop run failed: %v: %s", runErr, tail17(out))
	c.expect(strings.Contains(out, "final machine state: Done"), "the run must end in Done: %s", tail17(out))
	worktree := filepath.Join(filepath.Dir(clone), filepath.Base(clone)+fmt.Sprintf("-%d-%s", work.IID, program.Slug("Pop profile run")))
	_, serr := os.Stat(filepath.Join(worktree, ".claude", "skills", "implement", "SKILL.md"))
	c.expect(serr == nil, "the worktree %s must exist with the skills installed: %v", worktree, serr)
	i, err := r.issue(po1, work.IID)
	c.expect(err == nil && len(i.Assignees) == 1 && i.Assignees[0] == r.user, "assignees after the pop: %v %v", i.Assignees, err)

	// The refusal: a closed issue ends in Refused with tpm's reason.
	gone, err := file("Pop profile refusal")
	if err != nil {
		return err
	}
	closeState := "close"
	if _, err := r.tr.UpdateIssue(po1, gone.IID, tracker.IssueUpdate{State: &closeState}); err != nil {
		return err
	}
	out, runErr = r.runPopAgent(rel, po1, clone, gone.IID)
	c.expect(runErr != nil, "a refused run must not report success: %s", tail17(out))
	c.expect(strings.Contains(out, "final machine state: Refused"), "the run must end in Refused: %s", tail17(out))
	c.expect(strings.Contains(out, "is closed"), "the Refused state must carry tpm's reason: %s", tail17(out))
	return c.err("popProfile")
}

// runPopAgent runs the release's agent over the pop profile for one issue,
// from the project's clone, with the rig's tracker and the built tpm on
// PATH — the environment the profile's declarations name.
func (r *rigEnv) runPopAgent(rel runtime.Release, project, clone string, iid int) (string, error) {
	outDir, err := os.MkdirTemp(r.run.Dir, "pop-agent-*")
	if err != nil {
		return "", err
	}
	cmd := exec.Command(rel.Agent,
		"--profile", filepath.Join(r.root, "plugin", "agents", "pop", "profile.yaml"),
		"--core-root", rel.CoreRoot(),
		"--directory", clone,
		"--output", outDir,
	)
	cmd.Dir = clone
	cmd.Env = append(os.Environ(),
		"PATH="+r.run.Dir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"TPM_TRACKER="+filepath.Join(r.run.Dir, "tracker.git"),
		"TPM_TRACKER_CLONE="+filepath.Join(r.run.Dir, "tracker-"+r.user),
		"TPM_USER="+r.user,
		"TPM_WEB_BASE="+r.webBase,
		"TPM_PROJECT="+project,
		"TPM_ISSUE="+fmt.Sprint(iid),
	)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err = cmd.Run()
	return out.String(), err
}

// tail17 is the last part of a run's output, enough to carry its terminal
// state and reason into a finding.
func tail17(out string) string {
	const n = 800
	if len(out) <= n {
		return out
	}
	return "…" + out[len(out)-n:]
}
