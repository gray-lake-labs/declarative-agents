package scaffold

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/tracker"
	"github.com/gray-lake-labs/scrum-master/internal/tracker/fake"
)

// subsystemSRD lists one component, the repository the program already has.
const subsystemSRD = `---
id: srd010-ingest
title: "Subsystem: ingest"
---

<!-- field: context | form: "tree" -->
## Context

- owner: Product Owner, ingest

<!-- req: R1 -->
## R1 - Ingest

<!-- req: R1.1 -->
### R1.1

Records arrive.

<!-- field: implementation_targets -->
## Implementation targets

| name | repository | path |
| --- | --- | --- |
| ingest | https://git.example.invalid/g/ingest | / |
`

func system(t *testing.T) *fake.Fake {
	t.Helper()
	f := fake.New()
	f.AddFile("g/system", "main", "docs/specs/software-requirements/srd010-ingest.md", []byte(subsystemSRD))
	return f
}

func joining(dir string) OnboardOptions {
	return OnboardOptions{Dir: dir, System: "g/system", Ref: "main", Subsystem: "srd010-ingest", Repository: "https://git.example.invalid/g/store"}
}

func TestOnboardWritesMemberSkeleton(t *testing.T) {
	dir := repo(t)
	r, err := Onboard(system(t), joining(dir))
	if err != nil {
		t.Fatal(err)
	}
	got := actions(r.Files)
	if got["docs/specs/software-requirements/srd001-subsystem.md"] != "created" || got["docs/VISION.md"] != "created" {
		t.Fatalf("member skeleton: %v", got)
	}
	for p := range got {
		if strings.HasPrefix(p, "control/") || strings.Contains(p, "srd001-system") {
			t.Errorf("a member repository got %s", p)
		}
	}
	vision, _ := os.ReadFile(filepath.Join(dir, "docs", "VISION.md"))
	if !strings.Contains(string(vision), "The Product Owner writes here") || strings.Contains(string(vision), "{{") {
		t.Fatalf("member VISION:\n%s", vision)
	}
}

func TestOnboardFilesOneRegistration(t *testing.T) {
	f := system(t)
	before := len(f.Files)
	r, err := Onboard(f, joining(repo(t)))
	if err != nil || !r.Registration.Filed || r.Registration.Issue == nil {
		t.Fatalf("registration: %+v, %v", r.Registration, err)
	}
	issues, _ := f.ListIssues("g/system", tracker.IssueQuery{State: "opened"})
	if len(issues) != 1 {
		t.Fatalf("%d issues in the system repository, want 1", len(issues))
	}
	i := issues[0]
	if !slices.Contains(i.Labels, "plane::management") || !strings.Contains(i.Title, "g/store") ||
		!strings.Contains(i.Description, "srd010-ingest") || !strings.Contains(i.Description, registrationMarker("g/store")) || len(i.Assignees) != 0 {
		t.Fatalf("registration issue: %+v", i)
	}
	if len(f.Files) != before {
		t.Fatalf("onboard wrote to the system repository: %d files, was %d", len(f.Files), before)
	}
}

func TestOnboardFilesNoSecondRegistration(t *testing.T) {
	f := system(t)
	if _, err := Onboard(f, joining(repo(t))); err != nil {
		t.Fatal(err)
	}
	again, err := Onboard(f, joining(repo(t)))
	if err != nil || again.Registration.Filed || !strings.Contains(again.Registration.Reason, "already asks") {
		t.Fatalf("second onboard: %+v, %v", again.Registration, err)
	}
	listed := joining(repo(t))
	listed.Repository = "https://git.example.invalid/g/ingest"
	r, err := Onboard(f, listed)
	if err != nil || r.Registration.Filed || !strings.Contains(r.Registration.Reason, "already lists") {
		t.Fatalf("onboard of a listed repository: %+v, %v", r.Registration, err)
	}
	if issues, _ := f.ListIssues("g/system", tracker.IssueQuery{State: "opened"}); len(issues) != 1 {
		t.Fatalf("%d registrations, want 1", len(issues))
	}
}
