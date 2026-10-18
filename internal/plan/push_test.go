package plan

import (
	"slices"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab/gitlabtest"
	"github.com/gray-lake-labs/scrum-master/internal/program"
)

const (
	sys   = "program/system"
	fleet = "program/delivery/fleet"
)

// planned is a system checkout whose road map holds release 01.1, and a
// tracker with the fleet's subsystem SRD and its objective issue filed for
// release 01.0.
func planned(t *testing.T) (PushInput, *gitlabtest.Fake) {
	t.Helper()
	dir := system(t)
	in, _ := input(t, dir)
	c, err := MakeCandidates(in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteRelease(dir, Release{Version: "01.1", Milestone: "01.1", Tag: "corpus/v0.1.0",
		Items: []string{"obj001-reliable-agents#R2.1", "obj002-fast-scoring#R1.1"}}, c, nil, "", 0); err != nil {
		t.Fatal(err)
	}
	f := gitlabtest.New()
	f.AddFile(sys, "main", "docs/specs/software-requirements/srd010-delivery.md", read(t, "../../testdata/system/srd010-delivery.md"))
	f.Issues[fleet] = []*gitlab.Issue{{IID: 4, State: "opened", Labels: []string{"plane::management"},
		Description: "Objective R2\n\n<!-- tpm goals objective=obj001-reliable-agents#R2 release=01.0 -->"}}
	return PushInput{Dir: dir, System: sys, Group: "program", Version: "01.1", End: "2026-11-27",
		WebBase: "https://gitlab.example.com", Policy: program.Defaults(), Ref: "main"}, f
}

func read(t *testing.T, p string) []byte {
	t.Helper()
	return []byte(readFile(t, p))
}

func TestPushSprintUpdatesMilestoneAndChecklist(t *testing.T) {
	in, f := planned(t)
	res, err := Push(f, in)
	if err != nil {
		t.Fatal(err)
	}
	m := f.Milestones["01.1"]
	if res.Milestone != gitlab.MilestoneCreated || m.DueDate != "2026-11-27" || m.StartDate != "2026-11-07" || m.Description != "Release 01.1, 2026-W45 to 2026-W48." {
		t.Fatalf("milestone %s %+v", res.Milestone, m)
	}
	sprint := f.Issues[sys][0]
	if res.Sprint.Action != "filed" || !slices.Contains(sprint.Labels, "type::sprint") || sprint.Milestone != "01.1" ||
		!strings.Contains(sprint.Description, "- [ ] obj001-reliable-agents#R2: A scorecard a product team reads\n- [ ] obj002-fast-scoring#R1: Score in an hour\n") {
		t.Fatalf("sprint issue %s:\n%s", res.Sprint.Action, sprint.Description)
	}
	// The lead ticks one objective; the period moves a week.
	sprint.Description = strings.Replace(sprint.Description, "- [ ] obj002", "- [x] obj002", 1)
	in.End = "2026-12-04"
	res, err = Push(f, in)
	if err != nil {
		t.Fatal(err)
	}
	if res.Milestone != gitlab.MilestoneUpdated || res.Sprint.Action != "updated" || len(f.Issues[sys]) != 1 ||
		!strings.Contains(f.Issues[sys][0].Description, "- [x] obj002-fast-scoring#R1") || f.Issues[sys][0].DueDate != "2026-12-04" {
		t.Fatalf("second push: %+v\n%s", res, f.Issues[sys][0].Description)
	}
}

func TestPlanPushFilesSprintEndIssuesOnce(t *testing.T) {
	in, f := planned(t)
	res, err := Push(f, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.SprintEnds) != 1 || res.SprintEnds[0].Project != fleet || res.SprintEnds[0].Action != "filed" || !res.SprintEnds[0].Admitted {
		t.Fatalf("sprint ends = %+v", res.SprintEnds)
	}
	end := f.Issues[fleet][1]
	if end.DueDate != "2026-11-27" || end.Milestone != "01.1" || !slices.Contains(end.Labels, "component::fleet") ||
		!slices.Contains(end.Labels, "type::recurring") || !strings.Contains(end.Description, "docs/sprint-ends/01.1.md") ||
		!strings.Contains(end.Description, "## Recurrence") {
		t.Fatalf("sprint-end issue %+v", end)
	}
	if len(res.Skipped) != 1 || !strings.Contains(res.Skipped[0], sys) {
		t.Fatalf("skipped = %v", res.Skipped)
	}
	if len(res.Planned) != 1 || res.Planned[0].Issue != fleet+"#4" || !res.Planned[0].Posted {
		t.Fatalf("planned = %+v", res.Planned)
	}
	note := f.Notes[fleet+"#4"][0].Body
	if !strings.Contains(note, "release=01.1") || !strings.Contains(note, "/-/blob/corpus/v0.1.0/docs/specs/objectives/obj001-reliable-agents.md#R2.1") {
		t.Fatalf("planned event:\n%s", note)
	}
	// GitLab stores descriptions without the trailing newline.
	for _, project := range []string{sys, fleet} {
		for _, i := range f.Issues[project] {
			i.Description = strings.TrimRight(i.Description, "\n")
		}
	}
	writes := f.Writes
	res, err = Push(f, in)
	if err != nil {
		t.Fatal(err)
	}
	if f.Writes != writes || res.Milestone != gitlab.MilestoneUnchanged || res.Sprint.Action != "unchanged" ||
		res.SprintEnds[0].Action != "unchanged" || res.Planned[0].Posted || len(f.Issues[fleet]) != 2 {
		t.Fatalf("second push wrote %d: %+v", f.Writes-writes, res)
	}
}
