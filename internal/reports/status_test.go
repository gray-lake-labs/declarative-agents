package reports

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
	"github.com/gray-lake-labs/scrum-master/internal/trackertest"
)

const po1 = "g/product-owners/po-1"

func statusFixture(p trackertest.Program) (tracker.Tracker, StatusInput) {
	p.AddFile("g/system", "main", "docs/specs/software-requirements/srd010-po-1.md", []byte(`---
id: srd010-po-1
title: "Subsystem: po-1"
---

<!-- field: context | form: "tree" -->
## Context

- owner: Product Owner, po-1

<!-- req: R1 -->
## R1 - Receive

<!-- req: R1.1 -->
### R1.1

Objective issues arrive.

<!-- field: implementation_targets -->
## Implementation targets

| name | repository | path |
| --- | --- | --- |
| po-1 | https://gitlab.example.com/g/product-owners/po-1 | / |
`))
	doc, _ := configDocument(Config{Team: []Member{{Username: "po-1", Project: po1}}, System: "g/system"})
	cfg, _ := ParseConfig([]byte(doc))
	return p.Tracker(), StatusInput{Config: cfg, Reports: "g/reports", Policy: program.Defaults(), Ref: "main", Now: monday}
}

// issueCount counts project's issues, open or closed.
func issueCount(t *testing.T, f tracker.Tracker, project string) int {
	t.Helper()
	all, err := f.ListIssues(project, tracker.IssueQuery{State: "all"})
	if err != nil {
		t.Fatal(err)
	}
	return len(all)
}

func TestStatusIssueDueAfterReminderWeekdayMentionsOwner(t *testing.T) {
	trackertest.Each(t, testStatusIssueDueAfterReminderWeekdayMentionsOwner)
}

func testStatusIssueDueAfterReminderWeekdayMentionsOwner(t *testing.T, p trackertest.Program) {
	f, in := statusFixture(p)
	res, err := StatusIssues(f, in)
	if err != nil || len(res) != 1 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	r := res[0]
	if r.Action != "filed" || r.Week != "2026-W42" || r.DueDate != "2026-10-15" || !r.Admitted {
		t.Fatalf("result %+v", r)
	}
	i, _ := f.GetIssue(po1, r.IID)
	if i.Title != "Recurring: weekly status 2026-W42" || i.DueDate != "2026-10-15" || len(i.Assignees) != 0 {
		t.Fatalf("issue %+v", i)
	}
	for _, l := range []string{"plane::data", "type::recurring", "kind::review", "component::po-1", "op::status"} {
		if !slices.Contains(i.Labels, l) {
			t.Errorf("labels %v lack %s", i.Labels, l)
		}
	}
	for _, want := range []string{"@po-1:", "## Requirements", "tpm reports week --project " + po1 + " --user po-1", "reports/2026-Q4/2026-W42-po-1.md",
		"--project g/reports --issue-project " + po1, "## Acceptance Criteria", "## Recurrence", "day after Wednesday", "## Previous Runs"} {
		if !strings.Contains(i.Description, want) {
			t.Errorf("body lacks %q", want)
		}
	}

	friday := monday.AddDate(0, 0, 4)
	if got := TargetWeek(friday, time.Wednesday); ISOWeek(got) != "2026-W43" || DueDate(got, time.Wednesday).Format("2006-01-02") != "2026-10-22" {
		t.Fatalf("after the due date the target is next week: %s", ISOWeek(got))
	}
	if d := DueDate(WeekStart(monday), time.Sunday).Format("2006-01-02"); d != "2026-10-19" {
		t.Fatalf("a Sunday reminder is due Monday: %s", d)
	}
}

func TestStatusIssuesIdempotentPerWeek(t *testing.T) {
	trackertest.Each(t, testStatusIssuesIdempotentPerWeek)
}

