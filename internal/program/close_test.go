package program

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/beads"
	"github.com/gray-lake-labs/scrum-master/internal/beads/beadstest"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab/gitlabtest"
)

// TestPopAndCloseRunOnBeads drives the real Pop and Close against a beads
// tracker and a code repository, both on local disk (srd042 R1.1, R2.3):
// pop assigns the caller and opens the branch at origin and the worktree;
// close refuses an unmerged branch, closes on the local merge citing the
// merge commit, and changes nothing the second time.
func TestPopAndCloseRunOnBeads(t *testing.T) {
	root := t.TempDir()
	remote := beadstest.TrackerRepository(t, root)
	trackerClone := filepath.Join(root, "alice-tracker")
	beadstest.Clone(t, remote, trackerClone, "alice")
	checkout := beadstest.CodeRepository(t, root, "po-1", "alice")
	tr := beads.NewTracker(trackerClone, "alice", checkout)

	i, err := tr.CreateIssue("g/po-1", gitlab.NewIssue{Title: "Score one agent", Milestone: "m", Description: goodBody,
		Labels: []string{"plane::data", "kind::build", "component::po-1"}})
	if err != nil {
		t.Fatal(err)
	}

	p, err := Pop(tr, ExecGit{}, "g/po-1", i.IID, checkout, Defaults(), false)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Assigned || !p.Created || !p.Added {
		t.Fatalf("pop = %+v, want assigned, branch created, worktree added", p)
	}
	if got, _ := tr.GetIssue("g/po-1", i.IID); len(got.Assignees) != 1 || got.Assignees[0] != "alice" {
		t.Fatalf("assignees = %v, want alice", got.Assignees)
	}
	if out := beadstest.Git(t, checkout, "ls-remote", "--heads", "origin", p.Branch); out == "" {
		t.Fatalf("origin has no branch %s", p.Branch)
	}

	if _, err := Close(tr, ExecGit{}, "g/po-1", i.IID, checkout, false); err == nil || !strings.Contains(err.Error(), "not merged") {
		t.Fatalf("close before the merge = %v, want a refusal", err)
	}

	if err := os.WriteFile(filepath.Join(p.Worktree, "score.go"), []byte("package score\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	beadstest.Git(t, p.Worktree, "add", "-A")
	beadstest.Git(t, p.Worktree, "commit", "-q", "-m", "Score one agent")
	beadstest.Git(t, checkout, "merge", "-q", "--no-ff", p.Branch, "-m", "Merge g/po-1#1")
	merge := beadstest.Git(t, checkout, "rev-parse", "HEAD")

	c, err := Close(tr, ExecGit{}, "g/po-1", i.IID, checkout, false)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Closed || c.Commit != merge {
		t.Fatalf("close = %+v, want closed on %s", c, merge)
	}
	got, _ := tr.GetIssue("g/po-1", i.IID)
	notes, _ := tr.ListNotes("g/po-1", i.IID)
	if got.State != "closed" || len(notes) == 0 || !strings.Contains(notes[len(notes)-1].Body, merge) {
		t.Fatalf("issue = %+v, notes = %+v, want closed citing %s", got, notes, merge)
	}

	again, err := Close(tr, ExecGit{}, "g/po-1", i.IID, checkout, false)
	if err != nil || again.Closed {
		t.Fatalf("second close = %+v, %v, want no change", again, err)
	}
}

// TestPopOfflineIsProvisional: a pop taken while the tracker repository is
// unreachable reports the assignment as provisional (eng04-beads-sync).
func TestPopOfflineIsProvisional(t *testing.T) {
	root := t.TempDir()
	remote := beadstest.TrackerRepository(t, root)
	trackerClone := filepath.Join(root, "alice-tracker")
	beadstest.Clone(t, remote, trackerClone, "alice")
	checkout := beadstest.CodeRepository(t, root, "po-1", "alice")
	tr := beads.NewTracker(trackerClone, "alice", checkout)
	i, err := tr.CreateIssue("g/po-1", gitlab.NewIssue{Title: "Score one agent", Milestone: "m", Description: goodBody,
		Labels: []string{"plane::data", "kind::build", "component::po-1"}})
	if err != nil {
		t.Fatal(err)
	}
	beadstest.Git(t, trackerClone, "remote", "set-url", "origin", filepath.Join(root, "unreachable.git"))

	p, err := Pop(tr, ExecGit{}, "g/po-1", i.IID, checkout, Defaults(), false)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Assigned || !p.Provisional {
		t.Fatalf("offline pop = %+v, want assigned and provisional", p)
	}
}

// TestCloseNeedsTheBeadsTracker: on GitLab the merge request's merge closes
// the issue (srd041 R9.3), so close refuses rather than closing by hand.
func TestCloseNeedsTheBeadsTracker(t *testing.T) {
	f := gitlabtest.New()
	i := admitted(f, "Score one agent")
	if _, err := Close(f, ExecGit{}, "g/po-1", i.IID, ".", false); err == nil || !strings.Contains(err.Error(), "merge request") {
		t.Fatalf("close on GitLab = %v, want a refusal naming the merge request", err)
	}
}
