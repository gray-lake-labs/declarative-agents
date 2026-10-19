package goals

import "github.com/gray-lake-labs/scrum-master/internal/tracker"

func openIssues() tracker.IssueQuery { return tracker.IssueQuery{State: "opened"} }

func sprintMilestone(due string) tracker.Milestone {
	return tracker.Milestone{Title: "2026-W43..W45", DueDate: due}
}
