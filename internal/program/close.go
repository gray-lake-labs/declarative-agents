package program

import (
	"fmt"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
)

// MergeCloser is a tracker that closes an issue on a merge it is shown,
// citing the merge commit. Only the beads tracker is one (srd042 R2.3); on
// GitLab the merge request's merge closes the issue (srd041 R9.3).
type MergeCloser interface {
	CloseOnMerge(project string, iid int, commit string) (bool, error)
}

// CloseResult is what close reports.
type CloseResult struct {
	Project string `json:"project"`
	IID     int    `json:"iid"`
	Branch  string `json:"branch"`
	Commit  string `json:"commit"`
	Closed  bool   `json:"closed"` // close changed the issue now
	DryRun  bool   `json:"dry_run"`
}

// Close closes an issue on the local merge that delivered it (srd042 R2.3):
// a merge commit on the default branch of the checkout around dir that
// brings in the issue's pop branch and cites project#iid. A fast-forward or
// a branch with nothing of its own is not a delivery. The commit cited is
// the first such merge; a second close changes nothing.
func Close(t gitlab.Tracker, git Git, project string, iid int, dir string, dryRun bool) (CloseResult, error) {
	r := CloseResult{Project: project, IID: iid, DryRun: dryRun}
	c, ok := t.(MergeCloser)
	if !ok {
		return r, fmt.Errorf("%s#%d closes when its merge request merges; tpm program close serves the beads tracker", project, iid)
	}
	i, err := t.GetIssue(project, iid)
	if err != nil {
		return r, err
	}
	r.Branch = fmt.Sprintf("%d-%s", iid, Slug(i.Title))
	base, err := t.DefaultBranch(project)
	if err != nil {
		return r, err
	}
	tip, err := git.Run(dir, "rev-parse", "--verify", "-q", "refs/heads/"+r.Branch)
	if err != nil {
		return r, fmt.Errorf("no local branch %s for %s#%d", r.Branch, project, iid)
	}
	tip = strings.TrimSpace(tip)
	cite := fmt.Sprintf("%s#%d", project, iid)
	merges, err := git.Run(dir, "rev-list", "--ancestry-path", "--merges", "--reverse", "--fixed-strings",
		"--grep="+cite, tip+"..refs/heads/"+base)
	if err != nil {
		return r, err
	}
	first, _, _ := strings.Cut(strings.TrimSpace(merges), "\n")
	if first == "" {
		return r, fmt.Errorf("%s is not merged into %s by a merge commit citing %s (git merge --no-ff)", r.Branch, base, cite)
	}
	r.Commit = first
	if dryRun {
		return r, nil
	}
	r.Closed, err = c.CloseOnMerge(project, iid, r.Commit)
	return r, err
}
