// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package reports

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/events"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/program"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/sprintgoal"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/tracker"
)

// Signal kinds aggregate reports (srd018 R3.2).
const (
	SignalSharedBlocker  = "shared-blocker"
	SignalAgingBlocker   = "aging-blocker"
	SignalSlipped        = "slipped-commitment"
	SignalMissingReport  = "missing-report"
	SignalContradiction  = "tracker-contradiction"
	SignalUnreportedWork = "unreported-merged-work"
)

// Signal is one finding that no single report shows. It names components,
// blockers, Sprint Goals, and repositories, never a person (srd018 R3.4).
type Signal struct {
	Kind       string `json:"kind"`
	Component  string `json:"component"`
	Subject    string `json:"subject"`
	Week       string `json:"week"`
	Detail     string `json:"detail"`
	Actionable bool   `json:"actionable"`
	// Blocked names the components a shared blocker holds up.
	Blocked []string `json:"blocked,omitempty"`
}

// Key identifies a signal across runs, so a pushed signal is pushed once.
func (s Signal) Key() string {
	return strings.Trim(nonKey.ReplaceAllString(s.Kind+"/"+s.Component+"/"+s.Subject, "-"), "-")
}

var nonKey = regexp.MustCompile(`[^A-Za-z0-9._/#-]+`)

// Progress is one Sprint Goal's state for the range: its latest reported
// status and whether its ledger holds a met event (srd018 R3.5).
type Progress struct {
	SprintGoal string `json:"sprint_goal"`
	Component  string `json:"component"`
	Reported   string `json:"reported"`
	Complete   bool   `json:"reported_complete"`
	Met        bool   `json:"met"`
	State      string `json:"state"` // met, reported complete, or the reported status
}

// Aggregation is what tpm reports aggregate returns.
type Aggregation struct {
	From        string              `json:"from"`
	To          string              `json:"to"`
	Reports     int                 `json:"reports"`
	ByComponent map[string][]string `json:"by_component"` // component -> Sprint Goals and blockers
	Signals     []Signal            `json:"signals"`
	Progress    []Progress          `json:"progress"`
	Pushed      []PushResult        `json:"pushed,omitempty"`
}

// AggregateInput is one run over a range of ISO weeks.
type AggregateInput struct {
	Dir      string // reports checkout
	Config   Config
	Policy   program.Policy
	From, To string // YYYY-Www, inclusive
}

// ParseWeek reads YYYY-Www as the Monday that starts it.
func ParseWeek(w string) (time.Time, error) {
	var year, week int
	if _, err := fmt.Sscanf(w, "%d-W%d", &year, &week); err != nil || week < 1 || week > 53 {
		return time.Time{}, fmt.Errorf("week %q is not YYYY-Www", w)
	}
	return WeekStart(time.Date(year, 1, 4, 0, 0, 0, 0, time.UTC)).AddDate(0, 0, 7*(week-1)), nil
}

type weekly struct {
	week string
	r    Report
}

