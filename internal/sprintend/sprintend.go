// Package sprintend is the tpm side of a sprint-end issue (srd038 R1, R6):
// the facts its Requirements summarize, so the model drafts the period's
// summary from computed state and asks only what tpm cannot know.
package sprintend

import (
	"fmt"

	"github.com/gray-lake-labs/scrum-master/internal/events"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/release"
	"github.com/gray-lake-labs/scrum-master/internal/reports"
	"github.com/gray-lake-labs/scrum-master/internal/sprintgoal"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// SprintGoal is one Sprint Goal issue of the repository, or an objective
// issue read as one (srd034 R3.7), and its state.
type SprintGoal struct {
	Issue string `json:"issue"`
	// Serves lists the initiative outcomes from tasks' marker, doc#Rn.
	Serves           []string `json:"serves"`
	Legacy           bool     `json:"legacy,omitempty"`
	Release          string   `json:"release"`
	Title            string   `json:"title"`
	State            string   `json:"state"` // opened or closed
	Reported         string   `json:"reported,omitempty"`
	ReportedComplete bool     `json:"reported_complete"`
	Met              bool     `json:"met"`
	Ledger           []string `json:"ledger"` // the event names posted on it, in order
	// Stories lists the story ids its Sprint Backlog section names.
	Stories []string `json:"stories"`
}

// Facts is what tpm sprint-end facts returns.
type Facts struct {
	Project     string              `json:"project"`
	Aggregate   reports.Aggregation `json:"aggregate"`
	SprintGoals []SprintGoal        `json:"sprint_goals"`
	Next        release.NextResult  `json:"next"`
	Recommended string              `json:"recommended_version"` // the next minor version (srd015 R4.1)
}

// Input is one sprint-end facts run.
type Input struct {
	Tracker   tracker.Tracker
	Project   string
	Aggregate reports.Aggregation
	Releases  program.Releases
	Artifact  string
	Tags      []string
}

// MakeFacts lists the repository's Sprint Goals with their ledger and
// reported status, beside the period's aggregate and the version tpm
// release next recommends.
func MakeFacts(in Input) (Facts, error) {
	f := Facts{Project: in.Project, Aggregate: in.Aggregate}
	// A Sprint Goal is known by tasks' marker, not by a label.
	issues, err := in.Tracker.ListIssues(in.Project, tracker.IssueQuery{State: "all"})
	if err != nil {
		return f, err
	}
	progress := map[string]reports.Progress{}
	for _, p := range in.Aggregate.Progress {
		progress[p.SprintGoal] = p
	}
	for _, i := range issues {
		m, ok := sprintgoal.Parse(i.Description)
		if !ok {
			continue
		}
		ref := fmt.Sprintf("%s#%d", in.Project, i.IID)
		o := SprintGoal{Issue: ref, Serves: m.Serves, Legacy: m.Legacy, Release: m.Release, Title: i.Title, State: i.State,
			Ledger: []string{}, Stories: sprintgoal.SprintBacklog(i.Description)}
		notes, err := in.Tracker.ListNotes(in.Project, i.IID)
		if err != nil {
			return f, err
		}
		for _, n := range notes {
			if e, ok := events.Parse(n.Body); ok {
				o.Ledger = append(o.Ledger, e.Name)
				o.Met = o.Met || e.Name == "met"
			}
		}
		if p, ok := progress[ref]; ok {
			o.Reported, o.ReportedComplete = p.Reported, p.Complete || p.Reported == "done"
		}
		f.SprintGoals = append(f.SprintGoals, o)
	}
	f.Next = release.Next(in.Releases, in.Artifact, in.Tags)
	for _, c := range f.Next.Candidates {
		if c.Bump == "minor" {
			f.Recommended = c.Version
		}
	}
	return f, nil
}
