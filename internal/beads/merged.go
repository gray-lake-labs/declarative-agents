package beads

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
)

// citation is a project#iid reference in a merge commit's message.
var citation = regexp.MustCompile(`[A-Za-z0-9_.\-]+(?:/[A-Za-z0-9_.\-]+)*#\d+`)

// MergedMergeRequests reads a project's merged history from the merge
// commits on its default branch, the delivery a merge request is on GitLab
// (srd042 R2.3). A merge is numbered by its place in the branch's
// first-parent history, so the number is stable; its author is the merge
// commit's author and its merge date the commit date; and it closes every
// issue its message cites, written into the description as GitLab's
// Closes lines. The branch read is whichever of the local and origin
// default branches is the newer.
func (t *Tracker) MergedMergeRequests(project, author, since string) ([]gitlab.MergeRequest, error) {
	if _, err := time.Parse("2006-01-02", since); err != nil {
		return nil, fmt.Errorf("since %q is not YYYY-MM-DD", since)
	}
	base, err := t.DefaultBranch(project)
	if err != nil {
		return nil, err
	}
	t.git(project, "fetch", "-q", "origin", base)
	ref := "refs/heads/" + base
	if _, err := t.git(project, "rev-parse", "--verify", "-q", ref); err != nil {
		ref = "refs/remotes/origin/" + base
	} else if _, err := t.git(project, "merge-base", "--is-ancestor", "refs/remotes/origin/"+base, ref); err != nil {
		if _, err := t.git(project, "rev-parse", "--verify", "-q", "refs/remotes/origin/"+base); err == nil {
			ref = "refs/remotes/origin/" + base
		}
	}
	log, err := t.git(project, "log", "--first-parent", "--merges", "--reverse", "--format=%an%x1f%cI%x1f%B%x1e", ref)
	if err != nil {
		return nil, err
	}
	var out []gitlab.MergeRequest
	n := 0
	for _, entry := range strings.Split(log, "\x1e") {
		fields := strings.SplitN(strings.TrimLeft(entry, "\n"), "\x1f", 3)
		if len(fields) != 3 {
			continue
		}
		n++
		by, date, message := fields[0], fields[1][:10], strings.TrimSpace(fields[2])
		if date < since || (author != "" && by != author) {
			continue
		}
		title, body, _ := strings.Cut(message, "\n")
		description := strings.TrimSpace(body)
		for _, ref := range citation.FindAllString(message, -1) {
			description += "\n\nCloses " + ref
		}
		out = append(out, gitlab.MergeRequest{IID: n, Title: title, Description: strings.TrimSpace(description),
			TargetBranch: base, State: "merged", Author: by, MergedAt: date})
	}
	return out, nil
}
