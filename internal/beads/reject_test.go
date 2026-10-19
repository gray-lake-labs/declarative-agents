package beads

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// rejectPushes makes the fixture's tracker repository refuse every push, as
// a remote does to a writer that keeps losing the race.
func (f *fixture) rejectPushes(on bool) {
	f.t.Helper()
	hook := filepath.Join(f.remote, "hooks", "pre-receive")
	if !on {
		os.Remove(hook)
		return
	}
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		f.t.Fatal(err)
	}
}

func head(f *fixture, s *Store) string {
	f.t.Helper()
	out, err := s.git("rev-parse", "HEAD")
	if err != nil {
		f.t.Fatal(err)
	}
	return out
}

func TestRejectedWriteIsNotWritten(t *testing.T) {
	f := newFixture(t, ready("sm-1"), ready("sm-2"))
	bob := f.clone("bob")
	bob.sleep = func(time.Duration) {}
	if _, _, err := bob.Issues(); err != nil {
		t.Fatal(err)
	}
	before := head(f, bob)

	f.rejectPushes(true)
	if _, err := bob.Pop("sm-1"); err == nil || !strings.Contains(err.Error(), "nothing was written") {
		t.Fatalf("pop with every push rejected = %v, want a refusal saying nothing was written", err)
	}
	if got := head(f, bob); got != before {
		t.Fatalf("the clone moved from %s to %s; the failed write is still in it", before, got)
	}
	if got := f.remoteIssue("sm-1"); got.Assignee != "" {
		t.Fatalf("sm-1 assignee at the remote = %q, want none", got.Assignee)
	}

	f.rejectPushes(false)
	if _, err := bob.Pop("sm-2"); err != nil {
		t.Fatal(err)
	}
	if got := f.remoteIssue("sm-1"); got.Assignee != "" {
		t.Fatalf("a later write landed the failed pop of sm-1: assignee %q", got.Assignee)
	}
	if got := f.remoteIssue("sm-2"); got.Assignee != "bob" {
		t.Fatalf("sm-2 assignee = %q, want bob", got.Assignee)
	}
}

func TestRejectedWriteKeepsQueuedWrites(t *testing.T) {
	f := newFixture(t, ready("sm-1"), ready("sm-2"))
	bob := f.clone("bob")
	bob.sleep = func(time.Duration) {}
	if _, _, err := bob.Issues(); err != nil {
		t.Fatal(err)
	}

	f.offline(bob)
	queued, err := bob.Pop("sm-1")
	if err != nil || !queued.Provisional {
		t.Fatalf("offline pop of sm-1 = %+v, %v, want provisional", queued, err)
	}
	f.online(bob)

	f.rejectPushes(true)
	if _, err := bob.Pop("sm-2"); err == nil {
		t.Fatal("pop with every push rejected succeeded")
	}
	f.rejectPushes(false)

	r, err := bob.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	confirmed := false
	for _, x := range r {
		confirmed = confirmed || (x.ID == "sm-1" && x.Outcome == "confirmed")
	}
	if !confirmed {
		t.Fatalf("reconcile = %+v, want the queued pop of sm-1 confirmed", r)
	}
	if got := f.remoteIssue("sm-1"); got.Assignee != "bob" {
		t.Fatalf("sm-1 assignee = %q, want bob's queued pop", got.Assignee)
	}
	if got := f.remoteIssue("sm-2"); got.Assignee != "" {
		t.Fatalf("sm-2 assignee = %q, want none: its pop failed", got.Assignee)
	}
}
