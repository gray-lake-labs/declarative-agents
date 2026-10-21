package reports

import (
	"bytes"
	"encoding/json"
	"github.com/gray-lake-labs/scrum-master/internal/corpus"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/cli"
	"github.com/gray-lake-labs/scrum-master/internal/events"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
	"github.com/gray-lake-labs/scrum-master/internal/trackertest"
)

func work(obj, comp, status, complete string) Entry {
	return Entry{"objective": obj, "component": comp, "status": status, "complete": complete, "problems": "none"}
}

func block(what, comp, since, waiting string) Entry {
	return Entry{"blocker": what, "component": comp, "since": since, "waiting-on": waiting}
}

// aggregateFixture is two weeks of reports from three people, a tracker
// with one met objective and two merges, and the
// configuration naming them.
func aggregateFixture(t *testing.T, p trackertest.Program) (string, tracker.Tracker, Config) {
	t.Helper()
	dir := t.TempDir()
	put := func(user, week string, w, b []Entry) {
		start, _ := ParseWeek(week)
		writeReport(t, dir, ReportPath(start, user), Render(user, week, w, b))
	}
	put("alice", "2026-W41", []Entry{work("g/po-1#4", "grader", "at-risk", "no")},
		[]Entry{block("stale scores", "grader", "2026-09-01", "scoring-api")})
	put("alice", "2026-W42", []Entry{work("g/po-1#4", "grader", "blocked", "no"), work("g/po-1#5", "grader", "done", "yes")},
		[]Entry{block("stale scores", "grader", "2026-09-01", "scoring-api")})
	put("bob", "2026-W42", []Entry{work("g/po-2#1", "fleet", "on-track", "no"), work("g/po-2#2", "fleet", "done", "yes")},
		[]Entry{block("no scoring quota", "fleet", "2026-10-12", "scoring-api")})

	p.AddMerge("g/po-1", tracker.Merge{IID: 7, Title: "Fix the grader cache", Author: "alice", MergedAt: "2026-10-13", Description: "Closes #9"})
	p.AddMerge("g/po-1", tracker.Merge{IID: 8, Title: "Score one agent", Author: "alice", MergedAt: "2026-10-14", Description: "Closes #4"})
	f := p.Tracker()
	for i := 0; i < 2; i++ {
		f.CreateIssue("g/po-2", tracker.NewIssue{Title: "objective"})
	}
	f.CreateNote("g/po-2", 2, "Met.\n\n<!-- tpm program event=met objective=obj001 -->")
	cfg := Config{Team: []Member{{"alice", "g/po-1"}, {"bob", "g/po-2"}, {"carol", "g/po-3"}}, System: "g/system"}
	cfg.defaults()
	return dir, f, cfg
}

func TestAggregateSignalsByComponentNeverByPerson(t *testing.T) {
	trackertest.Each(t, testAggregateSignalsByComponentNeverByPerson)
}

func testAggregateSignalsByComponentNeverByPerson(t *testing.T, p trackertest.Program) {
	dir, f, cfg := aggregateFixture(t, p)
	a, err := Aggregate(f, AggregateInput{Dir: dir, Config: cfg, Policy: program.Defaults(), From: "2026-W41", To: "2026-W42"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Reports != 3 || len(a.ByComponent["grader"]) != 3 || len(a.ByComponent["fleet"]) != 3 {
		t.Fatalf("grouped %d reports: %+v", a.Reports, a.ByComponent)
	}
	got := map[string][]Signal{}
	for _, s := range a.Signals {
		got[s.Kind] = append(got[s.Kind], s)
	}
	expect := func(kind string, n int, subject string) {
		t.Helper()
		if len(got[kind]) != n || (n > 0 && !strings.Contains(got[kind][0].Detail, subject)) {
			t.Errorf("%s: %+v", kind, got[kind])
		}
	}
	expect(SignalSharedBlocker, 1, "scoring-api blocks fleet, grader in 2 reports")
	expect(SignalAgingBlocker, 2, "grader has been blocked")
	expect(SignalSlipped, 1, "g/po-1#4 (grader)")
	expect(SignalContradiction, 1, "g/po-1#5 (grader)")
	expect(SignalMissingReport, 3, "no status report for g/po-2 in 2026-W41")
	expect(SignalUnreportedWork, 1, "g/po-1!7")

	progress := map[string]string{}
	for _, p := range a.Progress {
		progress[p.SprintGoal] = p.State
	}
	if progress["g/po-1#5"] != "reported complete" || progress["g/po-2#2"] != "met" || progress["g/po-1#4"] != "blocked" {
		t.Fatalf("progress %v", progress)
	}
	all, _ := json.Marshal(a.Signals)
	for _, person := range []string{"alice", "bob", "carol"} {
		if strings.Contains(string(all), person) {
			t.Errorf("a signal names %s:\n%s", person, all)
		}
	}
}

func TestActionableSignalIsPushedAsIssue(t *testing.T) {
	trackertest.Each(t, testActionableSignalIsPushedAsIssue)
}

// systemIssues lists g/system's issues, open or closed.
func systemIssues(t *testing.T, f tracker.Tracker) []tracker.Issue {
	t.Helper()
	all, err := f.ListIssues("g/system", tracker.IssueQuery{State: "all"})
	if err != nil {
		t.Fatal(err)
	}
	return all
}

func testActionableSignalIsPushedAsIssue(t *testing.T, p trackertest.Program) {
	f := p.Tracker()
	signals := []Signal{
		{SignalSharedBlocker, "scoring-api", "scoring-api", "2026-W42", "scoring-api blocks fleet, grader in 2 reports", true},
		{SignalMissingReport, "g/po-3", "g/po-3", "2026-W42", "no status report for g/po-3 in 2026-W42", false},
	}
	res, err := PushSignals(f, "g/system", signals, program.Defaults(), false)
	if err != nil || len(res) != 1 || res[0].Action != "filed" || len(systemIssues(t, f)) != 1 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	i := systemIssues(t, f)[0]
	for _, want := range []string{"## Requirements", "## Acceptance Criteria", "(test: tpm_reports_aggregate; environment: local)", "tpm reports signal=shared-blocker/scoring-api/scoring-api"} {
		if !strings.Contains(i.Description, want) {
			t.Errorf("issue lacks %q:\n%s", want, i.Description)
		}
	}
	if !strings.Contains(strings.Join(i.Labels, ","), "plane::management") || !strings.Contains(strings.Join(i.Labels, ","), "component::scoring-api") ||
		!strings.Contains(strings.Join(i.Labels, ","), "op::report") {
		t.Fatalf("labels %v", i.Labels)
	}
	again, err := PushSignals(f, "g/system", signals, program.Defaults(), false)
	if err != nil || again[0].Action != "exists" || len(systemIssues(t, f)) != 1 {
		t.Fatalf("second push: %+v %v", again, err)
	}
}

func TestAggregateHonoursJSONAndDryRun(t *testing.T) {
	trackertest.Each(t, testAggregateHonoursJSONAndDryRun)
}

func testAggregateHonoursJSONAndDryRun(t *testing.T, p trackertest.Program) {
	dir, f, cfg := aggregateFixture(t, p)
	b, _ := configDocument(cfg)
	writeReport(t, dir, ConfigFile, b)
	writes := p.Writes()
	var out bytes.Buffer
	err := cli.Default.Run([]string{"reports", "aggregate", "--json", "--dry-run", "--push", "--dir", dir, "--from", "2026-W41", "--to", "2026-W42"},
		&out, func() (tracker.Tracker, error) { return f, nil })
	if err != nil {
		t.Fatal(err)
	}
	var a Aggregation
	if err := json.Unmarshal(out.Bytes(), &a); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out.String())
	}
	if len(a.Signals) == 0 || len(a.Pushed) == 0 || p.Writes() != writes || len(systemIssues(t, f)) != 0 {
		t.Fatalf("dry run: %d signals, %d pushed, writes %d -> %d", len(a.Signals), len(a.Pushed), writes, p.Writes())
	}
}

