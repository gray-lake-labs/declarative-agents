package sprintend

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab/gitlabtest"
	"github.com/gray-lake-labs/scrum-master/internal/plan"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/reports"
)

const po1 = "g/po-1"

// period is a reports checkout with one week in which po-1's objective
// g/po-1#1 is reported done and complete, and a tracker holding that
// objective issue and the repository's sprint-end issue g/po-1#2.
func period(t *testing.T) (string, *gitlabtest.Fake, reports.Config) {
	t.Helper()
	dir := t.TempDir()
	start, _ := reports.ParseWeek("2026-W42")
	path := filepath.Join(dir, reports.ReportPath(start, "lead"))
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(reports.Render("lead", "2026-W42", []reports.Entry{
		{"objective": po1 + "#1", "component": "grader", "status": "done", "complete": "yes", "problems": "none"},
	}, nil)), 0o644)
	f := gitlabtest.New()
	f.CreateIssue(po1, gitlab.NewIssue{Title: "Objective R1", Labels: []string{"plane::management"},
		Description: "Objective R1\n\n<!-- tpm goals objective=obj001-reliable-agents#R1 release=01.0 -->"})
	f.CreateIssue(po1, gitlab.NewIssue{Title: "Recurring: sprint end 01.0", Labels: []string{"type::recurring"},
		Description: plan.SprintEndBody("01.0", "2026-11-06")})
	cfg := reports.Config{Team: []reports.Member{{Username: "lead", Project: po1}}, ReminderWeekday: "wednesday", StatusIssueTitle: "x"}
	return dir, f, cfg
}

func aggregate(t *testing.T, dir string, f *gitlabtest.Fake, cfg reports.Config) reports.Aggregation {
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
	dir, f, cfg := period(t)
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

	// The sprint-end merge request closes the sprint-end issue and the
	// objective the owner reported complete.
	f.Ahead[po1+"@2-sprint-end"] = gitlabtest.Commits{Count: 1, Paths: []string{"docs/sprint-ends/01.0.md"}}
	gate := program.AuditGate{Command: "mage audit", Run: func(string, string) (string, error) { return "", nil }}
	mr, err := program.OpenMergeRequestClosing(f, po1, po1, 2, "2-sprint-end", gate, []string{po1 + "#1"}, false)
	if err != nil {
		t.Fatal(err)
	}
	desc := f.MRs[po1][0].Description
	if !mr.Created || !strings.Contains(desc, "Closes #2\n") || !strings.Contains(desc, "Closes "+po1+"#1\n") {
		t.Fatalf("merge request %+v:\n%s", mr, desc)
	}
	if _, err := program.OpenMergeRequestClosing(f, po1, po1, 2, "2-sprint-end", gate, []string{"not-a-ref"}, false); err == nil {
		t.Fatal("a --closes that is not project#iid was accepted")
	}

	// It merges: both issues close, and no met event is posted.
	closed := "close"
	for _, iid := range []int{1, 2} {
		f.UpdateIssue(po1, iid, gitlab.IssueUpdate{State: &closed})
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
func TestSprintEndReleasesOnlyAfterConfirmation(t *testing.T) {
	body := plan.SprintEndBody("01.1", "2026-11-27")
	steps := []string{"tpm sprint-end facts", "docs/sprint-ends/01.1.md", "--closes", "confirm the release and its version", "the merge alone closes this issue", "Once the merge request merges", "tpm release gate"}
	at := -1
	for _, s := range steps {
		i := strings.Index(body, s)
		if i < 0 || i < at {
			t.Fatalf("%q is missing or out of order in:\n%s", s, body)
		}
		at = i
	}
	dir, f, cfg := period(t)
	facts, err := MakeFacts(Input{Tracker: f, Project: po1, Aggregate: aggregate(t, dir, f, cfg), Releases: program.Defaults().Releases,
		Artifact: "grader", Tags: []string{"grader/v0.1.0"}})
	if err != nil || facts.Recommended != "0.2.0" || facts.Next.LatestTag != "grader/v0.1.0" {
		t.Fatalf("recommendation = %q %+v %v", facts.Recommended, facts.Next, err)
	}
}
