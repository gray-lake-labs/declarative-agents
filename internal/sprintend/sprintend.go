// Package sprintend is the tpm side of a sprint-end issue (srd038 R1, R6):
// the facts its Requirements summarize, so the model drafts the period's
// summary from computed state and asks only what tpm cannot know.
package sprintend

import (
	"fmt"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/events"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/release"
	"github.com/gray-lake-labs/scrum-master/internal/reports"
)

// Objective is one objective issue of the repository and its state.
type Objective struct {
	Issue            string   `json:"issue"`
	Objective        string   `json:"objective"` // objNNN-short-name#Rn from create-goals' marker
	Release          string   `json:"release"`
	Title            string   `json:"title"`
	State            string   `json:"state"` // opened or closed
	Reported         string   `json:"reported,omitempty"`
	ReportedComplete bool     `json:"reported_complete"`
	Met              bool     `json:"met"`
	Ledger           []string `json:"ledger"` // the event names posted on it, in order
}

// Facts is what tpm sprint-end facts returns.
type Facts struct {
	Project     string              `json:"project"`
	Aggregate   reports.Aggregation `json:"aggregate"`
	Objectives  []Objective         `json:"objectives"`
	Next        release.NextResult  `json:"next"`
	Recommended string              `json:"recommended_version"` // the next minor version (srd015 R4.1)
}

// Input is one sprint-end facts run.
type Input struct {
	Tracker   gitlab.Tracker
	Project   string
	Aggregate reports.Aggregation
	Releases  program.Releases
	Artifact  string
	Tags      []string
}

const goalsMarker = "<!-- tpm goals objective="

// MakeFacts lists the repository's objective issues with their ledger and
// reported status, beside the period's aggregate and the version tpm
// release next recommends.
func MakeFacts(in Input) (Facts, error) {
	f := Facts{Project: in.Project, Aggregate: in.Aggregate}
	// An objective issue is known by create-goals' marker, not by a label.
	issues, err := in.Tracker.ListIssues(in.Project, gitlab.IssueQuery{State: "all"})
	if err != nil {
		return f, err
	}
	progress := map[string]reports.Progress{}
	for _, p := range in.Aggregate.Progress {
		progress[p.Objective] = p
	}
	for _, i := range issues {
		_, rest, ok := strings.Cut(i.Description, goalsMarker)
		if !ok {
			continue
		}
		key, rest, _ := strings.Cut(rest, " ")
		rel, _, _ := strings.Cut(strings.TrimPrefix(rest, "release="), " ")
		ref := fmt.Sprintf("%s#%d", in.Project, i.IID)
		o := Objective{Issue: ref, Objective: key, Release: rel, Title: i.Title, State: i.State, Ledger: []string{}}
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
		f.Objectives = append(f.Objectives, o)
	}
	f.Next = release.Next(in.Releases, in.Artifact, in.Tags)
	for _, c := range f.Next.Candidates {
		if c.Bump == "minor" {
			f.Recommended = c.Version
		}
	}
	return f, nil
}
