package program

import (
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab/gitlabtest"
)

const goodBody = `Satisfy the evaluation objective

Cites: https://gitlab.example.com/g/system/-/blob/v1.0.0/docs/specs/objectives/obj001-a.md#R1.1
Environment: dev

## Requirements

1. Score one agent's trajectories.

## Acceptance Criteria

- One agent is scored (test: grader_scores_fixture_agent; environment: dev)
`

func objective(body string) gitlab.Issue {
	return gitlab.Issue{Project: "g/po-1", IID: 1, Description: body, Milestone: "2026-W43..W45",
		Labels: []string{"plane::data", "kind::build", "component::fleet"}}
}

func TestAdmittedObjectiveHasNoReasons(t *testing.T) {
	if r := Lint(objective(goodBody), Defaults()); len(r) != 0 {
		t.Fatalf("reasons: %v", r)
	}
}

func TestLintReportsEveryFailure(t *testing.T) {
	body := strings.Replace(goodBody, "/-/blob/v1.0.0/", "/-/blob/main/", 1)
	body = strings.Replace(body, " (test: grader_scores_fixture_agent; environment: dev)", "", 1)
	i := objective(body)
	i.Labels = []string{"plane::data", "plane::control", "kind::nonsense"}
	i.Milestone = ""
	got := strings.Join(Lint(i, Defaults()), "\n")
	for _, want := range []string{"exactly one plane:: label, has 2", "exactly one component:: label, has 0", "kind::nonsense is not a listed kind",
		"needs a sprint milestone", "names no (test: T; environment: E)", `branch "main"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestSprintIssueNeedsNoComponent(t *testing.T) {
	i := gitlab.Issue{Labels: []string{"plane::management", "type::sprint"}}
	if r := Lint(i, Defaults()); len(r) != 0 {
		t.Fatalf("reasons: %v", r)
	}
}

func TestManagementFilersAreEnforcedWhenListed(t *testing.T) {
	p := Defaults()
	p.ManagementFilers = []string{"tpm-lead"}
	i := gitlab.Issue{Author: "someone", Labels: []string{"plane::management", "component::system"}}
	if r := Lint(i, p); len(r) != 1 || !strings.Contains(r[0], "someone may not file a management issue") {
		t.Fatalf("reasons: %v", r)
	}
}

func TestApplySetsAndClearsNeedsTriageAndNothingElse(t *testing.T) {
	f := gitlabtest.New()
	p := Defaults()
	bad := objective(strings.Replace(goodBody, "/-/blob/v1.0.0/", "/-/blob/main/", 1))
	created, _ := f.CreateIssue("g/po-1", gitlab.NewIssue{Title: "o", Description: bad.Description, Labels: bad.Labels, Milestone: bad.Milestone})

	r, err := Apply(f, created, Lint(created, p), p, false)
	if err != nil || r.Admitted || !r.Changed {
		t.Fatalf("r=%+v err=%v", r, err)
	}
	got, _ := f.GetIssue("g/po-1", created.IID)
	if !got.HasLabel("needs-triage") || len(got.Labels) != len(created.Labels)+1 || len(got.Assignees) != 0 {
		t.Fatalf("labels=%v assignees=%v", got.Labels, got.Assignees)
	}
	notes, _ := f.ListNotes("g/po-1", created.IID)
	if len(notes) != 1 || !strings.Contains(notes[0].Body, "not pinned") {
		t.Fatalf("notes=%+v", notes)
	}

	if r, _ := Apply(f, got, Lint(got, p), p, false); r.Changed {
		t.Fatal("second identical lint changed something")
	}

	fixed := got
	fixed.Description = goodBody
	if _, err := Apply(f, fixed, Lint(fixed, p), p, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.GetIssue("g/po-1", created.IID); got.HasLabel("needs-triage") {
		t.Fatalf("needs-triage not cleared: %v", got.Labels)
	}
}

func TestFileCreatesUnassignedLintedIssueAndDryRunWritesNothing(t *testing.T) {
	f := gitlabtest.New()
	in := gitlab.NewIssue{Title: "objective", Description: goodBody, Milestone: "2026-W43..W45", Labels: Labels("data", "", "build", "fleet")}
	r, err := File(f, "g/po-1", in, Defaults(), true)
	if err != nil || !r.DryRun || !r.Lint.Admitted || f.Writes != 0 {
		t.Fatalf("dry run: r=%+v err=%v writes=%d", r, err, f.Writes)
	}
	r, err = File(f, "g/po-1", in, Defaults(), false)
	if err != nil || r.IID != 1 || !r.Lint.Admitted {
		t.Fatalf("r=%+v err=%v", r, err)
	}
	got, _ := f.GetIssue("g/po-1", 1)
	if len(got.Assignees) != 0 || !got.HasLabel("component::fleet") {
		t.Fatalf("%+v", got)
	}
}

func TestLoadPolicyReadsControlFiles(t *testing.T) {
	f := gitlabtest.New()
	f.AddFile("g/system", "main", "control/policy.md", []byte(`---
id: policy
title: Policy
---

<!-- field: admission -->
## Admission

<!-- field: require_milestone -->
### Require milestone

false

<!-- field: management_filers -->
### Management filers

- lead-a
`))
	f.AddFile("g/system", "main", "control/labels.md", []byte(`---
id: labels
title: Labels
---

<!-- field: axes -->
## Axes

<!-- field: kind -->
### Kind

<!-- field: values -->
#### Values

| name |
| --- |
| build |
`))
	p, err := LoadPolicy(f, "g/system", "main")
	if err != nil || p.RequireMilestone || p.ManagementFilers[0] != "lead-a" || len(p.Kinds) != 1 {
		t.Fatalf("p=%+v err=%v", p, err)
	}
	if p, err := LoadPolicy(f, "g/other", "main"); err != nil || !p.RequireMilestone {
		t.Fatalf("absent control files: p=%+v err=%v", p, err)
	}
}
