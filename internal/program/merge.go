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
	AuditCommand string   `json:"audit_command,omitempty"`
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
// its own, the corpus audit passing when docs/ changes, and, when the gate
// requires it, another person's approval of the commit it merges (srd013
// R5.5); an approval given without the requirement is still reported. It
// then merges the branch --no-ff into the default branch of the repository's main checkout with a message
// citing the issue and each issue it closes, pushes it, and closes each on
// that merge. A branch already merged is not merged again; its issues
// still close.
func Merge(t tracker.Tracker, git Git, project, issueProject string, iid int, branch, dir string, gate MergeGate, closes []string, dryRun bool) (MergeResult, error) {
	r := MergeResult{Project: project, IID: iid, Branch: branch, DryRun: dryRun}
	if _, ok := tracker.For(t, issueProject).(MergeCloser); !ok {
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
	// The base is what origin holds now: the main checkout's own branch may
	// be behind it, and a diff against that would count others' merges as
	// this branch's changes.
	upstream := "refs/heads/" + base
	if _, err := inMain("rev-parse", "--verify", "-q", "refs/remotes/origin/"+base); err == nil {
		upstream = "refs/remotes/origin/" + base
	}
	if _, err := inMain("merge-base", "--is-ancestor", "refs/heads/"+branch, upstream); err == nil {
		exact := "--grep=" + regexp.QuoteMeta(fmt.Sprintf("%s#%d", issueProject, iid)) + "([^0-9]|$)"
		r.Commit, _ = inMain("rev-list", "-1", "--merges", "-E", exact, upstream)
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
		paths, err := inMain("diff", "--name-only", upstream+"...refs/heads/"+branch)
		if err != nil {
			return r, err
		}
		for _, p := range strings.Split(paths, "\n") {
			if p == "docs" || strings.HasPrefix(p, "docs/") {
				r.Audit = true
			}
			if path.Dir(p) == sprintgoal.SummaryDir && path.Ext(p) == ".md" {
				// Through textconv, as GetFile reads: in a git-crypt
				// repository the blob is ciphertext.
				summary, err := inMain("cat-file", "--textconv", "refs/heads/"+branch+":"+p)
				if err != nil {
					continue // the branch deletes it
				}
				if problems := sprintgoal.CheckSummary(summary); len(problems) > 0 {
					return r, fmt.Errorf("the sprint-end summary %s is not in its format (design.md sprint_end_summary): %s; nothing merged", p, strings.Join(problems, "; "))
				}
			}
		}
		if r.Audit {
			r.AuditCommand = gate.Command
			if err := gate.checkBefore("nothing merged"); err != nil {
				return r, err
			}
		}
		me, err := tracker.UserIn(t, issueProject)
		if err != nil {
			return r, err
		}
		tip, err := inMain("rev-parse", "refs/heads/"+branch)
		if err != nil {
			return r, err
		}
		d := deliveryFor(t, git, project)
		by, err := d.approval(issueProject, iid, branch, tip, me.Username)
		if err != nil && gate.RequireApproval {
			return r, err
		}
		if err == nil {
			r.ApprovedBy = by
		}
		if dryRun {
			return r, nil
		}
		if r.Commit, err = d.merge(main, base, branch, fmt.Sprintf("Merge %s (%s)\n\n%s %s", branch, cite, MergedByTrailer, me.Username)); err != nil {
			return r, err
		}
		r.Merged = true
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
	// A Sprint Goal another merge closed first, as a team's sprint-end merge
	// does before the program's, is still reported complete here: its
	// ledger keeps the reported_complete event that merge posted, or gets
	// one now; one closed as met is not (srd038 R2.2).
	for _, ref := range r.AlreadyClosed {
		reported, err := reportClosed(t, ref, r.Commit)
		if err != nil {
			return r, err
		}
		if reported {
			r.Reported = append(r.Reported, ref)
		}
	}
	return r, nil
}

// reportClosed reports whether the closed issue ref is a Sprint Goal
// reported complete, posting its reported_complete event when its ledger
// has neither that nor a met event.
func reportClosed(t tracker.Tracker, ref, commit string) (bool, error) {
	p, n, ok := strings.Cut(ref, "#")
	iid, err := strconv.Atoi(n)
	if !ok || err != nil {
		return false, nil
	}
	i, err := t.GetIssue(p, iid)
	if err != nil {
		return false, err
	}
	m, ok := sprintgoal.Parse(i.Description)
	if !ok {
		return false, nil
	}
	notes, err := t.ListNotes(p, iid)
	if err != nil {
		return false, err
	}
	met, reported := false, false
	for _, note := range notes {
		if e, ok := events.Parse(note.Body); ok {
			met = met || e.Name == "met"
			reported = reported || e.Name == "reported_complete"
		}
	}
	if met || reported {
		return !met, nil
	}
	e, text := events.ReportedComplete(m.Release, commit)
	if _, err := events.Post(t, p, iid, e, events.ReportedCompleteRequired, text, false); err != nil {
		return false, fmt.Errorf("%s was closed, but its ledger has no reported_complete event: %w", ref, err)
	}
	return true, nil
}
