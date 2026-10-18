package sprintend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/plan"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/reports"
	"github.com/gray-lake-labs/scrum-master/internal/trackertest"
)

const po1 = "g/po-1"

// period is a reports checkout with one week in which po-1's objective
// g/po-1#1 is reported done and complete, and a tracker holding that
// objective issue and the repository's sprint-end issue g/po-1#2.
func period(t *testing.T, p trackertest.Program) (string, gitlab.Tracker, reports.Config) {
	t.Helper()
	dir := t.TempDir()
	start, _ := reports.ParseWeek("2026-W42")
	path := filepath.Join(dir, reports.ReportPath(start, "lead"))
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(reports.Render("lead", "2026-W42", []reports.Entry{
		{"objective": po1 + "#1", "component": "grader", "status": "done", "complete": "yes", "problems": "none"},
	}, nil)), 0o644)
	p.AddRepository(po1)
	f := p.Tracker()
	f.CreateIssue(po1, gitlab.NewIssue{Title: "Objective R1", Labels: []string{"plane::management"},
		Description: "Objective R1\n\n<!-- tpm goals objective=obj001-reliable-agents#R1 release=01.0 -->"})
	f.CreateIssue(po1, gitlab.NewIssue{Title: "Recurring: sprint end 01.0", Labels: []string{"type::recurring"},
		Description: plan.SprintEndBody("01.0", "2026-11-06")})
	cfg := reports.Config{Team: []reports.Member{{Username: "lead", Project: po1}}, ReminderWeekday: "wednesday", StatusIssueTitle: "x"}
	return dir, f, cfg
}

func aggregate(t *testing.T, dir string, f gitlab.Tracker, cfg reports.Config) reports.Aggregation {
	t.Helper()
	a, err := reports.Aggregate(f, reports.AggregateInput{Dir: dir, Config: cfg, Policy: program.Defaults(), From: "2026-W42", To: "2026-W42"})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func contradictions(a reports.Aggregation) int {
	n := 0
	for _, s := range a.Signals {
		if s.Kind == reports.SignalContradiction {
			n++
		}
	}
	return n
}

func TestReportedCompleteClosesWithoutMet(t *testing.T) {
	trackertest.Each(t, testReportedCompleteClosesWithoutMet)
}

func testReportedCompleteClosesWithoutMet(t *testing.T, p trackertest.Program) {
	dir, f, cfg := period(t, p)
	before := aggregate(t, dir, f, cfg)
	facts, err := MakeFacts(Input{Tracker: f, Project: po1, Aggregate: before, Releases: program.Defaults().Releases, Artifact: "grader"})
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.Objectives) != 1 || !facts.Objectives[0].ReportedComplete || facts.Objectives[0].Met || facts.Objectives[0].Objective != "obj001-reliable-agents#R1" {
		t.Fatalf("facts = %+v", facts.Objectives)
	}
	if contradictions(before) != 1 {
		t.Fatalf("an open objective reported complete is a contradiction until sprint end closes it: %+v", before.Signals)
	}

	// The sprint-end delivery closes the sprint-end issue and the objective
	// the owner reported complete, and no met event is posted.
	if checkout := p.Checkout(po1); checkout == "" {
		closeBoth(t, f)
	} else {
		localMergeCloses(t, f, checkout)
	}
	after := aggregate(t, dir, f, cfg)
	if contradictions(after) != 0 || after.Progress[0].State != "reported complete" || after.Progress[0].Met {
		t.Fatalf("after the merge: %+v %+v", after.Progress, after.Signals)
	}
	facts, _ = MakeFacts(Input{Tracker: f, Project: po1, Aggregate: after, Releases: program.Defaults().Releases, Artifact: "grader"})
	if facts.Objectives[0].State != "closed" || facts.Objectives[0].Met || slices.Contains(facts.Objectives[0].Ledger, "met") {
		t.Fatalf("closed objective = %+v", facts.Objectives[0])
	}
}

// TestSprintEndReleasesOnlyAfterConfirmation holds the sprint-end issue's
// steps to the order srd038 R1.2 and R5.3 state: the summary merges, then
// the owner confirms, then the gate and tag run on main; without
// confirmation the merge alone closes the issue.
// closeBoth stands in for delivery on the fake, which has no repositories
// to merge in: both issues close.
func closeBoth(t *testing.T, f gitlab.Tracker) {
	t.Helper()
	closed := "close"
	for _, iid := range []int{1, 2} {
		if _, err := f.UpdateIssue(po1, iid, gitlab.IssueUpdate{State: &closed}); err != nil {
			t.Fatal(err)
		}
	}
}

// localMergeCloses is the beads delivery (srd042 R2.3): the sprint-end
// branch merges locally with a message citing both issues, and close
// closes each on that merge commit.
func localMergeCloses(t *testing.T, f gitlab.Tracker, checkout string) {
	t.Helper()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = checkout
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	branch := "2-" + program.Slug("Recurring: sprint end 01.0")
	git("checkout", "-q", "-b", branch, "main")
	if err := os.MkdirAll(filepath.Join(checkout, "docs", "sprint-ends"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(checkout, "docs", "sprint-ends", "01.0.md"), []byte("# Sprint end 01.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-q", "-m", "Sprint end 01.0")
	git("checkout", "-q", "main")
	git("merge", "-q", "--no-ff", branch, "-m", fmt.Sprintf("Merge %s (%s#2, closes %s#1)", branch, po1, po1))
	merge := git("rev-parse", "HEAD")
	for _, iid := range []int{2, 1} {
		c, err := program.Close(f, program.ExecGit{}, po1, iid, checkout, false)
		if err != nil || !c.Closed || c.Commit != merge {
			t.Fatalf("close %s#%d = %+v, %v, want closed on %s", po1, iid, c, err, merge)
		}
	}
}

func TestSprintEndReleasesOnlyAfterConfirmation(t *testing.T) {
	body := plan.SprintEndBody("01.1", "2026-11-27")
	steps := []string{"tpm sprint-end facts", "docs/sprint-ends/01.1.md", "tpm program merge", "--closes", "confirm the release and its version", "the merge alone closes this issue", "Once the summary merges", "tpm release gate"}
	at := -1
	for _, s := range steps {
		i := strings.Index(body, s)
		if i < 0 || i < at {
			t.Fatalf("%q is missing or out of order in:\n%s", s, body)
		}
		at = i
	}
	trackertest.Each(t, func(t *testing.T, p trackertest.Program) {
		dir, f, cfg := period(t, p)
		facts, err := MakeFacts(Input{Tracker: f, Project: po1, Aggregate: aggregate(t, dir, f, cfg), Releases: program.Defaults().Releases,
			Artifact: "grader", Tags: []string{"grader/v0.1.0"}})
		if err != nil || facts.Recommended != "0.2.0" || facts.Next.LatestTag != "grader/v0.1.0" {
			t.Fatalf("recommendation = %q %+v %v", facts.Recommended, facts.Next, err)
		}
	})
}
