package program

import (
	"errors"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab/gitlabtest"
)

func TestOpenMergeRequestClosesTheIssue(t *testing.T) {
	f := gitlabtest.New()
	i := admitted(f, "Score one agent")
	f.Ahead["g/po-1@1-score-one-agent"] = gitlabtest.Commits{Count: 2, Paths: []string{"internal/grader/grader.go"}}

	r, err := OpenMergeRequest(f, "g/po-1", i.IID, "1-score-one-agent", AuditGate{}, false)
	if err != nil || !r.Created || r.MR != 1 || r.Audit {
		t.Fatalf("r=%+v err=%v", r, err)
	}
	mr := f.MRs["g/po-1"][0]
	for _, want := range []string{"Closes #1", "## Review focus", "(test: grader_scores_fixture_agent; environment: dev)"} {
		if !strings.Contains(mr.Description, want) {
			t.Errorf("description lacks %q:\n%s", want, mr.Description)
		}
	}
	if mr.TargetBranch != "main" || mr.Title != "Score one agent" {
		t.Fatalf("mr = %+v", mr)
	}
	writes := f.Writes
	again, err := OpenMergeRequest(f, "g/po-1", i.IID, "1-score-one-agent", AuditGate{}, false)
	if err != nil || again.Created || again.MR != 1 || f.Writes != writes {
		t.Fatalf("second call: r=%+v err=%v", again, err)
	}
}

func TestOpenMergeRequestRefusesEmptyBranchOrClosedIssue(t *testing.T) {
	f := gitlabtest.New()
	i := admitted(f, "Nothing yet")
	if _, err := OpenMergeRequest(f, "g/po-1", i.IID, "1-nothing-yet", AuditGate{}, false); err == nil || !strings.Contains(err.Error(), "no commits") {
		t.Fatalf("empty branch: err = %v", err)
	}
	f.Ahead["g/po-1@1-nothing-yet"] = gitlabtest.Commits{Count: 1}
	f.Issues["g/po-1"][0].State = "closed"
	if _, err := OpenMergeRequest(f, "g/po-1", i.IID, "1-nothing-yet", AuditGate{}, false); err == nil || !strings.Contains(err.Error(), "is closed") {
		t.Fatalf("closed issue: err = %v", err)
	}
	if len(f.MRs["g/po-1"]) != 0 {
		t.Fatal("a refused open-merge-request opened a merge request")
	}
}

func TestOpenMergeRequestNamesAuditForDocsChange(t *testing.T) {
	f := gitlabtest.New()
	i := admitted(f, "Split the use case")
	f.Ahead["g/po-1@1-split-the-use-case"] = gitlabtest.Commits{Count: 1, Paths: []string{"docs/specs/use-cases/rel01.0-uc001-x.md"}}
	var ran string
	pass := AuditGate{Command: "mage audit", Dir: "/wt", Run: func(dir, cmd string) (string, error) { ran = dir + " " + cmd; return "ok", nil }}
	r, err := OpenMergeRequest(f, "g/po-1", i.IID, "1-split-the-use-case", pass, false)
	if err != nil || !r.Audit || ran != "/wt mage audit" || !strings.Contains(f.MRs["g/po-1"][0].Description, "mage audit must pass") {
		t.Fatalf("r=%+v ran=%q err=%v", r, ran, err)
	}
}

func TestOpenMergeRequestRefusesWhenAuditFails(t *testing.T) {
	f := gitlabtest.New()
	i := admitted(f, "Break the corpus")
	f.Ahead["g/po-1@1-break-the-corpus"] = gitlabtest.Commits{Count: 1, Paths: []string{"docs/specs/use-cases/x.md"}}
	fail := AuditGate{Command: "mage audit", Dir: "/wt", Run: func(string, string) (string, error) {
		return "validate: broken-touchpoint in rel01.0-uc001\nFAIL", errors.New("exit status 1")
	}}
	_, err := OpenMergeRequest(f, "g/po-1", i.IID, "1-break-the-corpus", fail, false)
	if err == nil || !strings.Contains(err.Error(), "broken-touchpoint") || !strings.Contains(err.Error(), "no merge request opened") {
		t.Fatalf("err = %v", err)
	}
	if len(f.MRs["g/po-1"]) != 0 {
		t.Fatal("a failing audit opened a merge request")
	}
	if _, err := OpenMergeRequest(f, "g/po-1", i.IID, "1-break-the-corpus", AuditGate{}, false); err == nil || !strings.Contains(err.Error(), "no audit command") {
		t.Fatalf("no command: err = %v", err)
	}
}

func TestOpenMergeRequestClosesAnIssueInAnotherProject(t *testing.T) {
	f := gitlabtest.New()
	i := admitted(f, "Weekly status")
	f.Ahead["g/reports@2026-W42-po-1"] = gitlabtest.Commits{Count: 1, Paths: []string{"reports/2026-Q4/2026-W42-po-1.md"}}
	r, err := OpenMergeRequestFor(f, "g/reports", "g/po-1", i.IID, "2026-W42-po-1", AuditGate{}, false)
	if err != nil || !r.Created || r.Project != "g/reports" || r.IssueProject != "g/po-1" {
		t.Fatalf("r=%+v err=%v", r, err)
	}
	if mr := f.MRs["g/reports"][0]; !strings.Contains(mr.Description, "Closes g/po-1#1") || len(f.MRs["g/po-1"]) != 0 {
		t.Fatalf("merge request %+v", mr)
	}
}
