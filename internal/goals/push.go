package goals

import (
	"errors"
	"fmt"

	"github.com/gray-lake-labs/scrum-master/internal/events"
	"github.com/gray-lake-labs/scrum-master/internal/links"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// Pushed is what push reports for one objective.
type Pushed struct {
	Key     string   `json:"key"`
	Action  string   `json:"action"` // created, updated, unchanged, refused
	Project string   `json:"project,omitempty"`
	IID     int      `json:"iid,omitempty"`
	WebURL  string   `json:"web_url,omitempty"`
	Reasons []string `json:"reasons,omitempty"`
	Planned bool     `json:"planned_event_posted"`
}

// Push files or updates one objective issue per confirmed objective and
// posts its planned event (srd022 R4.1, R4.3). An objective with no
// repository, no environment, or an unpinned citation is refused with its
// reason (R5.1); the others proceed.
func Push(t tracker.Tracker, s Sources, p Plan, webBase string, policy program.Policy, dryRun bool) ([]Pushed, error) {
	due := ""
	if p.Milestone != "" {
		if m, err := t.FindMilestone(p.System, p.Milestone); err == nil {
			due = m.DueDate
		} else if !errors.Is(err, tracker.ErrNotFound) {
			return nil, err
		}
	}
	var out []Pushed
	for _, o := range p.Objectives {
		r := Pushed{Key: o.Key, Project: o.Repository}
		d := s.Documents[o.Document]
		if d == nil {
			return out, fmt.Errorf("plan objective %s names a document the release does not", o.Key)
		}
		title, body, cites := Render(webBase, p, o, d)
		if o.Repository == "" {
			r.Reasons = append(r.Reasons, "no repository confirmed")
		}
		if o.Environment == "" {
			r.Reasons = append(r.Reasons, "no environment confirmed")
		}
		for _, c := range cites {
			if err := links.CheckPinned(c); err != nil {
				r.Reasons = append(r.Reasons, err.Error())
			}
		}
		comp, _, ok := s.ComponentFor(o.Repository)
		if o.Repository != "" && !ok {
			r.Reasons = append(r.Reasons, fmt.Sprintf("no subsystem SRD declares %s", o.Repository))
		}
		if len(r.Reasons) > 0 {
			r.Action = "refused"
			out = append(out, r)
			continue
		}
		issueDue := o.Due
		if issueDue == "" {
			issueDue = due
		}
		existing, err := findExisting(t, o.Repository, o.Key, p.Release)
		if err != nil {
			return out, err
		}
		if existing != nil {
			body = KeepNotes(body, existing.Description)
		}
		var issue tracker.Issue
		switch {
		case existing == nil:
			in := tracker.NewIssue{Title: title, Description: body, DueDate: issueDue, Milestone: p.Milestone,
				Labels: program.Labels("management", "", "", comp.Name)}
			f, err := program.File(t, o.Repository, in, policy, dryRun)
			if err != nil {
				return out, err
			}
			r.Action, r.IID, r.WebURL = "created", f.IID, f.WebURL
			issue = tracker.Issue{Project: o.Repository, IID: f.IID}
		case existing.Description == body && existing.DueDate == issueDue:
			r.Action, r.IID, r.WebURL = "unchanged", existing.IID, existing.WebURL
			issue = *existing
		default:
			r.Action, r.IID, r.WebURL = "updated", existing.IID, existing.WebURL
			issue = *existing
			if !dryRun {
				u := tracker.IssueUpdate{Description: &body}
				if issueDue != "" {
					u.DueDate = &issueDue
				}
				if _, err := t.UpdateIssue(o.Repository, existing.IID, u); err != nil {
					return out, err
				}
			}
		}
		if !dryRun || issue.IID != 0 {
			e, text := events.Planned(p.Release, cites, o.Environment)
			posted, err := events.Post(t, issue.Project, issue.IID, e, events.PlannedRequired, text, dryRun)
			if err != nil {
				return out, err
			}
			r.Planned = posted
		} else {
			r.Planned = true
		}
		out = append(out, r)
	}
	return out, nil
}
