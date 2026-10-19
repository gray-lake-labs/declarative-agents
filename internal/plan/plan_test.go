package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/corpus"
	"github.com/gray-lake-labs/scrum-master/internal/goals"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/reports"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
	"github.com/gray-lake-labs/scrum-master/internal/trackertest"
)

// obj002 is a second agreed objective document: R1 is due before obj001's
// objectives and fully evidenced; R2.1 has a criterion with no environment.
const obj002 = `---
id: obj002-fast-scoring
title: Fast scoring
status: agreed
---

<!-- field: context | form: "tree" -->
## Context

- owner: technical program manager
- horizon: 2026-W43..2027-W03

<!-- field: problem -->
## Problem

Scoring takes a day.

<!-- field: goals -->
## Goals

- G1: A score is ready within an hour.

<!-- req: R1 -->
## R1 - Score in an hour

<!-- field: applicability | form: "tree" -->
### Applicability

- due: "2026-10-30"

<!-- req: R1.1 -->
### R1.1

A score is ready within an hour of a trajectory.

<!-- req: R2 -->
## R2 - Score on demand

<!-- field: applicability | form: "tree" -->
### Applicability

- due: "2026-12-04"

<!-- req: R2.1 -->
### R2.1

A person asks for a score and gets one.

<!-- field: non_goals -->
## Non-goals

- Scoring models.

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 -->
### AC1

A score is ready within an hour.

<!-- field: traces -->
#### Traces

- R1.1

<!-- field: tests -->
#### Tests

- scorer_meets_the_hour

<!-- field: applicability -->
#### Applicability

dev

<!-- ac: AC2 -->
### AC2

A score comes on demand.

<!-- field: traces -->
#### Traces

- R2.1

<!-- field: tests -->
#### Tests

- scorer_on_demand
`

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

var now = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

// system is a system repository checkout: the testdata road map with
// release 01.0 placing obj001's items, and both objective documents.
func system(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(path string, b []byte) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, path), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	read := func(p string) []byte { return []byte(readFile(t, p)) }
	write(RoadMap, read("../../testdata/system/road-map.md"))
	write("docs/specs/objectives/obj001-reliable-agents.md", read("../../testdata/objectives/valid/obj001-reliable-agents.md"))
	write("docs/specs/objectives/obj002-fast-scoring.md", []byte(obj002))
	return dir
}

// input is the planning input over p's tracker: one objective issue in
// g/po-1 for obj001's R2, which the aggregate reports slipped.
func input(t *testing.T, dir string, p trackertest.Program) (Input, tracker.Tracker) {
	t.Helper()
	f := p.Tracker()
	i, err := f.CreateIssue("g/po-1", tracker.NewIssue{Title: "Objective R2",
		Description: "Objective\n\n<!-- tpm goals objective=obj001-reliable-agents#R2 release=01.0 -->"})
	if err != nil {
		t.Fatal(err)
	}
	agg := &reports.Aggregation{Signals: []reports.Signal{{Kind: reports.SignalSlipped, Subject: fmt.Sprintf("g/po-1#%d", i.IID)}}}
	return Input{Dir: dir, System: "g/system", Tracker: f, Aggregate: agg, Releases: program.Defaults().Releases, Artifact: "corpus", Now: now}, f
}

func ids(items []Item) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.ID)
	}
	return out
}

func TestPlanReleaseRanksSlippedFirst(t *testing.T) {
	trackertest.Each(t, testPlanReleaseRanksSlippedFirst)
}

func testPlanReleaseRanksSlippedFirst(t *testing.T, p trackertest.Program) {
	in, _ := input(t, system(t), p)
	c, err := MakeCandidates(in)
	if err != nil {
		t.Fatal(err)
	}
	// obj001's R1 items are placed in 01.0 and did not slip; its R2.1 is
	// placed but slipped, so it comes back first; obj002's R1.1 is due
	// earliest of the rest; obj002's R2.1 lacks an environment.
	if got := ids(c.Items); !slices.Equal(got, []string{"obj001-reliable-agents#R2.1", "obj002-fast-scoring#R1.1"}) {
		t.Fatalf("items = %v", got)
	}
	if !c.Items[0].Slipped || c.Items[0].Placed != "01.0" {
		t.Fatalf("slipped item = %+v", c.Items[0])
	}
	if c.Latest != "01.0" || c.Version != "01.1" || c.Tag != "corpus/v0.1.0" || c.EndDate != "2026-11-08" {
		t.Fatalf("recommendations = %s %s %s %s", c.Latest, c.Version, c.Tag, c.EndDate)
	}
}

