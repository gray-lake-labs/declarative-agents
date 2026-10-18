package program

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
)

// MergeResult is what merge reports.
type MergeResult struct {
	Project      string   `json:"project"`
	IssueProject string   `json:"issue_project,omitempty"`
	IID          int      `json:"issue"`
	Branch       string   `json:"branch"`
	Commit       string   `json:"commit"`
	Audit        bool     `json:"audit_run"` // the branch changes docs/, so the audit ran
	Merged       bool     `json:"merged"`    // merge made the merge commit now
	Closed       []string `json:"closed,omitempty"`
	DryRun       bool     `json:"dry_run"`
}

// Merge is delivery on the beads tracker, which has no merge requests
// (srd042 R2.3): it holds the branch to what open-merge-request holds it to
// on GitLab, the issue open, each closes open, commits of its own, and the
// corpus audit passing when docs/ changes, then merges the branch --no-ff
// into the default branch of the repository's main checkout with a message
// citing the issue and each issue it closes, pushes it, and closes each on
// that merge. A branch already merged is not merged again; its issues
// still close.
func Merge(t gitlab.Tracker, git Git, project, issueProject string, iid int, branch, dir string, gate AuditGate, closes []string, dryRun bool) (MergeResult, error) {
	r := MergeResult{Project: project, IID: iid, Branch: branch, DryRun: dryRun}
	if _, ok := t.(MergeCloser); !ok {
		return r, fmt.Errorf("on GitLab the merge request is the delivery; use tpm program open-merge-request")
	}
	if issueProject != project {
		r.IssueProject = issueProject
	}
	others, err := checkCloses(t, closes)
	if err != nil {
		return r, err
	}
	base, err := t.DefaultBranch(project)
	if err != nil {
		return r, err
	}
	run := func(args ...string) (string, error) {
		out, err := git.Run(dir, args...)
		return strings.TrimSpace(out), err
	}
	common, err := run("rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return r, fmt.Errorf("merge needs a local checkout of %s: %w", project, err)
	}
	main := strings.TrimSuffix(strings.TrimSuffix(common, "/"), "/.git")
	inMain := func(args ...string) (string, error) {
		out, err := git.Run(main, args...)
		return strings.TrimSpace(out), err
	}

	cite := fmt.Sprintf("%s#%d", issueProject, iid)
	for _, ref := range closes {
		cite += ", closes " + ref
	}
	inMain("fetch", "-q", "origin", base)
	if _, err := inMain("merge-base", "--is-ancestor", "refs/heads/"+branch, "refs/heads/"+base); err == nil {
		exact := "--grep=" + regexp.QuoteMeta(fmt.Sprintf("%s#%d", issueProject, iid)) + "([^0-9]|$)"
		r.Commit, _ = inMain("rev-list", "-1", "--merges", "-E", exact, "refs/heads/"+base)
		if r.Commit == "" {
			return r, fmt.Errorf("%s has no commits over %s; commit the work first", branch, base)
		}
	} else {
		i, err := t.GetIssue(issueProject, iid)
		if err != nil {
			return r, err
		}
		if i.State == "closed" {
			return r, fmt.Errorf("%s#%d is closed", issueProject, iid)
		}
		paths, err := inMain("diff", "--name-only", "refs/heads/"+base+"...refs/heads/"+branch)
		if err != nil {
			return r, err
		}
		for _, p := range strings.Split(paths, "\n") {
			if p == "docs" || strings.HasPrefix(p, "docs/") {
				r.Audit = true
			}
		}
		if r.Audit {
			if err := gate.checkBefore("nothing merged"); err != nil {
				return r, err
			}
		}
		if dryRun {
			return r, nil
		}
		if current, _ := inMain("branch", "--show-current"); current != base {
			if _, err := inMain("checkout", "-q", base); err != nil {
				return r, err
			}
		}
		inMain("merge", "-q", "--ff-only", "refs/remotes/origin/"+base)
		if _, err := inMain("merge", "-q", "--no-ff", branch, "-m", fmt.Sprintf("Merge %s (%s)", branch, cite)); err != nil {
			return r, err
		}
		if _, err := inMain("push", "-q", "origin", base); err != nil {
			return r, fmt.Errorf("merged locally but could not push %s: %w", base, err)
		}
		r.Merged = true
		if r.Commit, err = inMain("rev-parse", "HEAD"); err != nil {
			return r, err
		}
	}
	if dryRun {
		return r, nil
	}
	targets := append([][2]any{{issueProject, iid}}, others...)
	for _, tgt := range targets {
		p, n := tgt[0].(string), tgt[1].(int)
		c, err := Close(t, git, p, n, main, false)
		if err != nil {
			return r, fmt.Errorf("merged %s, but %s#%d did not close: %w", r.Commit, p, n, err)
		}
		if c.Closed {
			r.Closed = append(r.Closed, p+"#"+strconv.Itoa(n))
		}
	}
	return r, nil
}
