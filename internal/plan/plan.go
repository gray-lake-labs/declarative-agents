// Package plan is the tpm side of plan in the system repository
// (srd031): it ranks the objective items the next release could prove and
// writes the chosen release into the road map. Nothing here is a model
// step (design.md D10).
package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/corpus"
	"github.com/gray-lake-labs/scrum-master/internal/goals"
	"github.com/gray-lake-labs/scrum-master/internal/objectives"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/release"
	"github.com/gray-lake-labs/scrum-master/internal/reports"
	"github.com/gray-lake-labs/scrum-master/internal/sprintgoal"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// RoadMap is the road map's path in a repository checkout.
const RoadMap = "docs/road-map.md"

// periodDays is the length of a period: three weeks (srd031 R3.2).
const periodDays = 21

// Item is one objective item a release can prove.
type Item struct {
	ID      string   `json:"id"`    // objNNN-short-name#Rn.m
	Group   string   `json:"group"` // objNNN-short-name#Rn
	Title   string   `json:"title"` // the objective's title
	Text    string   `json:"text"`
	Due     string   `json:"due,omitempty"`
	Slipped bool     `json:"slipped,omitempty"`
	Placed  string   `json:"placed_in,omitempty"` // the release that already lists it
	Missing []string `json:"missing,omitempty"`   // test, environment
}

// Candidates is what tpm plan candidates returns (srd031 R2, R3).
type Candidates struct {
	Latest          string `json:"latest_release,omitempty"`
	Version         string `json:"recommended_version"`
	EndDate         string `json:"recommended_end_date,omitempty"`
	Tag             string `json:"recommended_tag,omitempty"`
	Items           []Item `json:"items"`
	WithoutEvidence []Item `json:"without_evidence,omitempty"`
	// Backlog is the team's Product Backlog, when the checkout holds stories
	// (srd031 R6.4).
	Backlog *Backlog `json:"backlog,omitempty"`

	agreed map[string]bool // every item of an agreed objective document
}

// Input is where candidates come from: the system repository's checkout,
// the tracker for slipped objective issues and the last milestone, and
// the period's aggregate, if the caller has one.
type Input struct {
	Dir       string
	System    string
	Tracker   tracker.Tracker
	Aggregate *reports.Aggregation
	Releases  program.Releases
	Artifact  string
	Tags      []string
	Now       time.Time
}

// MakeCandidates ranks the agreed objective items not yet placed in a
// release, plus the placed ones whose objective slipped: slipped first,
// then by due date, then in document order (srd031 R3.1). Items whose
// criteria lack a test or an environment come back apart (R4.4, R5.1).
func MakeCandidates(in Input) (Candidates, error) {
	var c Candidates
	releases, err := readRoadMap(in.Dir)
	if err != nil {
		return c, err
	}
	placed := map[string]string{}
	for _, r := range releases {
		for _, id := range r.Objectives {
			placed[id] = r.Version
		}
	}
	slipped, err := slippedGroups(in)
	if err != nil {
		return c, err
	}
	docs, err := objectives.LoadAll(in.Dir)
	if err != nil {
		return c, err
	}
	var items []Item
	c.agreed = map[string]bool{}
	for _, d := range docs {
		if d.Status != "agreed" {
			continue
		}
		for _, gid := range d.ObjectiveIDs() {
			o := d.Requirements[gid]
			group := d.ID + "#" + gid
			for _, it := range o.ItemsOf() {
				item := Item{ID: d.ID + "#" + it.ID, Group: group, Title: o.Title, Text: it.Text, Due: o.Due,
					Slipped: slipped[group], Placed: placed[d.ID+"#"+it.ID], Missing: missing(d, gid, it.ID)}
				c.agreed[item.ID] = true
				switch {
				case item.Placed != "" && !item.Slipped:
				case len(item.Missing) > 0:
					c.WithoutEvidence = append(c.WithoutEvidence, item)
				default:
					items = append(items, item)
				}
			}
		}
	}
	slices.SortStableFunc(items, func(a, b Item) int {
		switch {
		case a.Slipped != b.Slipped:
			if a.Slipped {
				return -1
			}
			return 1
		case a.Due != b.Due:
			if a.Due == "" {
				return 1
			}
			if b.Due == "" {
				return -1
			}
			return strings.Compare(a.Due, b.Due)
		}
		return 0
	})
	c.Items = items
	if n := len(releases); n > 0 && unplanned(releases[n-1]) {
		// tpm init lays down 01.0 with no items and no tag: that record is
		// the next release, not a released one.
		c.Version = releases[n-1].Version
		if n > 1 {
			c.Latest = releases[n-2].Version
		}
	} else {
		if n > 0 {
			c.Latest = releases[n-1].Version
		}
		c.Version = nextVersion(c.Latest)
	}
	c.EndDate = endDate(in, releases)
	next := release.Next(in.Releases, in.Artifact, in.Tags)
	for _, cand := range next.Candidates {
		if cand.Bump == "minor" {
			c.Tag = cand.Tag
		}
	}
	return c, nil
}

