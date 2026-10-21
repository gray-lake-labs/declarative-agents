package program

import (
	"slices"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/tracker"
	"github.com/gray-lake-labs/scrum-master/internal/tracker/fake"
)

func TestPlaneAndKindFollowTheChangedPaths(t *testing.T) {
	cases := []struct {
		name     string
		paths    []string
		kind     string
		inSystem bool
		plane    string
		wantKind string
	}{
		{"road-map anywhere", []string{"docs/road-map.md"}, "", false, "management", ""},
		{"objective document", []string{"docs/specs/objectives/obj001-a.md"}, "", false, "management", ""},
		{"initiative", []string{"docs/specs/initiatives/in001-a.md"}, "", false, "management", ""},
		{"story in a team repository", []string{"docs/specs/stories/st001-a.md"}, "", false, "management", ""},
		{"theme, epic, and feature", []string{"docs/specs/themes/th001-a.md", "docs/specs/epics/ep001-a.md", "docs/specs/features/ft001-a.md"}, "", true, "management", ""},
		{"system SRD", []string{"docs/specs/software-requirements/srd010-po-1.md"}, "", true, "management", ""},
		{"component SRD", []string{"docs/specs/software-requirements/srd003-reader.md"}, "", false, "data", "spec"},
		{"use case", []string{"./docs/specs/use-cases/rel01.0-uc001-x.md"}, "", false, "data", "spec"},
		{"architecture", []string{"docs/ARCHITECTURE.md"}, "", false, "data", "spec"},
		{"policy file", []string{"control/policy.md"}, "", false, "control", ""},
		{"code with stated kind", []string{"internal/reader/reader.go"}, "build", false, "data", "build"},
		{"planning beats design", []string{"docs/ARCHITECTURE.md", "docs/road-map.md"}, "", false, "management", ""},
		{"control beats design", []string{"docs/ARCHITECTURE.md", "control/labels.md"}, "", false, "control", ""},
		{"design beats code", []string{"internal/x.go", "docs/specs/test-suites/test-rel01.0-x.md"}, "build", false, "data", "spec"},
		{"theme directory with its slash", []string{"docs/specs/themes/"}, "", true, "management", ""},
		{"epic directory without its slash", []string{"docs/specs/epics"}, "", false, "management", ""},
		{"SRD directory in the system repository", []string{"docs/specs/software-requirements/"}, "", true, "management", ""},
		{"SRD directory in a team repository", []string{"docs/specs/software-requirements/"}, "", false, "data", "spec"},
		{"sprint-end summary", []string{"docs/sprint-ends/01.0.md"}, "", true, "management", ""},
	}
	for _, c := range cases {
		got, err := Classify(c.paths, c.kind, c.inSystem)
		if err != nil || got.Plane != c.plane || got.Kind != c.wantKind || got.Reason == "" {
			t.Errorf("%s: got %+v err %v, want plane %s kind %q", c.name, got, err, c.plane, c.wantKind)
		}
	}
}

func TestProjectWorkWithoutAKindIsReported(t *testing.T) {
	if _, err := Classify([]string{"internal/x.go"}, "", false); err == nil || !strings.Contains(err.Error(), "needs a kind") {
		t.Fatalf("err = %v", err)
	}
}

const subsystemFixture = `---
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
`

func TestDeriveFillsWhatIsNotGivenAndExplicitWins(t *testing.T) {
	f := fake.New()
	f.AddFile("g/system", "main", "docs/specs/software-requirements/srd010-po-1.md", []byte(subsystemFixture))

	l, err := Derive(f, "g/product-owners/po-1", "g/system", "main", []string{"docs/specs/use-cases/rel01.0-uc001-x.md"}, Labelling{})
	if err != nil || l.Plane != "data" || l.Kind != "spec" || l.Component != "po-1" || l.Derived == nil {
		t.Fatalf("derived: %+v %v", l, err)
	}
	l, err = Derive(f, "g/product-owners/po-1", "g/system", "main", []string{"docs/ARCHITECTURE.md"}, Labelling{Plane: "data", Kind: "research", Component: "other"})
	if err != nil || l.Plane != "data" || l.Kind != "research" || l.Component != "other" || l.Derived != nil {
		t.Fatalf("explicit: %+v %v", l, err)
	}
	l, err = Derive(f, "g/system", "g/system", "main", []string{"docs/specs/software-requirements/srd010-po-1.md"}, Labelling{})
	if err != nil || l.Plane != "management" || l.Kind != "" {
		t.Fatalf("system SRD: %+v %v", l, err)
	}
}

const subsystemMarkdown = `---
id: srd011-po-2
title: "Subsystem: po-2"
---

<!-- field: context | form: "tree" -->
## Context

- owner: Product Owner, po-2

<!-- req: R1 -->
## R1 - Receive

<!-- req: R1.1 -->
### R1.1

Objective issues arrive.

<!-- field: implementation_targets -->
## Implementation targets

| name | repository | path |
| --- | --- | --- |
| po-2 | https://gitlab.example.com/g/product-owners/po-2 | / |
`

