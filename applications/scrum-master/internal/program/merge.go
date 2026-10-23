// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package program

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/events"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/sprintgoal"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/tracker"
)

// MergeResult is what merge reports.
type MergeResult struct {
	Project      string `json:"project"`
	IssueProject string `json:"issue_project,omitempty"`
	IID          int    `json:"issue"`
	Branch       string `json:"branch"`
	Commit       string `json:"commit"`
	Audit        bool   `json:"audit_run"` // the branch changes docs/, so the audit ran
	AuditCommand string `json:"audit_command"`
	// AuditReason says why the audit ran or did not: "docs/ changed" or
	// "no docs/ change" (srd013 R5.3).
	AuditReason string   `json:"audit_reason"`
	ApprovedBy  string   `json:"approved_by,omitempty"`
	Merged      bool     `json:"merged"` // merge made the merge commit now
	Closed      []string `json:"closed,omitempty"`
	// AlreadyClosed are --closes issues another merge closed first.
	AlreadyClosed []string `json:"already_closed,omitempty"`
	// Reported are the Sprint Goals the merge closed as reported complete,
	// each with a reported_complete event on its ledger (srd038 R2.2).
	Reported []string `json:"reported_complete,omitempty"`
	// ToClose is what merge-branch leaves for close-delivered, the --closes
	// references comma-joined (empty when none, never omitted, so a
	// machine's close word resolves it from the merge word's output; GH-201).
	ToClose string `json:"closes"`
	DryRun  bool   `json:"dry_run"`
}

// MergedByTrailer names, on a merge commit, the tracker user who ran the
// merge, whatever git identity made the commit: reports attribute merges by
// it (srd018).
const MergedByTrailer = "Merged-by:"

// Merge is delivery on the beads tracker, which delivers by local merge
// (srd042 R2.3): the gates and the merge of MergeBranch, then the closes of
// CloseDelivered — the two words the merge profile composes, so the command
// and the machine run the same code (GH-201, srd013 R5.7). A branch already
// merged is not merged again; its issues still close.
func Merge(t tracker.Tracker, git Git, project, issueProject string, iid int, branch, dir string, gate MergeGate, closes []string, dryRun bool) (MergeResult, error) {
	r, err := MergeBranch(t, git, project, issueProject, iid, branch, dir, gate, closes, dryRun)
	if err != nil || dryRun {
		return r, err
	}
	r.ToClose = ""
	r.Closed, r.Reported, r.AlreadyClosed, err = CloseDelivered(t, git, dir, r.Commit, issueProject, iid, closes)
	return r, err
}

// MergeBranch is the merge word (GH-201, srd013 R5.7): the gates — the
// issue and each closes open, commits of its own, the sprint-end summary
// format, the corpus audit when docs/ changes, and, when the gate requires
// it, another person's approval of the commit it merges (srd013 R5.5; an
// approval given without the requirement is still reported) — then the
// --no-ff merge into the default branch of the repository's main checkout,
// with a message citing the issue and each issue it closes, pushed. It
// closes nothing: close-delivered does, on the commit this word reports.
func MergeBranch(t tracker.Tracker, git Git, project, issueProject string, iid int, branch, dir string, gate MergeGate, closes []string, dryRun bool) (MergeResult, error) {
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

	// A reference already closed is reported, not cited: the merge closes
	// only what is open, and every cited reference reads as closed by it.
	cite := fmt.Sprintf("%s#%d", issueProject, iid)
	for _, ref := range closes {
		if !slices.Contains(already, ref) {
			cite += ", closes " + ref
		}
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
		r.AuditCommand, r.AuditReason = gate.Command, "no docs/ change"
		if r.Audit {
			r.AuditReason = "docs/ changed"
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
	_ = others
	r.ToClose = strings.Join(closes, ",")
	return r, nil
}

// CloseDelivered is the close word (GH-201, srd013 R5.7): on the merge
// commit the merge word reports, it closes the issue and each --closes
// reference that is still open, posts a reported_complete event on each
// closed Sprint Goal's ledger, and reports complete a Sprint Goal another
// merge closed first — its ledger keeps the event that merge posted, or
// gets one now; one closed as met is not (srd038 R2.2). A second run
// changes nothing: a closed issue stays closed on its commit.
func CloseDelivered(t tracker.Tracker, git Git, dir, commit, issueProject string, iid int, closes []string) (closed, reported, alreadyClosed []string, err error) {
	others, already, err := checkCloses(t, closes)
	if err != nil {
		return nil, nil, nil, err
	}
	out, err := git.Run(dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("close-delivered needs a local checkout of %s: %w", issueProject, err)
	}
	main := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(out), "/"), "/.git")
	targets := append([][2]any{{issueProject, iid}}, others...)
	for _, tgt := range targets {
		p, n := tgt[0].(string), tgt[1].(int)
		c, err := Close(t, git, p, n, main, commit, false)
		if err != nil {
			return closed, reported, already, fmt.Errorf("merged %s, but %s#%d did not close: %w", commit, p, n, err)
		}
		if c.Closed {
			closed = append(closed, p+"#"+strconv.Itoa(n))
		}
	}
	for _, tgt := range others {
		p, n := tgt[0].(string), tgt[1].(int)
		i, err := t.GetIssue(p, n)
		if err != nil {
			return closed, reported, already, err
		}
		m, ok := sprintgoal.Parse(i.Description)
		if !ok {
			continue
		}
		e, text := events.ReportedComplete(m.Release, commit)
		if _, err := events.Post(t, p, n, e, events.ReportedCompleteRequired, text, false); err != nil {
			return closed, reported, already, fmt.Errorf("closed %s#%d, but its ledger has no reported_complete event: %w", p, n, err)
		}
		reported = append(reported, p+"#"+strconv.Itoa(n))
	}
	for _, ref := range already {
		was, err := reportClosed(t, ref, commit)
		if err != nil {
			return closed, reported, already, err
		}
		if was {
			reported = append(reported, ref)
		}
	}
	return closed, reported, already, nil
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
