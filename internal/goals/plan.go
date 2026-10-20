package goals

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/links"
	"github.com/gray-lake-labs/scrum-master/internal/objectives"
	"github.com/gray-lake-labs/scrum-master/internal/sprintgoal"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// Plan is what tpm goals plan returns and tpm goals push takes back once the
// person has confirmed or changed each Repository and Environment.
type Plan struct {
	System     string             `json:"system"`
	Release    string             `json:"release"`
	Tag        string             `json:"tag"`
	Milestone  string             `json:"milestone,omitempty"`
	Objectives []PlannedObjective `json:"objectives"`
}

// PlannedObjective is one initiative item the release carries, with the
// outcome (requirement group) it belongs to, its recommendations, and the
// Sprint Goal or objective issue already filed for it. An outcome's items
// come one per entry, so the person can confirm each for a different team
// (srd022, srd034: one Cites line per item the team serves). Push files one
// Sprint Goal per team serving every item confirmed for its repository.
type PlannedObjective struct {
	Key         string   `json:"key"` // objNNN-name#Rn
	Document    string   `json:"document"`
	Group       string   `json:"group"`
	Title       string   `json:"title"`
	Items       []string `json:"items"`
	Due         string   `json:"due"`
	Repository  string   `json:"repository"`  // recommended; the person confirms or changes it
	Environment string   `json:"environment"` // recommended; the person confirms or changes it
	Candidates  []string `json:"candidates"`
	Existing    string   `json:"existing,omitempty"` // web URL of an issue already filed for the release
}

// MakePlan computes the facts and recommendations for every initiative
// item of the release (srd022 R2, R3.1, R3.2). It asks nothing.
func MakePlan(t tracker.Tracker, s Sources) (Plan, error) {
	p := Plan{System: s.System, Release: s.Release.Version, Tag: s.Release.Tag, Milestone: s.Release.Milestone}
	candidates := s.Subsystems.Repositories()
	for _, id := range s.Release.Objectives {
		doc, item, _ := strings.Cut(id, "#")
		group, _, ok := strings.Cut(item, ".")
		d := s.Documents[doc]
		obj, found := d.Requirements[group]
		if !ok || !found || !hasItem(obj, item) {
			return p, fmt.Errorf("release objective %s names no item of %s", id, doc)
		}
		key := doc + "#" + group
		po := PlannedObjective{Key: key, Document: doc, Group: group, Title: obj.Title, Items: []string{item}, Due: obj.Due, Candidates: candidates}
		po.Repository = recommendRepository(s, obj)
		po.Environment = recommendEnvironment(d, group)
		if po.Repository != "" {
			goal, legacy, err := findExisting(t, po.Repository, p.Release)
			if err != nil {
				return p, err
			}
			if old, ok := legacy[key]; ok {
				po.Existing = old.WebURL
			} else if goal != nil {
				po.Existing = goal.WebURL
			}
		}
		p.Objectives = append(p.Objectives, po)
	}
	return p, nil
}

func hasItem(o objectives.Objective, id string) bool {
	for _, it := range o.ItemsOf() {
		if it.ID == id {
			return true
		}
	}
	return false
}

// recommendRepository is the repository of a delivered_by reference that a
// subsystem SRD declares; empty when none does, so the person chooses.
func recommendRepository(s Sources, o objectives.Objective) string {
	for _, ref := range o.DeliveredBy {
		r, err := links.Parse(ref)
		if err != nil {
			continue
		}
		if c, _, ok := s.ComponentFor(r.Project); ok {
			return projectPath(c.Repository)
		}
	}
	return ""
}

// recommendEnvironment is the environment the objective's first
// acceptance criterion names, else dev (srd022 R3.2).
func recommendEnvironment(d *objectives.Document, group string) string {
	if c := d.CriteriaFor(group); len(c) > 0 && c[0].Environment != "" {
		return c[0].Environment
	}
	return "dev"
}

// findExisting returns the team's Sprint Goal for the release in repo, and
// the objective issues filed there for the release before 04.0, keyed by the
// outcome each serves (srd022 R2.3, srd034 R3.7).
func findExisting(t tracker.Tracker, repo, release string) (*tracker.Issue, map[string]tracker.Issue, error) {
	issues, err := t.ListIssues(repo, tracker.IssueQuery{State: "opened", Labels: []string{"plane::management"}})
	if err != nil {
		if errors.Is(err, tracker.ErrNotFound) {
			return nil, nil, fmt.Errorf("repository %s does not exist or is not readable", repo)
		}
		return nil, nil, err
	}
	var goal *tracker.Issue
	legacy := map[string]tracker.Issue{}
	for i := range issues {
		r, ok := sprintgoal.Parse(issues[i].Description)
		if !ok || r.Release != release {
			continue
		}
		if r.Legacy {
			legacy[r.Serves[0]] = issues[i]
		} else if goal == nil {
			goal = &issues[i]
		}
	}
	return goal, legacy, nil
}
