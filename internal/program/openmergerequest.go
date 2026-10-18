package program

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
)

// AuditRunner runs the corpus gate command in a directory and returns its
// combined output. Tests replace it.
type AuditRunner func(dir, command string) (string, error)

// ShellAudit runs the command with sh -c.
func ShellAudit(dir, command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.String(), err
}

// AuditGate is the corpus gate open-merge-request runs on a branch that changes docs/:
// the command (mage audit by default, policy admission.audit_command) run
// in Dir, the worktree holding the branch.
type AuditGate struct {
	Command string
	Dir     string
	Run     AuditRunner
}

// OpenMergeRequestResult is what open-merge-request reports.
type OpenMergeRequestResult struct {
	Project string `json:"project"`
	// IssueProject holds the issue when it is not in Project.
	IssueProject string `json:"issue_project,omitempty"`
	IID          int    `json:"issue"`
	MR           int    `json:"merge_request"`
	WebURL       string `json:"web_url"`
	Created      bool   `json:"created"`
	Audit        bool   `json:"audit_named"` // the branch changes docs/, so mage audit is named
	DryRun       bool   `json:"dry_run"`
}

// OpenMergeRequest opens the merge request that closes an issue (srd013 R5). It
// refuses a closed issue and a branch with no commits over the default
// branch, and returns an open merge request for the branch instead of
// opening a second one.
func OpenMergeRequest(t gitlab.Tracker, project string, iid int, branch string, gate AuditGate, dryRun bool) (OpenMergeRequestResult, error) {
	return OpenMergeRequestFor(t, project, project, iid, branch, gate, dryRun)
}

// OpenMergeRequestFor opens the merge request in project for an issue in
// issueProject; the weekly status report merges in the reports repository
// and closes the status issue in the owner's repository (srd013 R5.4).
func OpenMergeRequestFor(t gitlab.Tracker, project, issueProject string, iid int, branch string, gate AuditGate, dryRun bool) (OpenMergeRequestResult, error) {
	return OpenMergeRequestClosing(t, project, issueProject, iid, branch, gate, nil, dryRun)
}

// OpenMergeRequestClosing also closes each issue in closes, project#iid,
// with its own Closes line: a sprint-end merge request closes the
// objectives the owner reports complete (srd038 R2.1). Each must be open.
func OpenMergeRequestClosing(t gitlab.Tracker, project, issueProject string, iid int, branch string, gate AuditGate, closes []string, dryRun bool) (OpenMergeRequestResult, error) {
	r := OpenMergeRequestResult{Project: project, IID: iid, DryRun: dryRun}
	for _, ref := range closes {
		p, n, ok := strings.Cut(ref, "#")
		other, err := strconv.Atoi(n)
		if !ok || p == "" || err != nil || other <= 0 {
			return r, fmt.Errorf("--closes %q is not project#iid", ref)
		}
		c, err := t.GetIssue(p, other)
		if err != nil {
			return r, fmt.Errorf("--closes %s: %w", ref, err)
		}
		if c.State == "closed" {
			return r, fmt.Errorf("--closes %s is already closed", ref)
		}
	}
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
	base, err := t.DefaultBranch(project)
	if err != nil {
		return r, err
	}
	ahead, paths, err := t.Compare(project, base, branch)
	if err != nil {
		return r, err
	}
	if ahead == 0 {
		return r, fmt.Errorf("%s has no commits over %s; commit and push the work first", branch, base)
	}
	for _, p := range paths {
		if p == "docs" || strings.HasPrefix(p, "docs/") {
			r.Audit = true
		}
	}
	if r.Audit {
		if err := gate.check(); err != nil {
			return r, err
		}
	}
	if existing, err := t.OpenMergeRequest(project, branch); err != nil {
		return r, err
	} else if existing != nil {
		r.MR, r.WebURL = existing.IID, existing.WebURL
		return r, nil
	}
	r.Created = true
	if dryRun {
		return r, nil
	}
	mr, err := t.CreateMergeRequest(project, gitlab.MergeRequest{
		Title: i.Title, SourceBranch: branch, TargetBranch: base,
		Description: mrDescription(i, r.Audit, project) + closesLines(closes),
	})
	if err != nil {
		return r, err
	}
	r.MR, r.WebURL = mr.IID, mr.WebURL
	return r, nil
}

// check runs the gate and reports its findings when it fails, so no merge
// request opens for a docs/ change the corpus audit rejects (srd036 R3.2).
func (g AuditGate) check() error {
	if g.Command == "" {
		return fmt.Errorf("the branch changes docs/ and no audit command is configured")
	}
	run := g.Run
	if run == nil {
		run = ShellAudit
	}
	out, err := run(g.Dir, g.Command)
	if err == nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
	}
	return fmt.Errorf("%s failed in %s; no merge request opened:\n%s", g.Command, g.Dir, strings.Join(lines, "\n"))
}

// MRDescription tells the reviewer what to look at, lists the issue's
// acceptance criteria with their tests and environments, and closes the
// issue on merge. A change under docs/ names the corpus gate (srd013 R5.3).
func MRDescription(i gitlab.Issue, audit bool) string { return mrDescription(i, audit, i.Project) }

// closesLines closes each further issue by its full reference.
func closesLines(closes []string) string {
	var b strings.Builder
	for _, ref := range closes {
		fmt.Fprintf(&b, "Closes %s\n", ref)
	}
	return b.String()
}

// mrDescription closes the issue by its full reference when the merge
// request opens in another project.
func mrDescription(i gitlab.Issue, audit bool, mrProject string) string {
	var b strings.Builder
	b.WriteString("## Review focus\n\n")
	b.WriteString("- The change satisfies every acceptance criterion below, each by its named test in its named environment.\n")
	if audit {
		b.WriteString("- The change is under docs/: mage audit must pass before this merges (srd034 R2.2).\n")
	}
	b.WriteString("\n## Acceptance Criteria\n\n")
	criteria, _ := section(i.Description, "Acceptance Criteria")
	if len(criteria) == 0 {
		b.WriteString("- (the issue names none)\n")
	}
	for _, c := range criteria {
		b.WriteString(c + "\n")
	}
	if i.Project != "" && i.Project != mrProject {
		fmt.Fprintf(&b, "\nCloses %s#%d\n", i.Project, i.IID)
	} else {
		fmt.Fprintf(&b, "\nCloses #%d\n", i.IID)
	}
	return b.String()
}
