package goals

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/links"
	"github.com/gray-lake-labs/scrum-master/internal/objectives"
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

// PlannedObjective is one objective (a requirement group) with the release
// items it carries, its recommendations, and any issue already filed.
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

// MakePlan computes the facts and recommendations for every objective of
// the release (srd022 R2, R3.1, R3.2). It asks nothing.
func MakePlan(t gitlab.Tracker, s Sources) (Plan, error) {
	p := Plan{System: s.System, Release: s.Release.Version, Tag: s.Release.Tag, Milestone: s.Release.Milestone}
	candidates := s.Subsystems.Repositories()
	index := map[string]int{}
	for _, id := range s.Release.Objectives {
		doc, item, _ := strings.Cut(id, "#")
		group, _, ok := strings.Cut(item, ".")
		d := s.Documents[doc]
		obj, found := d.Requirements[group]
		if !ok || !found || !hasItem(obj, item) {
			return p, fmt.Errorf("release objective %s names no item of %s", id, doc)
		}
		key := doc + "#" + group
		if i, seen := index[key]; seen {
			p.Objectives[i].Items = append(p.Objectives[i].Items, item)
			continue
		}
		po := PlannedObjective{Key: key, Document: doc, Group: group, Title: obj.Title, Items: []string{item}, Due: obj.Due, Candidates: candidates}
		po.Repository = recommendRepository(s, obj)
		po.Environment = recommendEnvironment(d, group)
		if po.Repository != "" {
			existing, err := findExisting(t, po.Repository, key, p.Release)
			if err != nil {
				return p, err
			}
			if existing != nil {
				po.Existing = existing.WebURL
			}
		}
		index[key] = len(p.Objectives)
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

// marker identifies the objective issue for one objective and release, so
// a second push updates instead of duplicating (srd022 R2.3).
func marker(key, release string) string {
	return fmt.Sprintf("<!-- tpm goals objective=%s release=%s -->", key, release)
}

func findExisting(t gitlab.Tracker, repo, key, release string) (*gitlab.Issue, error) {
	issues, err := t.ListIssues(repo, gitlab.IssueQuery{State: "opened", Labels: []string{"plane::management"}})
	if err != nil {
		if errors.Is(err, gitlab.ErrNotFound) {
			return nil, fmt.Errorf("repository %s does not exist or is not readable", repo)
		}
		return nil, err
	}
	m := marker(key, release)
	for i := range issues {
		if strings.Contains(issues[i].Description, m) {
			return &issues[i], nil
		}
	}
	return nil, nil
}