func testStatusIssuesIdempotentPerWeek(t *testing.T, p trackertest.Program) {
	f, in := statusFixture(p)
	first, _ := StatusIssues(f, in)
	again, err := StatusIssues(f, in)
	if err != nil || again[0].Action != "exists" || again[0].IID != first[0].IID || issueCount(t, f, po1) != 1 {
		t.Fatalf("second run: %+v err=%v issues=%d", again, err, issueCount(t, f, po1))
	}

	f.UpdateIssue(po1, first[0].IID, tracker.IssueUpdate{State: ptr("close")})
	in.Now = monday.AddDate(0, 0, 7)
	next, err := StatusIssues(f, in)
	if err != nil || next[0].Action != "recurred" || next[0].Week != "2026-W43" || next[0].DueDate != "2026-10-22" {
		t.Fatalf("next week: %+v err=%v", next, err)
	}
	i, _ := f.GetIssue(po1, next[0].IID)
	if i.Title != "Recurring: weekly status 2026-W43" || !strings.Contains(i.Description, "reports/2026-Q4/2026-W43-po-1.md") ||
		!strings.Contains(i.Description, "- "+po1+"#1 (due 2026-10-15): closed by its merge.") ||
		!strings.Contains(i.Description, "week=2026-W43 user=po-1") || strings.Contains(i.Description, "week=2026-W42") {
		t.Fatalf("recurred issue:\n%s", i.Description)
	}
	old, _ := f.GetIssue(po1, first[0].IID)
	if old.State != "closed" {
		t.Fatal("the closed instance changed state")
	}
	if again, _ := StatusIssues(f, in); again[0].Action != "exists" || issueCount(t, f, po1) != 2 {
		t.Fatalf("second run next week: %+v", again)
	}
}

func ptr[T any](v T) *T { return &v }

func TestWeekListsOpenObjectivesAndActivity(t *testing.T) {
	trackertest.Each(t, testWeekListsOpenObjectivesAndActivity)
}

func testWeekListsOpenObjectivesAndActivity(t *testing.T, p trackertest.Program) {
	for _, mr := range []tracker.Merge{
		{IID: 1, Title: "Add grader notes", Author: "po-1", MergedAt: "2026-10-13"},
		{IID: 2, Title: "Earlier work", Author: "po-1", MergedAt: "2026-10-09"},
		{IID: 3, Title: "Someone else", Author: "po-2", MergedAt: "2026-10-13"},
	} {
		p.AddMerge(po1, mr)
	}
	f := p.Tracker()
	obj, _ := f.CreateIssue(po1, tracker.NewIssue{Title: "Score one agent", DueDate: "2026-11-06",
		Description: "Body\n\n<!-- tpm goals objective=obj001-a:R1 release=01.0 -->"})
	f.CreateIssue(po1, tracker.NewIssue{Title: "Ad hoc", Description: "not an objective"})
	done, _ := f.CreateIssue(po1, tracker.NewIssue{Title: "Grader notes"})
	if err := p.As("po-1").AssignIssue(po1, done.IID, tracker.User{Username: "po-1"}); err != nil {
		t.Fatal(err)
	}
	f.UpdateIssue(po1, done.IID, tracker.IssueUpdate{State: ptr("close")})
	r, err := Week(f, po1, "po-1", monday.AddDate(0, 0, 2))
	if err != nil {
		t.Fatal(err)
	}
	if r.Week != "2026-W42" || r.Since != "2026-10-12" {
		t.Fatalf("week %s since %s", r.Week, r.Since)
	}
	if len(r.SprintGoals) != 1 || r.SprintGoals[0].Ref != po1+"#"+itoa(obj.IID) || r.SprintGoals[0].DueDate != "2026-11-06" {
		t.Fatalf("objectives %+v", r.SprintGoals)
	}
	if len(r.Merged) != 1 || r.Merged[0].IID != 1 || len(r.Closed) != 1 || r.Closed[0].Title != "Grader notes" {
		t.Fatalf("activity merged=%+v closed=%+v", r.Merged, r.Closed)
	}
}
