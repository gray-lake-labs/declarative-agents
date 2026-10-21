package reports

import (
	"slices"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/sprintgoal"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// SprintGoal is an open Sprint Goal issue, or an objective issue read as
// one, the person reports on.
type SprintGoal struct {
	Ref     string `json:"ref"`
	Title   string `json:"title"`
	DueDate string `json:"due_date,omitempty"`
	WebURL  string `json:"web_url"`
}

// WeekResult is the facts the status issue's script drafts from (srd018
// R8.1).
type WeekResult struct {
	Project     string          `json:"project"`
	User        string          `json:"user"`
	Week        string          `json:"week"`
	Since       string          `json:"since"`
	SprintGoals []SprintGoal    `json:"sprint_goals"`
	Merged      []tracker.Merge `json:"merged"`
	Closed      []SprintGoal    `json:"closed"`
}

// Week returns the project's open Sprint Goals with their due dates
// and the person's activity since the ISO week began, in UTC: branches
// they merged and issues assigned to them that closed. A merge from before
// the period's first Sprint Goal is not the period's work, so it does not
// count even inside the week (periodStart).
func Week(t tracker.Tracker, project, user string, now time.Time) (WeekResult, error) {
	now = now.UTC()
	start := WeekStart(now)
	r := WeekResult{Project: project, User: user, Week: ISOWeek(now), Since: start.Format("2006-01-02"),
		SprintGoals: []SprintGoal{}, Merged: []tracker.Merge{}, Closed: []SprintGoal{}}
	open, err := t.ListIssues(project, tracker.IssueQuery{State: "opened"})
	if err != nil {
		return r, err
	}
	for _, i := range open {
		if sprintgoal.Is(i.Description) {
			r.SprintGoals = append(r.SprintGoals, sprintGoalOf(i))
		}
	}
	closed, err := t.ListIssues(project, tracker.IssueQuery{State: "closed", UpdatedAfter: r.Since})
	if err != nil {
		return r, err
	}
	merged, err := t.Merges(project, user, periodStart(start, append(slices.Clone(open), closed...)))
	if err != nil {
		return r, err
	}
	r.Merged = append(r.Merged, merged...)
	for _, i := range closed {
		if slices.Contains(i.Assignees, user) {
			r.Closed = append(r.Closed, sprintGoalOf(i))
		}
	}
	return r, nil
}

// periodStart is the later of start and the creation of the earliest
// Sprint Goal among issues, as an RFC 3339 instant: merges before the
// period's first Sprint Goal, such as an earlier run's on the same days,
// deliver nothing the period planned (srd018 R3.2, R8.1).
func periodStart(start time.Time, issues []tracker.Issue) string {
	first := time.Time{}
	for _, i := range issues {
		if !sprintgoal.Is(i.Description) {
			continue
		}
		if at, err := time.Parse(time.RFC3339, i.CreatedAt); err == nil && (first.IsZero() || at.Before(first)) {
			first = at
		}
	}
	if first.After(start) {
		start = first
	}
	return start.UTC().Format(time.RFC3339)
}

func sprintGoalOf(i tracker.Issue) SprintGoal {
	return SprintGoal{Ref: i.Project + "#" + itoa(i.IID), Title: i.Title, DueDate: i.DueDate, WebURL: i.WebURL}
}
