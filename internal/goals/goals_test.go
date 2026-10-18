package goals

import (
	"os"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/events"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/trackertest"
)

const (
	system = "program/system"
	fleet  = "program/delivery/fleet"
	web    = "https://gitlab.example.com"
)

func read(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// fixture is the system repository: its road map, subsystem SRD, and
// objective document.
func fixture(t *testing.T, p trackertest.Program) (gitlab.Tracker, Sources) {
	t.Helper()
	p.AddFile(system, "main", "docs/road-map.md", read(t, "../../testdata/system/road-map.md"))
	p.AddTag(system, "v0.1.0")
	p.AddFile(system, "v0.1.0", "docs/specs/objectives/obj001-reliable-agents.md", read(t, "../../testdata/objectives/valid/obj001-reliable-agents.md"))
	p.AddFile(system, "v0.1.0", "docs/specs/software-requirements/srd010-delivery.md", read(t, "../../testdata/system/srd010-delivery.md"))
	f := p.Tracker()
	s, err := Load(f, system, "01.0", "main")
	if err != nil {
		t.Fatal(err)
	}
	return f, s
}

// fleetIssues lists every fleet issue, open or closed.
func fleetIssues(t *testing.T, f gitlab.Tracker) []gitlab.Issue {
	t.Helper()
	issues, err := f.ListIssues(fleet, gitlab.IssueQuery{State: "all"})
	if err != nil {
		t.Fatal(err)
	}
	return issues
}

func TestReadsTheDeclaredCorpus(t *testing.T) {
	trackertest.Each(t, testReadsTheDeclaredCorpus)
}

func testReadsTheDeclaredCorpus(t *testing.T, tp trackertest.Program) {
	_, s := fixture(t, tp)
	if s.Release.Tag != "v0.1.0" || s.Release.Milestone != "2026-W43..W45" || len(s.Release.Objectives) != 3 {
		t.Fatalf("release = %+v", s.Release)
	}
	if c, sub, ok := s.ComponentFor(fleet); !ok || c.Name != "fleet" || sub.Owner != "Product Owner, delivery" {
		t.Fatalf("component for %s = %+v %+v, %v", fleet, c, sub, ok)
	}
}

func TestPlanRecommendsOwnerAndEnvironment(t *testing.T) {
	trackertest.Each(t, testPlanRecommendsOwnerAndEnvironment)
}

func testPlanRecommendsOwnerAndEnvironment(t *testing.T, tp trackertest.Program) {
	f, s := fixture(t, tp)
	p, err := MakePlan(f, s)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Objectives) != 2 {
		t.Fatalf("objectives = %+v", p.Objectives)
	}
	r1, r2 := p.Objectives[0], p.Objectives[1]
	if r1.Key != "obj001-reliable-agents#R1" || strings.Join(r1.Items, ",") != "R1.1,R1.2" || r1.Repository != fleet || r1.Environment != "dev" || r1.Due != "2026-11-06" {
		t.Fatalf("R1 = %+v", r1)
	}
	if r2.Repository != "" || len(r2.Candidates) != 1 || r2.Candidates[0] != fleet {
		t.Fatalf("R2 = %+v", r2)
	}
	if tp.Writes() != 0 {
		t.Fatal("plan wrote to the tracker")
	}
}

func TestPushFilesOneDatedPinnedObjectiveIssue(t *testing.T) {
	trackertest.Each(t, testPushFilesOneDatedPinnedObjectiveIssue)
}

