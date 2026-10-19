package program

import (
	"fmt"
	"slices"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// approvalPrefix opens the marker an approval comment carries; the commit
// the reviewer read completes it (srd013 R5.6).
const approvalPrefix = "<!-- tpm program approve commit="

func approvalMarker(commit string) string { return approvalPrefix + commit + " -->" }

// ApproveResult is what approve reports.
type ApproveResult struct {
	Project      string `json:"project"`
	IssueProject string `json:"issue_project,omitempty"`
	IID          int    `json:"issue"`
	Branch       string `json:"branch"`
	Commit       string `json:"commit"`
	By           string `json:"by"`
	Posted       bool   `json:"posted"` // approve posted the comment now
}

// Approve posts a reviewer's approval of a branch on its issue, naming the
// tip of the branch at its origin, the commit the reviewer read (srd013
// R5.6). The issue's assignee cannot approve their own work, and a second
// approval of the same commit by the same person posts nothing. branch
// defaults to the issue's pop branch; dir is any checkout of project.
func Approve(t tracker.Tracker, git Git, project, issueProject string, iid int, branch, dir string) (ApproveResult, error) {
	r := ApproveResult{Project: project, IID: iid}
	if issueProject != project {
		r.IssueProject = issueProject
	}
	i, err := t.GetIssue(issueProject, iid)
	if err != nil {
		return r, err
	}
	if i.State == "closed" {
		return r, fmt.Errorf("%s#%d is closed", issueProject, iid)
	}
	me, err := t.CurrentUser()
	if err != nil {
		return r, err
	}
	r.By = me.Username
	if slices.Contains(i.Assignees, me.Username) {
		return r, fmt.Errorf("%s#%d is assigned to %s; an approval comes from someone other than the issue's assignee (srd013 R5.6)", issueProject, iid, me.Username)
	}
	r.Branch = branch
	if r.Branch == "" {
		r.Branch = fmt.Sprintf("%d-%s", iid, Slug(i.Title))
	}
	git.Run(dir, "fetch", "-q", "origin", r.Branch)
	for _, ref := range []string{"refs/remotes/origin/" + r.Branch, "refs/heads/" + r.Branch} {
		if out, err := git.Run(dir, "rev-parse", "--verify", "-q", ref); err == nil {
			r.Commit = strings.TrimSpace(out)
			break
		}
	}
	if r.Commit == "" {
		return r, fmt.Errorf("no branch %s to approve in %s", r.Branch, project)
	}
	notes, err := t.ListNotes(issueProject, iid)
	if err != nil {
		return r, err
	}
	for _, n := range notes {
		if n.Author == me.Username && strings.Contains(n.Body, approvalMarker(r.Commit)) {
			return r, nil
		}
	}
	r.Posted = true
	return r, t.CreateNote(issueProject, iid, fmt.Sprintf("Approved %s at %s for merge.\n\n%s", r.Branch, r.Commit, approvalMarker(r.Commit)))
}

// approval finds who approved the issue's branch at tip, other than the
// assignees or, for an issue nobody took, other than merger (srd013 R5.5).
// When nobody did, the error names what is missing, and the commit an
// earlier approval named, if any.
func approval(t tracker.Tracker, issueProject string, iid int, branch, tip, merger string) (string, error) {
	i, err := t.GetIssue(issueProject, iid)
	if err != nil {
		return "", err
	}
	excluded := i.Assignees
	if len(excluded) == 0 {
		excluded = []string{merger}
	}
	notes, err := t.ListNotes(issueProject, iid)
	if err != nil {
		return "", err
	}
	stale := ""
	for _, n := range notes {
		if slices.Contains(excluded, n.Author) {
			continue
		}
		if strings.Contains(n.Body, approvalMarker(tip)) {
			return n.Author, nil
		}
		if at := strings.Index(n.Body, approvalPrefix); at >= 0 {
			stale, _, _ = strings.Cut(n.Body[at+len(approvalPrefix):], " ")
		}
	}
	short := func(c string) string { return c[:min(12, len(c))] }
	if stale != "" {
		return "", fmt.Errorf("the approval of %s#%d is for %s, and %s is now at %s; ask for a new approval with tpm issue approve (srd013 R5.5)",
			issueProject, iid, short(stale), branch, short(tip))
	}
	return "", fmt.Errorf("no approval of %s at %s from someone other than %s; ask a reviewer to run tpm issue approve (srd013 R5.5)",
		branch, short(tip), strings.Join(excluded, ", "))
}
