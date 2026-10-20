package goals

import (
	"os"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/events"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
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
func fixture(t *testing.T, p trackertest.Program) (tracker.Tracker, Sources) {
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
func fleetIssues(t *testing.T, f tracker.Tracker) []tracker.Issue {
	t.Helper()
	issues, err := f.ListIssues(fleet, tracker.IssueQuery{State: "all"})
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
	// One entry per item, so each item can go to its own team.
	if len(p.Objectives) != 3 {
		t.Fatalf("objectives = %+v", p.Objectives)
	}
	for i, item := range []string{"R1.1", "R1.2"} {
		r1 := p.Objectives[i]
		if r1.Key != "obj001-reliable-agents#R1" || strings.Join(r1.Items, ",") != item || r1.Repository != fleet || r1.Environment != "dev" || r1.Due != "2026-11-06" {
			t.Fatalf("%s = %+v", item, r1)
		}
	}
	r2 := p.Objectives[2]
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
	if res[0].Action != "created" || res[1].Action != "created" || res[0].IID != res[1].IID || res[2].Action != "refused" ||
		!strings.Contains(strings.Join(res[2].Reasons, ";"), "no repository confirmed") {
		t.Fatalf("res = %+v", res)
	}
	issues, _ := f.ListIssues(fleet, openIssues())
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
		"## Notes\n\n## Sprint Backlog\n\n<!-- tpm goals sprint-goal release=01.0 serves=obj001-reliable-agents#R1 -->",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body lacks %q:\n%s", want, body)
		}
	}
	if i.DueDate != "2026-11-06" || len(i.Assignees) != 0 || !i.HasLabel("plane::management") || !i.HasLabel("component::fleet") || !i.HasLabel("op::goals") || i.HasLabel("needs-triage") {
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
	if _, err := f.UpdateIssue(fleet, i.IID, tracker.IssueUpdate{Description: &owned}); err != nil {
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

// TestSprintGoalPerTeam files one Sprint Goal for a team that serves two
// outcomes, citing the items of both, with one planned event (srd022 R4.1,
// srd034 R3.2).
func TestSprintGoalPerTeam(t *testing.T) {
	trackertest.Each(t, func(t *testing.T, tp trackertest.Program) {
		f, s := fixture(t, tp)
		p, _ := MakePlan(f, s)
		p.Objectives[2].Repository = fleet
		res, err := Push(f, s, p, web, program.Defaults(), false)
		if err != nil || res[0].Action != "created" || res[2].Action != "created" || res[0].IID != res[2].IID {
			t.Fatalf("res=%+v err=%v", res, err)
		}
		issues := fleetIssues(t, f)
		if len(issues) != 1 {
			t.Fatalf("issues = %d, want one Sprint Goal for the team", len(issues))
		}
		body := issues[0].Description
		for _, want := range []string{"#R1.1", "#R1.2", "#R2.1", "## Sprint Backlog", "serves=obj001-reliable-agents#R1,obj001-reliable-agents#R2"} {
			if !strings.Contains(body, want) {
				t.Errorf("body lacks %q:\n%s", want, body)
			}
		}
		if !strings.HasPrefix(issues[0].Title, "Sprint Goal: ") || issues[0].DueDate != "2026-11-06" {
			t.Errorf("title %q due %q", issues[0].Title, issues[0].DueDate)
		}
		notes, _ := f.ListNotes(fleet, issues[0].IID)
		e, ok := events.Parse(notes[0].Body)
		if len(notes) != 1 || !ok || len(e.Values("requirement")) != 3 {
			t.Fatalf("planned events %+v", notes)
		}
		writes := tp.Writes()
		if res, err := Push(f, s, p, web, program.Defaults(), false); err != nil || res[0].Action != "unchanged" || tp.Writes() != writes {
			t.Fatalf("second push res=%+v err=%v writes=%d", res, err, tp.Writes()-writes)
		}
	})
}

// TestObjectiveIssueIsSprintGoal keeps a release pushed before 04.0 as it
// is: its objective issue is found as the Sprint Goal for its outcome, and
// a second push files nothing (srd034 R3.7).
func TestObjectiveIssueIsSprintGoal(t *testing.T) {
	trackertest.Each(t, func(t *testing.T, tp trackertest.Program) {
		f, s := fixture(t, tp)
		old, err := f.CreateIssue(fleet, tracker.NewIssue{Title: "Objective obj001-reliable-agents#R1: Evaluation of one agent (release 01.0)",
			Description: "body\n\n## Notes\n\n<!-- tpm goals objective=obj001-reliable-agents#R1 release=01.0 -->\n", Labels: []string{"plane::management", "component::fleet"}})
		if err != nil {
			t.Fatal(err)
		}
		p, _ := MakePlan(f, s)
		if p.Objectives[0].Existing == "" {
			t.Errorf("plan does not name the objective issue: %+v", p.Objectives[0])
		}
		writes := tp.Writes()
		res, err := Push(f, s, p, web, program.Defaults(), false)
		if err != nil || res[0].Action != "unchanged" || !res[0].Legacy || res[0].IID != old.IID || tp.Writes() != writes {
			t.Fatalf("res=%+v err=%v writes=%d", res, err, tp.Writes()-writes)
		}
		if n := len(fleetIssues(t, f)); n != 1 {
			t.Fatalf("issues = %d, want the objective issue alone", n)
		}
	})
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
	if _, err := f.EnsureMilestone("program", sprintMilestone("2026-11-06")); err != nil {
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

// TestOneOutcomeAcrossTwoTeams confirms an outcome's two items for two
// teams: each team's Sprint Goal serves the outcome and cites only its own
// item (srd022, srd034 R3.2).
func TestOneOutcomeAcrossTwoTeams(t *testing.T) {
	trackertest.Each(t, func(t *testing.T, tp trackertest.Program) {
		const scoring = "program/delivery/scoring"
		tp.AddFile(system, "v0.1.0", "docs/specs/software-requirements/srd011-scoring.md", []byte(strings.NewReplacer(
			"srd010-delivery", "srd011-scoring", "| fleet |", "| scoring |", "/delivery/fleet", "/delivery/scoring").Replace(string(read(t, "../../testdata/system/srd010-delivery.md")))))
		f, s := fixture(t, tp)
		p, err := MakePlan(f, s)
		if err != nil || len(p.Objectives) != 3 || len(p.Objectives[1].Candidates) != 2 {
			t.Fatalf("plan = %+v, %v", p.Objectives, err)
		}
		p.Objectives[1].Repository = scoring
		res, err := Push(f, s, p, web, program.Defaults(), false)
		if err != nil || res[0].Action != "created" || res[1].Action != "created" || res[0].Project != fleet || res[1].Project != scoring {
			t.Fatalf("res=%+v err=%v", res, err)
		}
		for repo, want := range map[string]string{fleet: "#R1.1", scoring: "#R1.2"} {
			issues, _ := f.ListIssues(repo, openIssues())
			if len(issues) != 1 {
				t.Fatalf("%s has %d issues", repo, len(issues))
			}
			body := issues[0].Description
			other := map[string]string{"#R1.1": "#R1.2", "#R1.2": "#R1.1"}[want]
			if !strings.Contains(body, want) || strings.Contains(body, other) || !strings.Contains(body, "serves=obj001-reliable-agents#R1 ") {
				t.Errorf("%s Sprint Goal:\n%s", repo, body)
			}
		}
	})
}