func TestLoadSubsystemsReadsMarkdown(t *testing.T) {
	f := fake.New()
	f.AddFile("g/system", "main", "docs/specs/software-requirements/srd010-po-1.md", []byte(subsystemFixture))
	f.AddFile("g/system", "main", "docs/specs/software-requirements/srd011-po-2.md", []byte(subsystemMarkdown))
	f.AddFile("g/system", "main", "docs/specs/software-requirements/srd012-po-3.yaml", []byte("id: srd012-po-3\nowner: ignored\ncomponents: [{name: po-3}]\n"))
	subs, err := LoadSubsystems(f, "g/system", "main")
	if err != nil || len(subs) != 2 {
		t.Fatalf("subsystems = %+v, %v", subs, err)
	}
	c, sub, ok := subs.ComponentFor("g/product-owners/po-2")
	if !ok || c.Name != "po-2" || c.Path != "/" || sub.ID != "srd011-po-2" || sub.Owner != "Product Owner, po-2" {
		t.Fatalf("po-2 = %+v %+v %v", c, sub, ok)
	}
	if c, _, ok := subs.ComponentFor("g/product-owners/po-1"); !ok || c.Name != "po-1" {
		t.Fatalf("po-1 = %+v %v", c, ok)
	}
}

func TestSpecIssueNeedsAcceptanceCriteria(t *testing.T) {
	p := Defaults()
	issue := tracker.Issue{Milestone: "m", Labels: []string{"plane::data", "kind::spec", "component::po-1"}, Description: "Rewrite the use case\n"}
	if r := strings.Join(Lint(issue, p), ";"); !strings.Contains(r, "Acceptance Criteria") || !strings.Contains(r, "Requirements") {
		t.Fatalf("reasons: %s", r)
	}
	issue.Description = "Rewrite\n\n## Requirements\n\n1. Split F3.\n\n## Acceptance Criteria\n\n- mage audit passes (test: mage_audit; environment: local)\n"
	if r := Lint(issue, p); len(r) != 0 {
		t.Fatalf("reasons: %v", r)
	}
}

func TestSpecIssueIsAdmittedForAnyMember(t *testing.T) {
	f := fake.New()
	f.Me = tracker.User{ID: 9, Username: "developer"}
	p := Defaults()
	p.ManagementFilers = []string{"tpm-lead"}
	in := tracker.NewIssue{Title: "Split the use case", Milestone: "m",
		Description: "## Requirements\n\n1. Split F3.\n\n## Acceptance Criteria\n\n- mage audit passes (test: mage_audit; environment: local)\n",
		Labels:      Labels("data", "", "spec", "po-1")}
	r, err := File(f, "g/po-1", in, p, false)
	if err != nil || !r.Lint.Admitted {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestManagementIssueRefusedForNonLead(t *testing.T) {
	f := fake.New()
	f.Me = tracker.User{ID: 9, Username: "developer"}
	p := Defaults()
	p.ManagementFilers = []string{"tpm-lead"}
	in := tracker.NewIssue{Title: "Move release 01.1", Labels: Labels("management", "", "", "system")}
	if _, err := File(f, "g/system", in, p, false); err == nil || !strings.Contains(err.Error(), "may not file a plane::management issue") {
		t.Fatalf("err = %v", err)
	}
	if f.Writes != 0 {
		t.Fatal("a refused issue was filed")
	}
	f.Me = tracker.User{ID: 1, Username: "tpm-lead"}
	if _, err := File(f, "g/system", in, p, false); err != nil {
		t.Fatalf("lead refused: %v", err)
	}
}

// TestStatedKindMakesADataIssue files a data issue of the stated kind on a
// planning document, as tpm's own sprint-end issues are.
func TestStatedKindMakesADataIssue(t *testing.T) {
	l, err := Derive(fake.New(), "g/po-1", "", "main", []string{"docs/sprint-ends/01.0.md"}, Labelling{Kind: "release"})
	if err != nil || l.Plane != "data" || l.Kind != "release" || l.Derived == nil || !strings.Contains(l.Derived.Reason, "stated kind") {
		t.Fatalf("derived %+v (%+v), %v", l, l.Derived, err)
	}
	if r := Lint(tracker.Issue{Labels: Labels(l.Plane, "", l.Kind, "po-1"), Milestone: "01.0"}, Defaults()); slices.Contains(r, "only data issues carry a kind:: label") {
		t.Fatalf("lint: %v", r)
	}
}

// TestSetMilestoneAdmitsAFiledIssue puts a data issue refused for lacking a
// sprint milestone in one, and lint admits it.
func TestSetMilestoneAdmitsAFiledIssue(t *testing.T) {
	f := fake.New()
	f.EnsureMilestone("g", tracker.Milestone{Title: "01.0", StartDate: "2026-10-19", DueDate: "2026-11-06"})
	i, _ := f.CreateIssue("g/po-1", tracker.NewIssue{Title: "Score", Description: goodBody, Labels: []string{"plane::data", "kind::build", "component::po-1"}})
	if r := Lint(i, Defaults()); !slices.Contains(r, "a data issue needs a sprint milestone") {
		t.Fatalf("before: %v", r)
	}
	r, err := SetMilestone(f, "g/po-1", i.IID, "01.0", Defaults(), false)
	if err != nil || !r.Admitted {
		t.Fatalf("set milestone = %+v, %v", r, err)
	}
	if got, _ := f.GetIssue("g/po-1", i.IID); got.Milestone != "01.0" {
		t.Fatalf("milestone %q", got.Milestone)
	}
	if _, err := SetMilestone(f, "g/po-1", i.IID, "09.9", Defaults(), false); err == nil {
		t.Fatal("a milestone the project cannot find was set")
	}
}
