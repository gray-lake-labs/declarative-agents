// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package goals

import "github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/tracker"

func openIssues() tracker.IssueQuery { return tracker.IssueQuery{State: "opened"} }

func sprintMilestone(due string) tracker.Milestone {
	return tracker.Milestone{Title: "2026-W43..W45", DueDate: due}
}
