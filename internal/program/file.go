package program

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// FileResult is what file reports.
type FileResult struct {
	Project string     `json:"project"`
	IID     int        `json:"iid"`
	WebURL  string     `json:"web_url"`
	Lint    LintResult `json:"lint"`
	Plane   string     `json:"plane"`
	Kind    string     `json:"kind,omitempty"`
	// Component is the component the issue is labelled with, given or
	// derived from the system repository's subsystem SRDs.
	Component string `json:"component,omitempty"`
	// Derived explains a plane and kind tpm worked out from the paths.
	Derived *Classification `json:"derived,omitempty"`
	// Milestone is the milestone the issue carries; MilestoneReason says why
	// a data issue filed without --milestone carries none.
	Milestone       string `json:"milestone,omitempty"`
	MilestoneReason string `json:"milestone_reason,omitempty"`
	DryRun          bool   `json:"dry_run"`
}

// Labels builds the axis labels for a new issue from its plane, type, kind,
// and component; empty values are left out, and a type's earlier name is
// written as its current one.
func Labels(plane, typ, kind, component string) []string {
	var l []string
	for _, a := range [][2]string{{"plane::", plane}, {"type::", CanonicalType(typ)}, {"kind::", kind}, {"component::", component}} {
		if a[1] != "" {
			l = append(l, a[0]+a[1])
		}
	}
	return l
}

// File creates the issue unassigned and lints it before returning (srd013
// R3.1). A dry run creates nothing and reports how the issue would lint.
func File(t tracker.Tracker, project string, in tracker.NewIssue, p Policy, dryRun bool) (FileResult, error) {
	if err := mayFile(t, in, p); err != nil {
		return FileResult{}, err
	}
	if dryRun {
		would := tracker.Issue{Project: project, Title: in.Title, Description: in.Description, Labels: in.Labels, Milestone: in.Milestone, DueDate: in.DueDate}
		reasons := Lint(would, p)
		return FileResult{Project: project, DryRun: true, Lint: LintResult{Project: project, Admitted: len(reasons) == 0, Reasons: reasons}}, nil
	}
	if c, ok := t.(tracker.AdmittingCreator); ok {
		var r LintResult
		issue, err := c.CreateAdmitted(project, in, func(i tracker.Issue) ([]string, string) {
			reasons := Lint(i, p)
			r = LintResult{Project: project, Admitted: len(reasons) == 0, Reasons: reasons, Changed: true}
			if len(reasons) == 0 {
				return nil, ""
			}
			return reasons, lintNote(reasons)
		})
		if err != nil {
			return FileResult{}, err
		}
		r.IID = issue.IID
		return FileResult{Project: project, IID: issue.IID, WebURL: issue.WebURL, Lint: r}, nil
	}
	issue, err := t.CreateIssue(project, in)
	if err != nil {
		return FileResult{}, err
	}
	r, err := Apply(t, issue, Lint(issue, p), p, false)
	if err != nil {
		return FileResult{}, err
	}
	return FileResult{Project: project, IID: issue.IID, WebURL: issue.WebURL, Lint: r}, nil
}

// SprintMilestone is the active milestone of the project or its groups whose
// start date is on or before today and whose due date is on or after it
// (srd013 R3.3). With none, or more than one, it returns no title and the
// reason; tpm never guesses between sprints.
func SprintMilestone(t tracker.Tracker, project string, today time.Time) (string, string, error) {
	ms, err := t.ActiveMilestones(project)
	if err != nil {
		return "", "", err
	}
	day := today.Format("2006-01-02")
	var covering []string
	for _, m := range ms {
		if m.StartDate != "" && m.DueDate != "" && m.StartDate <= day && day <= m.DueDate && !slices.Contains(covering, m.Title) {
			covering = append(covering, m.Title)
		}
	}
	switch len(covering) {
	case 1:
		return covering[0], "", nil
	case 0:
		return "", "no active milestone with a start and due date covers " + day, nil
	default:
		return "", "several active milestones cover " + day + ": " + strings.Join(covering, ", ") + "; name one with --milestone", nil
	}
}

// mayFile refuses a plane::management issue from a caller the policy does
// not list as a filer (srd001 R1.2, srd035 R5.2). With no usernames listed,
// or the value leads, nothing is checked.
func mayFile(t tracker.Tracker, in tracker.NewIssue, p Policy) error {
	if len(p.ManagementFilers) == 0 || p.ManagementFilers[0] == "leads" {
		return nil
	}
	management := false
	for _, l := range in.Labels {
		management = management || l == "plane::management"
	}
	if !management {
		return nil
	}
	me, err := t.CurrentUser()
	if err != nil {
		return err
	}
	for _, u := range p.ManagementFilers {
		if u == me.Username {
			return nil
		}
	}
	return fmt.Errorf("%s may not file a plane::management issue; control/policy.md lists %s", me.Username, strings.Join(p.ManagementFilers, ", "))
}
