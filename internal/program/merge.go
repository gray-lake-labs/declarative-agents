package program

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/events"
	"github.com/gray-lake-labs/scrum-master/internal/sprintgoal"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// MergeResult is what merge reports.
type MergeResult struct {
	Project      string   `json:"project"`
	IssueProject string   `json:"issue_project,omitempty"`
	IID          int      `json:"issue"`
	Branch       string   `json:"branch"`
	Commit       string   `json:"commit"`
	Audit        bool     `json:"audit_run"` // the branch changes docs/, so the audit ran
	ApprovedBy   string   `json:"approved_by,omitempty"`
	Merged       bool     `json:"merged"` // merge made the merge commit now
	Closed       []string `json:"closed,omitempty"`
	// AlreadyClosed are --closes issues another merge closed first.
	AlreadyClosed []string `json:"already_closed,omitempty"`
	// Reported are the Sprint Goals the merge closed as reported complete,
	// each with a reported_complete event on its ledger (srd038 R2.2).
	Reported []string `json:"reported_complete,omitempty"`
	DryRun   bool     `json:"dry_run"`
}

// MergedByTrailer names, on a merge commit, the tracker user who ran the
// merge, whatever git identity made the commit: reports attribute merges by
// it (srd018).
const MergedByTrailer = "Merged-by:"

// Merge is delivery on the beads tracker, which delivers by local merge
// (srd042 R2.3): it requires the issue open, each closes open, commits of
// its own, the corpus audit passing when docs/ changes, and another person's approval
// of the commit it merges (srd013 R5.5), then merges the branch --no-ff
// into the default branch of the repository's main checkout with a message
// citing the issue and each issue it closes, pushes it, and closes each on
// that merge. A branch already merged is not merged again; its issues
// still close.
func Merge(t tracker.Tracker, git Git, project, issueProject string, iid int, branch, dir string, gate AuditGate, closes []string, dryRun bool) (MergeResult, error) {
	r := MergeResult{Project: project, IID: iid, Branch: branch, DryRun: dryRun}
	if _, ok := t.(MergeCloser); !ok {
		return r, fmt.Errorf("this tracker does not close issues on a local merge; tpm issue merge serves the beads tracker")
	}
	if issueProject != project {
		r.IssueProject = issueProject
	}
	others, already, err := checkCloses(t, closes)
	if err != nil {
		return r, err
	}
	r.AlreadyClosed = already
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
			if path.Dir(p) == sprintgoal.SummaryDir && path.Ext(p) == ".md" {
				summary, err := inMain("show", "refs/heads/"+branch+":"+p)
				if err != nil {
					continue // the branch deletes it
				}
				if problems := sprintgoal.CheckSummary(summary); len(problems) > 0 {
					return r, fmt.Errorf("the sprint-end summary %s is not in its format (design.md sprint_end_summary): %s; nothing merged", p, strings.Join(problems, "; "))
				}
			}
		}
		if r.Audit {
			if err := gate.checkBefore("nothing merged"); err != nil {
				return r, err
			}
		}
		me, err := t.CurrentUser()
		if err != nil {
			return r, err
		}
		tip, err := inMain("rev-parse", "refs/heads/"+branch)
		if err != nil {
			return r, err
		}
		if r.ApprovedBy, err = approval(t, issueProject, iid, branch, tip, me.Username); err != nil {
			return r, err
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
		if _, err := inMain("merge", "-q", "--no-ff", branch, "-m", fmt.Sprintf("Merge %s (%s)\n\n%s %s", branch, cite, MergedByTrailer, me.Username)); err != nil {
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
	for _, tgt := range others {
		p, n := tgt[0].(string), tgt[1].(int)
		i, err := t.GetIssue(p, n)
		if err != nil {
			return r, err
		}
		m, ok := sprintgoal.Parse(i.Description)
		if !ok {
			continue
		}
		e, text := events.ReportedComplete(m.Release, r.Commit)
		if _, err := events.Post(t, p, n, e, events.ReportedCompleteRequired, text, false); err != nil {
			return r, fmt.Errorf("closed %s#%d, but its ledger has no reported_complete event: %w", p, n, err)
		}
		r.Reported = append(r.Reported, p+"#"+strconv.Itoa(n))
	}
	return r, nil
}