func readRoadMap(dir string) ([]goals.Release, error) {
	b, err := os.ReadFile(filepath.Join(dir, RoadMap))
	if err != nil {
		return nil, err
	}
	d, err := corpus.ParseMarkdown(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", RoadMap, err)
	}
	return goals.ReleasesOf(d), nil
}

// missing names what an item's acceptance criteria lack: a criterion that
// traces it, and on each such criterion its tests and its environment.
func missing(d *objectives.Document, group, item string) []string {
	var tracing []objectives.Criterion
	for _, c := range d.CriteriaFor(group) {
		if slices.Contains(c.Traces, item) {
			tracing = append(tracing, c)
		}
	}
	if len(tracing) == 0 {
		return []string{"test", "environment"}
	}
	var out []string
	for _, c := range tracing {
		if len(c.Tests) == 0 && !slices.Contains(out, "test") {
			out = append(out, "test")
		}
		if c.Environment == "" && !slices.Contains(out, "environment") {
			out = append(out, "environment")
		}
	}
	return out
}

// slippedGroups maps each initiative outcome whose Sprint Goal the aggregate
// names in a slipped-commitment signal; the Sprint Goal's tasks marker names
// the outcomes it serves (srd018 R3.2, srd031 R2.3).
func slippedGroups(in Input) (map[string]bool, error) {
	out := map[string]bool{}
	if in.Aggregate == nil || in.Tracker == nil {
		return out, nil
	}
	for _, s := range in.Aggregate.Signals {
		if s.Kind != reports.SignalSlipped {
			continue
		}
		project, iid, err := release.ParseIssueRef(s.Subject)
		if err != nil {
			continue
		}
		issue, err := in.Tracker.GetIssue(project, iid)
		if err != nil {
			return nil, fmt.Errorf("slipped Sprint Goal %s: %w", s.Subject, err)
		}
		if m, ok := sprintgoal.Parse(issue.Description); ok {
			for _, key := range m.Serves {
				out[key] = true
			}
		}
	}
	return out, nil
}

// teamRepository reports whether the checkout at dir is a team's
// repository, by the subsystem SRD tpm onboard writes there.
func teamRepository(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "docs", "specs", "software-requirements", "srd001-subsystem.md"))
	return err == nil
}

// unplanned reports whether a release record names no items and no tag:
// a release laid down to be planned, not one planning has written.
func unplanned(r goals.Release) bool { return len(r.Objectives) == 0 && r.Tag == "" }

// nextVersion is the road map's next minor version: 01.1 is followed by
// 01.2, and an empty road map starts at 01.0.
func nextVersion(latest string) string {
	major, minor, ok := strings.Cut(latest, ".")
	n, err := strconv.Atoi(minor)
	if !ok || err != nil {
		return "01.0"
	}
	return fmt.Sprintf("%s.%0*d", major, len(minor), n+1)
}

// endDate recommends the end of the next three-week period: three weeks
// after the latest release's milestone ends, else three weeks from now.
func endDate(in Input, releases []goals.Release) string {
	if len(releases) > 0 && in.Tracker != nil && in.System != "" {
		if name := releases[len(releases)-1].Milestone; name != "" {
			if m, err := in.Tracker.FindMilestone(in.System, name); err == nil && m.DueDate != "" {
				if due, err := time.Parse(time.DateOnly, m.DueDate); err == nil {
					return due.AddDate(0, 0, periodDays).Format(time.DateOnly)
				}
			}
		}
	}
	return in.Now.AddDate(0, 0, periodDays).Format(time.DateOnly)
}