func configDocument(c Config) (string, error) {
	b, err := corpus.FormatConfig("reports", "Reports configuration", c)
	return string(b), err
}

// TestAggregateReadsTheLedgerAndOnlyDeliveries shows a Sprint Goal a
// sprint-end merge closed as reported complete in progress though its last
// report said blocked, counts only merges that close an issue as unreported
// work, and words a blocker a component reports on itself.
func TestAggregateReadsTheLedgerAndOnlyDeliveries(t *testing.T) {
	trackertest.Each(t, func(t *testing.T, p trackertest.Program) {
		dir := t.TempDir()
		for _, week := range []string{"2026-W41", "2026-W42"} {
			start, _ := ParseWeek(week)
			writeReport(t, dir, ReportPath(start, "carol"), Render("carol", week, []Entry{work("g/po-3#1", "platform", "blocked", "no")},
				[]Entry{block("own migration", "platform", "2026-10-05", "platform")}))
		}
		start, _ := ParseWeek("2026-W42")
		writeReport(t, dir, ReportPath(start, "dave"), Render("dave", "2026-W42", []Entry{work("g/po-3#1", "platform", "blocked", "no")},
			[]Entry{block("own migration", "platform", "2026-10-12", "platform")}))
		p.AddMerge("g/po-3", tracker.Merge{IID: 1, Title: "Merge pull request #12 from a fork", Author: "carol", MergedAt: "2026-10-14", Description: "From a pull request"})
		f := p.Tracker()
		f.CreateIssue("g/po-3", tracker.NewIssue{Title: "goal"})
		e, text := events.ReportedComplete("01.0", "abc123")
		if _, err := events.Post(f, "g/po-3", 1, e, events.ReportedCompleteRequired, text, false); err != nil {
			t.Fatal(err)
		}
		cfg := Config{Team: []Member{{"carol", "g/po-3"}, {"dave", "g/po-3"}}, System: "g/system"}
		cfg.defaults()

		a, err := Aggregate(f, AggregateInput{Dir: dir, Config: cfg, Policy: program.Defaults(), From: "2026-W41", To: "2026-W42"})
		if err != nil {
			t.Fatal(err)
		}
		if len(a.Progress) != 1 || a.Progress[0].State != "reported complete" || !a.Progress[0].Complete {
			t.Errorf("progress = %+v, want g/po-3#1 reported complete", a.Progress)
		}
		shared := false
		for _, s := range a.Signals {
			if s.Kind == SignalUnreportedWork {
				t.Errorf("a merge closing no issue counts as unreported work: %+v", s)
			}
			if s.Kind == SignalSharedBlocker {
				shared = true
				if strings.Contains(s.Detail, "platform blocks platform") {
					t.Errorf("shared blocker on itself reads %q", s.Detail)
				}
			}
		}
		if !shared {
			t.Errorf("no shared-blocker signal: %+v", a.Signals)
		}
	})
}
