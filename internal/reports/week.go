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
// and the person's activity since the ISO week began: branches they
// merged and issues assigned to them that closed.
func Week(t tracker.Tracker, project, user string, now time.Time) (WeekResult, error) {
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
	merged, err := t.Merges(project, user, r.Since)
	if err != nil {
		return r, err
	}
	r.Merged = append(r.Merged, merged...)
	closed, err := t.ListIssues(project, tracker.IssueQuery{State: "closed", UpdatedAfter: r.Since})
	if err != nil {
		return r, err
	}
	for _, i := range closed {
		if slices.Contains(i.Assignees, user) {
			r.Closed = append(r.Closed, sprintGoalOf(i))
		}
	}
	return r, nil
}

func sprintGoalOf(i tracker.Issue) SprintGoal {
	return SprintGoal{Ref: i.Project + "#" + itoa(i.IID), Title: i.Title, DueDate: i.DueDate, WebURL: i.WebURL}
}
