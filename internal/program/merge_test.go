package program

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/beads/beadstest"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab/gitlabtest"
)

// TestMergeDeliversOnBeads is the beads delivery (srd042 R2.3): a docs/
// change merges only once the audit passes, the merge is --no-ff on the
// default branch of the main checkout and is pushed, it cites the issue and
// each issue it also closes, and each closes on it. A second merge changes
// nothing.
func TestMergeDeliversOnBeads(t *testing.T) {
	tr, checkout := newBeadsTracker(t, "alice")
	objective, _ := tr.CreateIssue("g/po-1", gitlab.NewIssue{Title: "Objective R1", Labels: []string{"plane::management"}})
	work, _ := tr.CreateIssue("g/po-1", gitlab.NewIssue{Title: "Change the use case", Milestone: "m", Description: goodBody,
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

// TestMergeNeedsTheBeadsTracker: on GitLab a merge request is the delivery.
func TestMergeNeedsTheBeadsTracker(t *testing.T) {
	f := gitlabtest.New()
	i := admitted(f, "Score one agent")
	if _, err := Merge(f, ExecGit{}, "g/po-1", "g/po-1", i.IID, "1-score-one-agent", ".", AuditGate{}, nil, false); err == nil || !strings.Contains(err.Error(), "merge request") {
		t.Fatalf("merge on GitLab = %v, want a refusal naming the merge request", err)
	}
}
