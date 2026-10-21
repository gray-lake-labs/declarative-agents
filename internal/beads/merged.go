package beads

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// mergedBy is the Merged-by trailer tpm issue merge writes, naming the
// tracker user who merged.
var mergedBy = regexp.MustCompile(`(?m)^Merged-by: (\S+)$`)

// citation is a project#iid reference in a merge commit's message.
var citation = regexp.MustCompile(`[A-Za-z0-9_.\-]+(?:/[A-Za-z0-9_.\-]+)*#\d+`)

// Merges reads a project's merged history from the merge
// commits on its default branch, which are its deliveries (srd042 R2.3). A merge is numbered by its place in the branch's
// first-parent history, so the number is stable; its author is the tracker
// user its Merged-by trailer names, which tpm issue merge writes, else the
// merge commit's git author; its merge date is the commit's UTC day, and
// since is compared with the commit's instant, never a local date; and it closes every
// issue its message cites, written into the description as
// Closes lines. The branch read is whichever of the local and origin
// default branches is the newer.
func (t *Tracker) Merges(project, author, since string) ([]tracker.Merge, error) {
	from, err := time.Parse(time.RFC3339, since)
	if err != nil {
		if from, err = time.Parse("2006-01-02", since); err != nil {
			return nil, fmt.Errorf("since %q is neither YYYY-MM-DD nor an RFC 3339 instant", since)
		}
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
	var out []tracker.Merge
	n := 0
	for _, entry := range strings.Split(log, "\x1e") {
		fields := strings.SplitN(strings.TrimLeft(entry, "\n"), "\x1f", 3)
		if len(fields) != 3 {
			continue
		}
		n++
		at, err := time.Parse(time.RFC3339, fields[1])
		if err != nil {
			return nil, fmt.Errorf("merge commit date %q: %w", fields[1], err)
		}
		by, message := fields[0], strings.TrimSpace(fields[2])
		if m := mergedBy.FindStringSubmatch(message); m != nil {
			by = m[1]
			message = strings.TrimSpace(mergedBy.ReplaceAllString(message, ""))
		}
		if at.Before(from) || (author != "" && by != author) {
			continue
		}
		title, body, _ := strings.Cut(message, "\n")
		description := strings.TrimSpace(body)
		for _, ref := range citation.FindAllString(message, -1) {
			description += "\n\nCloses " + ref
		}
		out = append(out, tracker.Merge{IID: n, Title: title, Description: strings.TrimSpace(description),
			TargetBranch: base, State: "merged", Author: by, MergedAt: at.UTC().Format("2006-01-02")})
	}
	return out, nil
}
