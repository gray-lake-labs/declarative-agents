package standup

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/tracker"
	"github.com/gray-lake-labs/scrum-master/internal/trackertest"
)

func TestRecurReopensWithNextDueDateAndPreviousRuns(t *testing.T) {
	trackertest.Each(t, func(t *testing.T, p trackertest.Program) {
		f := p.Tracker()
		labels := []string{"plane::data", "type::recurring", "kind::review", "component::po-1"}
		first, _ := f.CreateIssue("g/po-1", tracker.NewIssue{Title: "Recurring: regression run", Labels: append(slices.Clone(labels), "op::status"), DueDate: "2026-10-15", Milestone: "rig-01.0",
			Description: "Run the suite.\n\n## Recurrence\n\nWeekly.\n\n## Previous Runs\n\n- none\n"})
		now := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)

		r, err := Recur(f, "g/po-1", first.IID, RecurOptions{Now: now})
		if err != nil || !r.Closed || r.Existing || r.DueDate != "2026-10-22" {
			t.Fatalf("recur open issue: %+v %v", r, err)
		}
		old, _ := f.GetIssue("g/po-1", first.IID)
		notes, _ := f.ListNotes("g/po-1", first.IID)
		if old.State != "closed" || len(notes) != 1 || !strings.Contains(notes[0].Body, "Run summary") {
			t.Fatalf("previous instance %s, notes %+v", old.State, notes)
		}
		next, _ := f.GetIssue("g/po-1", r.Next)
		if next.Title != first.Title || !slices.Equal(next.Labels, append(labels, "op::recur")) || next.DueDate != "2026-10-22" || next.Milestone != "rig-01.0" {
			t.Fatalf("next instance %+v", next)
		}
		if !strings.Contains(next.Description, "## Recurrence") || !strings.Contains(next.Description, "- g/po-1#1 (due 2026-10-15): closed by tpm issue recur without its work.") || strings.Contains(next.Description, "- none") {
			t.Fatalf("next body:\n%s", next.Description)
		}

		again, err := Recur(f, "g/po-1", first.IID, RecurOptions{Now: now})
		all, _ := f.ListIssues("g/po-1", tracker.IssueQuery{State: "all"})
		if err != nil || !again.Existing || again.Next != r.Next || len(all) != 2 {
			t.Fatalf("second recur: %+v %v, %d issues", again, err, len(all))
		}

		f.UpdateIssue("g/po-1", r.Next, tracker.IssueUpdate{State: ptrTo("close")})
		third, err := Recur(f, "g/po-1", r.Next, RecurOptions{Now: now.AddDate(0, 0, 7)})
		if err != nil || third.Closed || third.DueDate != "2026-10-29" {
			t.Fatalf("recur after a merge closed it: %+v %v", third, err)
		}
		body, _ := f.GetIssue("g/po-1", third.Next)
		if strings.Count(body.Description, "\n- g/po-1#") != 2 || !strings.Contains(body.Description, "#2 (due 2026-10-22): closed by its merge.") ||
			strings.Count(body.Description, "<!-- tpm standup recurs") != 1 {
			t.Fatalf("third body:\n%s", body.Description)
		}
	})
}

func ptrTo[T any](v T) *T { return &v }

func TestRecurOpensNextInstanceOnce(t *testing.T) {
	trackertest.Each(t, func(t *testing.T, p trackertest.Program) {
		f := p.Tracker()
		first, _ := f.CreateIssue("g/po-1", tracker.NewIssue{Title: "Recurring: audit", Labels: []string{"type::recurring"}, DueDate: "2026-10-15",
			Description: "Audit.\n\n## Previous Runs\n\n- none\n"})
		now := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
		r, err := Recur(f, "g/po-1", first.IID, RecurOptions{Now: now})
		if err != nil || r.Existing || r.Next == 0 {
			t.Fatalf("first recur: %+v %v", r, err)
		}
		again, err := Recur(f, "g/po-1", first.IID, RecurOptions{Now: now})
		all, _ := f.ListIssues("g/po-1", tracker.IssueQuery{State: "all"})
		if err != nil || !again.Existing || again.Next != r.Next || len(all) != 2 {
			t.Fatalf("second recur opened another instance: %+v %v, %d issues", again, err, len(all))
		}
	})
}

// An instance opened before recur moved to the issue group carries the
// marker tpm standup recur wrote; recur must find it and open none.
func TestRecurFindsEarlierMarker(t *testing.T) {
	trackertest.Each(t, func(t *testing.T, p trackertest.Program) {
		f := p.Tracker()
		first, _ := f.CreateIssue("g/po-1", tracker.NewIssue{Title: "Recurring: audit", Labels: []string{"type::recurring"}, DueDate: "2026-10-15",
			Description: "Audit.\n\n## Previous Runs\n\n- none\n"})
		earlier, _ := f.CreateIssue("g/po-1", tracker.NewIssue{Title: "Recurring: audit", Labels: []string{"type::recurring"}, DueDate: "2026-10-22",
			Description: "Audit.\n\n## Previous Runs\n\n- g/po-1#1 (due 2026-10-15): closed by tpm standup recur without its work.\n\n<!-- tpm standup recurs g/po-1#1 -->\n"})
		r, err := Recur(f, "g/po-1", first.IID, RecurOptions{Now: time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)})
		all, _ := f.ListIssues("g/po-1", tracker.IssueQuery{State: "all"})
		if err != nil || !r.Existing || r.Next != earlier.IID || len(all) != 2 {
			t.Fatalf("recur missed the earlier instance: %+v %v, %d issues", r, err, len(all))
		}
	})
}