func testPushFilesOneDatedPinnedObjectiveIssue(t *testing.T, tp trackertest.Program) {
	f, s := fixture(t, tp)
	p, _ := MakePlan(f, s)
	res, err := Push(f, s, p, web, program.Defaults(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Action != "created" || res[1].Action != "refused" || !strings.Contains(strings.Join(res[1].Reasons, ";"), "no repository confirmed") {
		t.Fatalf("res = %+v", res)
	}
	issues, _ := f.ListIssues(fleet, gitlabRequest())
	if len(issues) != 1 {
		t.Fatalf("issues = %d", len(issues))
	}
	i := issues[0]
	body := i.Description
	for _, want := range []string{
		"Cites: " + web + "/" + system + "/-/blob/v0.1.0/docs/specs/objectives/obj001-reliable-agents.md#R1.1",
		"Cites: " + web + "/" + system + "/-/blob/v0.1.0/docs/specs/objectives/obj001-reliable-agents.md#R1.2",
		"Environment: dev",
		"## Requirements\n\n1. A grader scores one agent's trajectories against a written rubric. (R1.1)\n2. ",
		"(test: grader_scores_fixture_agent; environment: dev)",
		"## Notes\n\n<!-- tpm goals objective=obj001-reliable-agents#R1 release=01.0 -->",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body lacks %q:\n%s", want, body)
		}
	}
	if i.DueDate != "2026-11-06" || len(i.Assignees) != 0 || !i.HasLabel("plane::management") || !i.HasLabel("component::fleet") || i.HasLabel("needs-triage") {
		t.Fatalf("issue = %+v", i)
	}
	notes, _ := f.ListNotes(fleet, i.IID)
	if len(notes) != 1 {
		t.Fatalf("notes = %+v", notes)
	}
	e, ok := events.Parse(notes[0].Body)
	if !ok || e.Name != "planned" || len(e.Values("requirement")) != 2 || e.Values("environment")[0] != "dev" {
		t.Fatalf("event = %+v", e)
	}
}

func TestSecondPushChangesNothingAndKeepsOwnerNotes(t *testing.T) {
	trackertest.Each(t, testSecondPushChangesNothingAndKeepsOwnerNotes)
}

func testSecondPushChangesNothingAndKeepsOwnerNotes(t *testing.T, tp trackertest.Program) {
	f, s := fixture(t, tp)
	p, _ := MakePlan(f, s)
	if _, err := Push(f, s, p, web, program.Defaults(), false); err != nil {
		t.Fatal(err)
	}
	i := fleetIssues(t, f)[0]
	owned := i.Description + "Owner: rubric drafted, waiting on trajectories.\n"
	if _, err := f.UpdateIssue(fleet, i.IID, gitlab.IssueUpdate{Description: &owned}); err != nil {
		t.Fatal(err)
	}
	writes := tp.Writes()
	res, err := Push(f, s, p, web, program.Defaults(), false)
	if err != nil || res[0].Action != "unchanged" || res[0].Planned || tp.Writes() != writes {
		t.Fatalf("res=%+v err=%v writes=%d", res, err, tp.Writes()-writes)
	}
	issues := fleetIssues(t, f)
	if !strings.Contains(issues[0].Description, "rubric drafted") {
		t.Fatal("owner notes lost")
	}
	if len(issues) != 1 {
		t.Fatal("duplicate issue")
	}
}

func TestUnpinnedCitationAndMissingEnvironmentAreRefused(t *testing.T) {
	trackertest.Each(t, testUnpinnedCitationAndMissingEnvironmentAreRefused)
}

func testUnpinnedCitationAndMissingEnvironmentAreRefused(t *testing.T, tp trackertest.Program) {
	f, s := fixture(t, tp)
	p, _ := MakePlan(f, s)
	p.Tag = "main"
	p.Objectives[1].Repository = fleet
	p.Objectives[1].Environment = ""
	res, err := Push(f, s, p, web, program.Defaults(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Action != "refused" || !strings.Contains(strings.Join(res[0].Reasons, ";"), `branch "main"`) {
		t.Fatalf("R1 = %+v", res[0])
	}
	if res[1].Action != "refused" || !strings.Contains(strings.Join(res[1].Reasons, ";"), "no environment confirmed") {
		t.Fatalf("R2 = %+v", res[1])
	}
	if tp.Writes() != 0 {
		t.Fatal("refused objectives wrote to the tracker")
	}
}

func TestDueFallsBackToMilestoneAndDryRunWritesNothing(t *testing.T) {
	trackertest.Each(t, testDueFallsBackToMilestoneAndDryRunWritesNothing)
}

func testDueFallsBackToMilestoneAndDryRunWritesNothing(t *testing.T, tp trackertest.Program) {
	f, s := fixture(t, tp)
	if _, err := f.EnsureMilestone("program", gitlabMilestone("2026-11-06")); err != nil {
		t.Fatal(err)
	}
	writes := tp.Writes()
	p, _ := MakePlan(f, s)
	p.Objectives = p.Objectives[:1]
	p.Objectives[0].Due = ""
	res, err := Push(f, s, p, web, program.Defaults(), true)
	if err != nil || res[0].Action != "created" || tp.Writes() != writes {
		t.Fatalf("dry run: res=%+v err=%v writes=%d", res, err, tp.Writes()-writes)
	}
	if _, err := Push(f, s, p, web, program.Defaults(), false); err != nil {
		t.Fatal(err)
	}
	if got := fleetIssues(t, f)[0].DueDate; got != "2026-11-06" {
		t.Fatalf("due = %q", got)
	}
}

func TestLoadRefusesADocumentThatIsNotAgreed(t *testing.T) {
	trackertest.Each(t, func(t *testing.T, p trackertest.Program) {
		p.AddFile(system, "main", "docs/road-map.md", read(t, "../../testdata/system/road-map.md"))
		doc := strings.Replace(string(read(t, "../../testdata/objectives/valid/obj001-reliable-agents.md")), "status: agreed", "status: to_be_agreed", 1)
		p.AddFile(system, "v0.1.0", "docs/specs/objectives/obj001-reliable-agents.md", []byte(doc))
		if _, err := Load(p.Tracker(), system, "01.0", "main"); err == nil || !strings.Contains(err.Error(), "only an agreed document") {
			t.Fatalf("err = %v", err)
		}
	})
}
