package program

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/beads"
	"github.com/gray-lake-labs/scrum-master/internal/beads/beadstest"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
	"github.com/gray-lake-labs/scrum-master/internal/tracker/fake"
)

// TestMergeDeliversOnBeads is the beads delivery (srd042 R2.3): a docs/
// change merges only once the audit passes, the merge is --no-ff on the
// default branch of the main checkout and is pushed, it cites the issue and
// each issue it also closes, and each closes on it. A second merge changes
// nothing.
func TestMergeDeliversOnBeads(t *testing.T) {
	tr, checkout := newBeadsTracker(t, "alice")
	objective, _ := tr.CreateIssue("g/po-1", tracker.NewIssue{Title: "Objective R1", Labels: []string{"plane::management"}})
	work, _ := tr.CreateIssue("g/po-1", tracker.NewIssue{Title: "Change the use case", Milestone: "m", Description: goodBody,
		Labels: []string{"plane::data", "kind::spec", "component::po-1"}})
	p, err := Pop(tr, ExecGit{}, "g/po-1", work.IID, checkout, Defaults(), false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(p.Worktree, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"docs/use-case.md", "docs/BROKEN"} {
		if err := os.WriteFile(filepath.Join(p.Worktree, f), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	beadstest.Git(t, p.Worktree, "add", "-A")
	beadstest.Git(t, p.Worktree, "commit", "-q", "-m", "Change the use case")
	gate := AuditGate{Command: "test ! -e docs/BROKEN", Dir: p.Worktree}
	closes := []string{"g/po-1#1"}

	if _, err := Merge(tr, ExecGit{}, "g/po-1", "g/po-1", work.IID, p.Branch, p.Worktree, gate, closes, false); err == nil || !strings.Contains(err.Error(), "nothing merged") {
		t.Fatalf("merge with a failing audit = %v, want a refusal", err)
	}
	if out := beadstest.Git(t, checkout, "log", "--merges", "--oneline", "main"); out != "" {
		t.Fatalf("a refused merge merged: %s", out)
	}

	beadstest.Git(t, p.Worktree, "rm", "-q", "docs/BROKEN")
	beadstest.Git(t, p.Worktree, "commit", "-q", "-m", "Fix the corpus")
	beadstest.Git(t, p.Worktree, "push", "-q", "-u", "origin", "HEAD")
	if _, err := Approve(reviewer(t, tr, checkout, "bob"), ExecGit{}, "g/po-1", "g/po-1", work.IID, p.Branch, checkout); err != nil {
		t.Fatal(err)
	}
	m, err := Merge(tr, ExecGit{}, "g/po-1", "g/po-1", work.IID, p.Branch, p.Worktree, gate, closes, false)
	if err != nil {
		t.Fatal(err)
	}
	head := beadstest.Git(t, checkout, "rev-parse", "main")
	remote := beadstest.Git(t, checkout, "ls-remote", "origin", "refs/heads/main")
	message := beadstest.Git(t, checkout, "log", "-1", "--format=%s", "main")
	if !m.Audit || m.Commit != head || !strings.HasPrefix(remote, head) || !strings.Contains(message, "g/po-1#2") || !strings.Contains(message, "closes g/po-1#1") {
		t.Fatalf("merge = %+v, head %s, origin %s, message %q", m, head, remote, message)
	}
	for _, iid := range []int{work.IID, objective.IID} {
		if got, _ := tr.GetIssue("g/po-1", iid); got.State != "closed" {
			t.Fatalf("g/po-1#%d = %s after the merge, want closed", iid, got.State)
		}
	}

	again, err := Merge(tr, ExecGit{}, "g/po-1", "g/po-1", work.IID, p.Branch, p.Worktree, gate, nil, false)
	if err != nil || again.Merged || again.Commit != head {
		t.Fatalf("second merge = %+v, %v, want nothing new", again, err)
	}
}

// TestMergeNeedsTheBeadsTracker: a tracker that does not close issues on a
// local merge is refused.
func TestMergeNeedsTheBeadsTracker(t *testing.T) {
	f := fake.New()
	i := admitted(f, "Score one agent")
	if _, err := Merge(f, ExecGit{}, "g/po-1", "g/po-1", i.IID, "1-score-one-agent", ".", AuditGate{}, nil, false); err == nil || !strings.Contains(err.Error(), "tpm issue merge serves the beads tracker") {
		t.Fatalf("merge on a tracker without local merges = %v, want a refusal naming the beads tracker", err)
	}
}

// reviewer is another person on tr's tracker repository, with their own
// clone of it, reading the same code checkout.
func reviewer(t *testing.T, tr *beads.Tracker, checkout, user string) *beads.Tracker {
	t.Helper()
	remote := beadstest.Git(t, tr.Store.Dir, "remote", "get-url", "origin")
	clone := filepath.Join(filepath.Dir(tr.Store.Dir), user+"-tracker")
	beadstest.Clone(t, remote, clone, user)
	return beads.NewTracker(clone, user, checkout)
}

// TestMergeNeedsAnApprovalOfItsTip holds merge to srd013 R5.5 and R5.6: no
// merge without an approval, none on the assignee's own, none on an
// approval of an earlier commit; another person's approval of the tip
// merges. For an issue nobody took, the approver must not be the person
// merging.
func TestMergeNeedsAnApprovalOfItsTip(t *testing.T) {
	alice, checkout := newBeadsTracker(t, "alice")
	bob := reviewer(t, alice, checkout, "bob")
	work, _ := alice.CreateIssue("g/po-1", tracker.NewIssue{Title: "Score one agent", Milestone: "m", Description: goodBody,
		Labels: []string{"plane::data", "kind::build", "component::po-1"}})
	p, err := Pop(alice, ExecGit{}, "g/po-1", work.IID, checkout, Defaults(), false)
	if err != nil {
		t.Fatal(err)
	}
	commit := func(name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(p.Worktree, name), []byte(name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		beadstest.Git(t, p.Worktree, "add", "-A")
		beadstest.Git(t, p.Worktree, "commit", "-q", "-m", "add "+name)
		beadstest.Git(t, p.Worktree, "push", "-q", "-u", "origin", "HEAD")
	}
	merge := func() (MergeResult, error) {
		return Merge(alice, ExecGit{}, "g/po-1", "g/po-1", work.IID, p.Branch, p.Worktree, AuditGate{}, nil, false)
	}
	commit("score.go")

	if _, err := merge(); err == nil || !strings.Contains(err.Error(), "approval") {
		t.Fatalf("merge with no approval = %v, want a refusal", err)
	}
	if _, err := Approve(alice, ExecGit{}, "g/po-1", "g/po-1", work.IID, "", checkout); err == nil || !strings.Contains(err.Error(), "assignee") {
		t.Fatalf("the assignee's approval = %v, want a refusal", err)
	}
	a, err := Approve(bob, ExecGit{}, "g/po-1", "g/po-1", work.IID, "", checkout)
	if err != nil || !a.Posted || a.Commit == "" {
		t.Fatalf("bob's approval = %+v, %v", a, err)
	}
	if again, err := Approve(bob, ExecGit{}, "g/po-1", "g/po-1", work.IID, "", checkout); err != nil || again.Posted {
		t.Fatalf("bob's second approval = %+v, %v, want nothing posted", again, err)
	}

	commit("later.go")
	if _, err := merge(); err == nil || !strings.Contains(err.Error(), a.Commit[:12]) {
		t.Fatalf("merge after a later commit = %v, want a refusal naming the approved commit", err)
	}
	if _, err := Approve(bob, ExecGit{}, "g/po-1", "g/po-1", work.IID, "", checkout); err != nil {
		t.Fatal(err)
	}
	if m, err := merge(); err != nil || !m.Merged {
		t.Fatalf("merge with bob's approval of the tip = %+v, %v", m, err)
	}

	status, _ := alice.CreateIssue("g/po-1", tracker.NewIssue{Title: "Weekly status", Labels: []string{"plane::management"}})
	beadstest.Git(t, checkout, "checkout", "-q", "-b", "status-week", "main")
	beadstest.Git(t, checkout, "commit", "-q", "--allow-empty", "-m", "status")
	beadstest.Git(t, checkout, "push", "-q", "-u", "origin", "status-week")
	beadstest.Git(t, checkout, "checkout", "-q", "main")
	statusMerge := func() (MergeResult, error) {
		return Merge(alice, ExecGit{}, "g/po-1", "g/po-1", status.IID, "status-week", checkout, AuditGate{}, nil, false)
	}
	if _, err := Approve(alice, ExecGit{}, "g/po-1", "g/po-1", status.IID, "status-week", checkout); err != nil {
		t.Fatal(err)
	}
	if _, err := statusMerge(); err == nil || !strings.Contains(err.Error(), "approval") {
		t.Fatalf("merge on the merger's own approval = %v, want a refusal", err)
	}
	if _, err := Approve(bob, ExecGit{}, "g/po-1", "g/po-1", status.IID, "status-week", checkout); err != nil {
		t.Fatal(err)
	}
	if m, err := statusMerge(); err != nil || !m.Merged {
		t.Fatalf("merge of the unassigned issue with bob's approval = %+v, %v", m, err)
	}
}
