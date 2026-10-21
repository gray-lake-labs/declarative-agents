// Package sprintend is the tpm side of a sprint-end issue (srd038 R1, R6):
// the facts its Requirements summarize, so the model drafts the period's
// summary from computed state and asks only what tpm cannot know.
package sprintend

import (
	"fmt"
	"slices"

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
	// Stories lists the stories its Sprint Backlog section names, as
	// project#story.
	Stories []string `json:"stories"`
}

// Facts is what tpm sprint-end facts returns.
type Facts struct {
	Project     string              `json:"project"`
	Aggregate   reports.Aggregation `json:"aggregate"`
	SprintGoals []SprintGoal        `json:"sprint_goals"`
	Next        release.NextResult  `json:"next"`
	// Recommended is the version release next recommends from the
	// checkout's commits since its latest tag, Next.Reason saying why
	// (srd015 R3.1); empty when nothing changed since that tag.
	Recommended string `json:"recommended_version"`
}

// Input is one sprint-end facts run.
type Input struct {
	Tracker tracker.Tracker
	Project string
	// Teams are the program's team repositories, read beside Project when
	// it is the system repository: every Sprint Goal lives in a team's
	// repository, and the program's period closes on all of them.
	Teams     []string
	Aggregate reports.Aggregation
	Releases  program.Releases
	Artifact  string
	Tags      []string
	// Dir is the artifact's checkout, whose commits since its latest tag
	// the recommendation reads; empty recommends the next minor.
	Dir string
}

// MakeFacts lists the repository's Sprint Goals with their ledger and
// reported status, beside the period's aggregate and the version tpm
// release next recommends.
func MakeFacts(in Input) (Facts, error) {
	f := Facts{Project: in.Project, Aggregate: in.Aggregate}
	progress := map[string]reports.Progress{}
	for _, p := range in.Aggregate.Progress {
		progress[p.SprintGoal] = p
	}
	projects := []string{in.Project}
	for _, team := range in.Teams {
		if !slices.Contains(projects, team) {
			projects = append(projects, team)
		}
	}
	for _, project := range projects {
		goals, err := sprintGoals(in.Tracker, project, progress)
		if err != nil {
			return f, err
		}
		f.SprintGoals = append(f.SprintGoals, goals...)
	}
	f.Next = release.Next(in.Releases, in.Artifact, in.Tags)
	if in.Dir != "" {
		next, err := release.Recommendation(in.Releases, in.Artifact, in.Dir, in.Tags)
		if err != nil {
			return f, err
		}
		f.Next = next
	} else {
		f.Next.Recommended, f.Next.Reason = "minor", "no checkout to read commits from: a period's release is minor"
	}
	for _, c := range f.Next.Candidates {
		if c.Bump == f.Next.Recommended {
			f.Recommended = c.Version
		}
	}
	return f, nil
}

// sprintGoals lists project's Sprint Goals with their ledgers; each story
// they list is named project#story, since story ids repeat across teams.
func sprintGoals(t tracker.Tracker, project string, progress map[string]reports.Progress) ([]SprintGoal, error) {
	// A Sprint Goal is known by tasks' marker, not by a label.
	issues, err := t.ListIssues(project, tracker.IssueQuery{State: "all"})
	if err != nil {
		return nil, err
	}
	var out []SprintGoal
	for _, i := range issues {
		m, ok := sprintgoal.Parse(i.Description)
		if !ok {
			continue
		}
		ref := fmt.Sprintf("%s#%d", project, i.IID)
		o := SprintGoal{Issue: ref, Serves: m.Serves, Legacy: m.Legacy, Release: m.Release, Title: i.Title, State: i.State,
			Ledger: []string{}, Stories: []string{}}
		for _, s := range sprintgoal.SprintBacklog(i.Description) {
			o.Stories = append(o.Stories, project+"#"+s)
		}
		notes, err := t.ListNotes(project, i.IID)
		if err != nil {
			return nil, err
		}
		for _, n := range notes {
			if e, ok := events.Parse(n.Body); ok {
				o.Ledger = append(o.Ledger, e.Name)
				o.Met = o.Met || e.Name == "met"
				o.ReportedComplete = o.ReportedComplete || e.Name == "reported_complete"
			}
		}
		if p, ok := progress[ref]; ok {
			o.Reported = p.Reported
			o.ReportedComplete = o.ReportedComplete || p.Complete || p.Reported == "done"
		}
		out = append(out, o)
	}
	return out, nil
}