// Aggregate reads every report in the range, groups their items by
// component, computes the signals, and states each Sprint Goal's progress
// (srd018 R3.1, R3.2, R3.4, R3.5).
func Aggregate(t tracker.Tracker, in AggregateInput) (Aggregation, error) {
	from, err := ParseWeek(in.From)
	if err != nil {
		return Aggregation{}, err
	}
	to, err := ParseWeek(in.To)
	if err != nil {
		return Aggregation{}, err
	}
	if to.Before(from) {
		return Aggregation{}, fmt.Errorf("the range ends at %s before it starts at %s", in.To, in.From)
	}
	var weeks []string
	for w := from; !w.After(to); w = w.AddDate(0, 0, 7) {
		weeks = append(weeks, ISOWeek(w))
	}
	a := Aggregation{From: in.From, To: in.To, ByComponent: map[string][]string{}, Signals: []Signal{}, Progress: []Progress{}}

	var reports []weekly
	err = filepath.WalkDir(filepath.Join(in.Dir, "reports"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		r, err := ParseReport(p, b)
		if err != nil || !slices.Contains(weeks, r.FrontMatter.Sprint) {
			return nil
		}
		reports = append(reports, weekly{r.FrontMatter.Sprint, r})
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return a, err
	}
	sort.SliceStable(reports, func(i, j int) bool { return reports[i].week < reports[j].week })
	a.Reports = len(reports)

	add := func(component, item string) {
		if !slices.Contains(a.ByComponent[component], item) {
			a.ByComponent[component] = append(a.ByComponent[component], item)
		}
	}
	type history struct {
		component string
		statuses  []string // per week in order, "" where unreported
		complete  bool
		latest    string
	}
	goals := map[string]*history{}
	blockedBy := map[string]map[string]bool{} // week/waiting-on -> components
	for _, wr := range reports {
		for _, e := range wr.r.Work {
			add(e["component"], e["sprint_goal"])
			h := goals[e["sprint_goal"]]
			if h == nil {
				h = &history{component: e["component"], statuses: make([]string, len(weeks))}
				goals[e["sprint_goal"]] = h
			}
			h.statuses[slices.Index(weeks, wr.week)] = e["status"]
			h.latest, h.complete = e["status"], e["complete"] == "yes"
		}
		for _, e := range wr.r.Blockers {
			add(e["component"], "blocker: "+e["blocker"])
			k := wr.week + "\x00" + e["waiting-on"]
			if blockedBy[k] == nil {
				blockedBy[k] = map[string]bool{}
			}
			blockedBy[k][e["component"]] = true
			if since, err := time.Parse("2006-01-02", e["since"]); err == nil {
				end, _ := ParseWeek(wr.week)
				if age := int(end.AddDate(0, 0, 6).Sub(since).Hours() / 24); age > in.Policy.Escalation.BlockerAgeDays {
					a.Signals = append(a.Signals, Signal{SignalAgingBlocker, e["component"], e["blocker"], wr.week,
						fmt.Sprintf("%s has been blocked by %q for %d days, past the %d-day threshold", e["component"], e["blocker"], age, in.Policy.Escalation.BlockerAgeDays), true, nil})
				}
			}
		}
	}
	for k, comps := range blockedBy {
		week, waiting, _ := strings.Cut(k, "\x00")
		if countReports(reports, week, waiting) < 2 {
			continue
		}
		names := slices.DeleteFunc(sortedKeys(comps), func(c string) bool { return c == waiting })
		text := fmt.Sprintf("%s blocks %s in %d reports", waiting, strings.Join(names, ", "), countReports(reports, week, waiting))
		if len(names) == 0 {
			text = fmt.Sprintf("%s is blocked on %s's own work in %d reports", waiting, waiting, countReports(reports, week, waiting))
		}
		a.Signals = append(a.Signals, Signal{SignalSharedBlocker, waiting, waiting, week, text, true, names})
	}

	for _, ref := range sortedKeys(goals) {
		h := goals[ref]
		run := 0
		for _, s := range h.statuses {
			if s == "at-risk" || s == "blocked" {
				run++
			} else if s != "" {
				run = 0
			}
		}
		if limit := in.Policy.Escalation.SlipReleases; limit > 0 && run >= limit {
			a.Signals = append(a.Signals, Signal{SignalSlipped, h.component, ref, in.To,
				fmt.Sprintf("Sprint Goal %s (%s) has been at risk or blocked in %d consecutive reports", ref, h.component, run), true, nil})
		}
		met, err := hasEvent(t, ref, "met")
		if err != nil {
			return a, err
		}
		closedComplete, err := hasEvent(t, ref, "reported_complete")
		if err != nil {
			return a, err
		}
		p := Progress{SprintGoal: ref, Component: h.component, Reported: h.latest, Complete: h.complete || closedComplete, Met: met, State: h.latest}
		switch {
		case met:
			p.State = "met"
		case closedComplete:
			// A sprint-end merge closed it as reported complete, after the
			// last report that named its status (srd038 R2.2).
			p.State = "reported complete"
		case h.complete || h.latest == "done":
			p.State = "reported complete"
			// Closed without met, the Sprint Goal closed as reported complete
			// by its repository's sprint-end merge (srd038 R2); still open,
			// the report contradicts the tracker.
			closed, err := isClosed(t, ref)
			if err != nil {
				return a, err
			}
			if !closed {
				a.Signals = append(a.Signals, Signal{SignalContradiction, h.component, ref, in.To,
					fmt.Sprintf("Sprint Goal %s (%s) is reported complete but its ledger has no met event", ref, h.component), true, nil})
			}
		}
		a.Progress = append(a.Progress, p)
	}

	have := map[string]bool{}
	for _, wr := range reports {
		have[wr.week+"\x00"+wr.r.FrontMatter.Author] = true
	}
	for _, m := range in.Config.Team {
		for _, w := range weeks {
			if !have[w+"\x00"+m.Username] {
				a.Signals = append(a.Signals, Signal{SignalMissingReport, m.Project, m.Project, w,
					fmt.Sprintf("no status report for %s in %s", m.Project, w), false, nil})
			}
		}
	}
	// A team whose reports name no Sprint Goal of its own is as missing as
	// one that sent none: nothing of it reaches the aggregate. The system
	// repository holds no Sprint Goal (srd018 R3.2).
	var projects []string
	sent := map[string]bool{}
	for _, m := range in.Config.Team {
		if !slices.Contains(projects, m.Project) {
			projects = append(projects, m.Project)
		}
		for _, w := range weeks {
			sent[m.Project] = sent[m.Project] || have[w+"\x00"+m.Username]
		}
	}
	for _, project := range projects {
		if project == in.Config.System || !sent[project] {
			continue
		}
		named := false
		for ref := range goals {
			if strings.HasPrefix(ref, project+"#") {
				named = true
				break
			}
		}
		if !named {
			a.Signals = append(a.Signals, Signal{SignalMissingReport, project, project, weeks[len(weeks)-1],
				fmt.Sprintf("no report from %s to %s names a Sprint Goal of %s", in.From, in.To, project), false, nil})
		}
	}

	mentioned := map[string]bool{}
	for ref := range goals {
		mentioned[ref] = true
	}
	end := to.AddDate(0, 0, 6).Format("2006-01-02")
	var reported []tracker.Issue
	for ref := range goals {
		if i, ok, err := issueOf(t, ref); err != nil {
			return a, err
		} else if ok {
			reported = append(reported, i)
		}
	}
	since := periodStart(from, reported, sprintgoal.Is).Format(time.RFC3339)
	for _, project := range projects {
		merged, err := t.Merges(project, "", since)
		if err != nil {
			return a, err
		}
		for _, mr := range merged {
			// A delivery closes an issue it cites; a merge citing none,
			// such as a pull request's, is not work the reports account for,
			// and neither is one closing only recurring issues, such as a
			// sprint-end or a status report.
			if mr.MergedAt > end || !strings.Contains(mr.Description, "Closes ") || closesMentioned(project, mr.Description, mentioned) {
				continue
			}
			if routine, err := closesOnlyRecurring(t, project, mr.Description); err != nil {
				return a, err
			} else if routine {
				continue
			}
			a.Signals = append(a.Signals, Signal{SignalUnreportedWork, project, fmt.Sprintf("%s!%d", project, mr.IID), ISOWeek(mustDate(mr.MergedAt)),
				fmt.Sprintf("%s!%d %q merged on %s and closes no reported Sprint Goal", project, mr.IID, mr.Title, mr.MergedAt), false, nil})
		}
	}
	for c := range a.ByComponent {
		sort.Strings(a.ByComponent[c])
	}
	sort.SliceStable(a.Signals, func(i, j int) bool {
		if a.Signals[i].Kind != a.Signals[j].Kind {
			return a.Signals[i].Kind < a.Signals[j].Kind
		}
		return a.Signals[i].Key() < a.Signals[j].Key()
	})
	return a, nil
}

func countReports(reports []weekly, week, waiting string) int {
	n := 0
	for _, wr := range reports {
		if wr.week != week {
			continue
		}
		for _, e := range wr.r.Blockers {
			if e["waiting-on"] == waiting {
				n++
				break
			}
		}
	}
	return n
}

var closesPattern = regexp.MustCompile(`(?i)closes\s+(\S*#\d+)`)

// closesMentioned reports whether the merge closes a Sprint Goal a
// report names.
func closesMentioned(project, description string, mentioned map[string]bool) bool {
	for _, m := range closesPattern.FindAllStringSubmatch(description, -1) {
		ref := m[1]
		if strings.HasPrefix(ref, "#") {
			ref = project + ref
		}
		if mentioned[ref] {
			return true
		}
	}
	return false
}

// closesOnlyRecurring reports whether every issue the merge closes is a
// recurring one (type::recurring): routine, not planned work.
func closesOnlyRecurring(t tracker.Tracker, project, description string) (bool, error) {
	refs := closesPattern.FindAllStringSubmatch(description, -1)
	for _, m := range refs {
		ref := m[1]
		if strings.HasPrefix(ref, "#") {
			ref = project + ref
		}
		i, ok, err := issueOf(t, ref)
		if err != nil || !ok || !i.HasLabel("type::recurring") {
			return false, err
		}
	}
	return len(refs) > 0, nil
}

// hasMet reports whether the Sprint Goal's ledger holds a met event
// (srd006 R10).
// isClosed reports whether the Sprint Goal ref is closed; an issue the
// tracker cannot find is not.
// issueOf reads the issue a project#iid reference names; ok is false when
// the reference is malformed or names no issue.
func issueOf(t tracker.Tracker, ref string) (tracker.Issue, bool, error) {
	project, n, ok := strings.Cut(ref, "#")
	iid, err := strconv.Atoi(n)
	if !ok || err != nil {
		return tracker.Issue{}, false, nil
	}
	i, err := t.GetIssue(project, iid)
	if errors.Is(err, tracker.ErrNotFound) {
		return tracker.Issue{}, false, nil
	}
	return i, err == nil, err
}

func isClosed(t tracker.Tracker, ref string) (bool, error) {
	i, ok, err := issueOf(t, ref)
	return ok && i.State == "closed", err
}

// hasEvent reports whether the issue ref's ledger holds an event named name.
func hasEvent(t tracker.Tracker, ref, name string) (bool, error) {
	project, n, ok := strings.Cut(ref, "#")
	iid, err := strconv.Atoi(n)
	if !ok || err != nil {
		return false, nil
	}
	notes, err := t.ListNotes(project, iid)
	if errors.Is(err, tracker.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, note := range notes {
		if e, ok := events.Parse(note.Body); ok && e.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func mustDate(s string) time.Time {
	d, _ := time.Parse("2006-01-02", s)
	return d
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
