package gitlabtest

import (
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
)

func TestFakeEnsureMilestoneCountsOnlyChanges(t *testing.T) {
	f := New()
	m := gitlab.Milestone{Title: "01.2", DueDate: "2026-11-06"}
	for i, want := range []string{gitlab.MilestoneCreated, gitlab.MilestoneUnchanged} {
		if got, _ := f.EnsureMilestone("program", m); got != want {
			t.Fatalf("run %d: %s, want %s", i, got, want)
		}
	}
	m.DueDate = "2026-11-13"
	if got, _ := f.EnsureMilestone("program", m); got != gitlab.MilestoneUpdated || f.Writes != 2 || len(f.Milestones) != 1 {
		t.Fatalf("moved: %s writes=%d milestones=%d", got, f.Writes, len(f.Milestones))
	}
}
