package goals

import "github.com/gray-lake-labs/scrum-master/internal/gitlab"

func gitlabRequest() gitlab.IssueQuery { return gitlab.IssueQuery{State: "opened"} }

func gitlabMilestone(due string) gitlab.Milestone {
	return gitlab.Milestone{Title: "2026-W43..W45", DueDate: due}
}
