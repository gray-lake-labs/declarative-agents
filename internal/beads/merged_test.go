package beads

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/beads/beadstest"
)

// deliver makes a branch with one commit in the checkout and merges it into
// main with --no-ff as user on date, the local merge that delivers work on
// beads (srd042 R2.3).
func deliver(t *testing.T, dir, branch, user, date, message string) {
	t.Helper()
	beadstest.Git(t, dir, "checkout", "-q", "-b", branch, "main")
	if err := os.WriteFile(dir+"/"+branch+".txt", []byte(branch), 0o644); err != nil {
		t.Fatal(err)
	}
	beadstest.Git(t, dir, "add", "-A")
	beadstest.Git(t, dir, "commit", "-q", "-m", "work on "+branch)
	beadstest.Git(t, dir, "checkout", "-q", "main")
	cmd := exec.Command("git", "-c", "user.name="+user, "-c", "user.email="+user+"@example.com",
		"merge", "-q", "--no-ff", branch, "-m", message)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+date+"T10:00:00Z", "GIT_COMMITTER_DATE="+date+"T10:00:00Z")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("merge %s: %v: %s", branch, err, out)
	}
}

// TestMergedHistoryFromMergeCommits reads a project's merged history on
// beads from the merge commits on its default branch: each merge is
// numbered by its place in that history, carries its author and merge
// date, and closes the issues its message cites; a plain commit is not a
// merge, and author and date filter as on GitLab.
func TestMergedHistoryFromMergeCommits(t *testing.T) {
	tr := newTestTracker(t, "alice")
	dir := tr.checkout("g/po-1")
	deliver(t, dir, "1-score", "alice", "2026-10-20", "Merge 1-score (g/po-1#1)\n\nScore one agent.")
	beadstest.Git(t, dir, "commit", "-q", "--allow-empty", "-m", "a plain commit on main")
	deliver(t, dir, "2-rubric", "bob", "2026-10-27", "Merge 2-rubric (g/po-1#2)")

	all, err := tr.MergedMergeRequests("g/po-1", "", "2026-10-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].IID != 1 || all[1].IID != 2 {
		t.Fatalf("merged = %+v, want merges 1 and 2 only", all)
	}
	first := all[0]
	if first.Author != "alice" || first.MergedAt != "2026-10-20" || first.State != "merged" || first.TargetBranch != "main" ||
		first.Title != "Merge 1-score (g/po-1#1)" || !strings.Contains(first.Description, "Score one agent.") ||
		!strings.Contains(first.Description, "Closes g/po-1#1") {
		t.Fatalf("first merge = %+v", first)
	}

	if recent, _ := tr.MergedMergeRequests("g/po-1", "", "2026-10-26"); len(recent) != 1 || recent[0].IID != 2 || recent[0].Author != "bob" {
		t.Fatalf("since 2026-10-26 = %+v, want bob's merge 2 only", recent)
	}
	if mine, _ := tr.MergedMergeRequests("g/po-1", "alice", "2026-10-01"); len(mine) != 1 || mine[0].IID != 1 {
		t.Fatalf("alice's = %+v, want merge 1 only", mine)
	}
	if _, err := tr.MergedMergeRequests("g/po-1", "", "last week"); err == nil {
		t.Fatal("a since that is not a date was accepted")
	}
}
