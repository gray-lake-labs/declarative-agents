package release

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
	"github.com/gray-lake-labs/scrum-master/internal/trackertest"
)

func TestTagRequiresPassingGateAndIncreasingVersion(t *testing.T) {
	trackertest.Each(t, func(t *testing.T, tp trackertest.Program) {
		testTagRequiresPassingGateAndIncreasingVersion(t, tp.Tracker())
	})
}

func testTagRequiresPassingGateAndIncreasingVersion(t *testing.T, f tracker.Tracker) {
	dir, git := repo(t, gatesMarkdown)
	issue, _ := f.CreateIssue("g/po-1", tracker.NewIssue{Title: "Release a"})
	p := policyWith("a", program.Check{Kind: "custom", Environment: "local", Command: "check"})
	failing := false
	gate := func(version string, at time.Time) {
		t.Helper()
		v, _ := ParseVersion(version)
		if _, err := RunGate(f, ReleaseGateInput{Project: "g/po-1", Issue: issue.IID, Artifact: "a", Version: v, Dir: dir,
			Policy: p, Git: git, Out: io.Discard, Now: at,
			Exec: func(string, string, io.Writer) error {
				if failing {
					return io.ErrUnexpectedEOF
				}
				return nil
			}}); err != nil {
			t.Fatal(err)
		}
	}
	tag := func(version string) (TagResult, error) {
		v, _ := ParseVersion(version)
		return ReleaseTag(f, TagInput{Project: "g/po-1", Issue: issue.IID, Artifact: "a", Version: v, Policy: p, Git: git})
	}
	tags := func() string { out, _ := git("tag", "--list"); return out }

	if _, err := tag("0.3.0"); err == nil || !strings.Contains(err.Error(), "no gate has run") {
		t.Fatalf("without a gate: %v", err)
	}
	failing = true
	gate("0.3.0", t0)
	if _, err := tag("0.3.0"); err == nil || !strings.Contains(err.Error(), "did not pass") {
		t.Fatalf("after a failing gate: %v", err)
	}
	if tags() != "" {
		t.Fatalf("tagged without a passing gate: %q", tags())
	}

	failing = false
	gate("0.3.0", t0.Add(time.Hour))
	r, err := tag("0.3.0")
	if err != nil {
		t.Fatal(err)
	}
	if r.Tag != "a/v0.3.0" || !r.Pushed || r.Class != "breaking" {
		t.Fatalf("result %+v", r)
	}
	msg, _ := git("tag", "-l", "--format=%(objecttype) %(contents)", "a/v0.3.0")
	if !strings.HasPrefix(msg, "tag Release a/v0.3.0, breaking") {
		t.Fatalf("tag message %q", msg)
	}
	if remote, _ := RemoteTags(git, "a/*"); !strings.Contains(remote, "a/v0.3.0") {
		t.Fatalf("not pushed: %q", remote)
	}
	notes, _ := f.ListNotes("g/po-1", issue.IID)
	if last := notes[len(notes)-1].Body; !strings.Contains(last, "Tagged a/v0.3.0") || !strings.Contains(last, "breaking") {
		t.Fatalf("release issue note %q", last)
	}

	if _, err := tag("0.2.9"); err == nil || !strings.Contains(err.Error(), "not greater") {
		t.Fatalf("a lower version: %v", err)
	}
	if _, err := git("-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "later"); err != nil {
		t.Fatal(err)
	}
	gate("0.3.1", t0.Add(2*time.Hour))
	if _, err := git("-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "after the gate"); err != nil {
		t.Fatal(err)
	}
	if _, err := tag("0.3.1"); err == nil || !strings.Contains(err.Error(), "not at HEAD") {
		t.Fatalf("HEAD moved after the gate: %v", err)
	}
	if strings.Contains(tags(), "a/v0.3.1") {
		t.Fatal("tagged a revision the gate did not check")
	}
}