func TestPlanReleaseReturnsItemsWithoutEvidence(t *testing.T) {
	trackertest.Each(t, testPlanReleaseReturnsItemsWithoutEvidence)
}

func testPlanReleaseReturnsItemsWithoutEvidence(t *testing.T, p trackertest.Program) {
	dir := system(t)
	in, f := input(t, dir, p)
	c, err := MakeCandidates(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.WithoutEvidence) != 1 || c.WithoutEvidence[0].ID != "obj002-fast-scoring#R2.1" || !slices.Equal(c.WithoutEvidence[0].Missing, []string{"environment"}) {
		t.Fatalf("without evidence = %+v", c.WithoutEvidence)
	}
	planning, err := f.CreateIssue("g/system", tracker.NewIssue{Title: "Plan release 01.1"})
	if err != nil {
		t.Fatal(err)
	}
	w, err := WriteRelease(dir, Release{Version: "01.1", Milestone: "01.1", Tag: "corpus/v0.1.0",
		Items: []string{"obj002-fast-scoring#R1.1", "obj002-fast-scoring#R2.1"}}, c, f, "g/system", planning.IID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(w.Release.Items, []string{"obj002-fast-scoring#R1.1"}) || len(w.Returned) != 1 {
		t.Fatalf("written = %+v", w)
	}
	notes, _ := f.ListNotes("g/system", planning.IID)
	if len(notes) != 1 || !strings.Contains(notes[0].Body, "obj002-fast-scoring#R2.1: missing environment") {
		t.Fatalf("notes = %+v", notes)
	}
	if _, err := WriteRelease(dir, Release{Version: "01.1", Milestone: "01.1", Tag: "t", Items: []string{"obj009-x#R1.1"}}, c, nil, "", 0); err == nil {
		t.Fatal("an item of no agreed objective document was placed")
	}
}

func TestPlanReleaseWritesTheRoadMapRelease(t *testing.T) {
	trackertest.Each(t, testPlanReleaseWritesTheRoadMapRelease)
}

func testPlanReleaseWritesTheRoadMapRelease(t *testing.T, p trackertest.Program) {
	dir := system(t)
	in, _ := input(t, dir, p)
	c, err := MakeCandidates(in)
	if err != nil {
		t.Fatal(err)
	}
	readBack := func() []goals.Release {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(dir, RoadMap))
		if err != nil {
			t.Fatal(err)
		}
		d, err := corpus.ParseMarkdown(b)
		if err != nil {
			t.Fatal(err)
		}
		return goals.ReleasesOf(d)
	}
	r := Release{Version: "01.1", Milestone: "01.1", Tag: "corpus/v0.1.0", Items: []string{"obj001-reliable-agents#R2.1", "obj002-fast-scoring#R1.1"}}
	if w, err := WriteRelease(dir, r, c, nil, "", 0); err != nil || w.Action != "added" {
		t.Fatalf("first write: %+v %v", w, err)
	}
	got := readBack()
	if len(got) != 2 || got[0].Version != "01.0" || got[1].Version != "01.1" || got[1].Tag != "corpus/v0.1.0" || got[1].Milestone != "01.1" ||
		!slices.Equal(got[1].Objectives, r.Items) || len(got[0].Objectives) != 3 {
		t.Fatalf("road map = %+v", got)
	}
	r.Items = []string{"obj002-fast-scoring#R1.1"}
	if w, err := WriteRelease(dir, r, c, nil, "", 0); err != nil || w.Action != "replaced" {
		t.Fatalf("second write: %+v %v", w, err)
	}
	if got := readBack(); len(got) != 2 || !slices.Equal(got[1].Objectives, r.Items) || got[0].Tag != "v0.1.0" {
		t.Fatalf("after replacing: %+v", got)
	}
}
