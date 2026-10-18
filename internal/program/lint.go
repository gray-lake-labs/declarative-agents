package program

import (
	"bufio"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/links"
)

// lintMarker heads every admission comment, so a repeated failure with the
// same reasons posts nothing new.
const lintMarker = "<!-- tpm program lint -->"

var (
	evidencePattern = regexp.MustCompile(`\(test:\s*[^;)]+;\s*environment:\s*[^)]+\)`)
	numberedEntry   = regexp.MustCompile(`^\d+\. `)
)

// Lint checks one issue against the label schema and admission policy
// (srd013 R1.1) and returns every reason it fails; none means admitted.
func Lint(i gitlab.Issue, p Policy) []string {
	var reasons []string
	axis := func(prefix string) []string {
		var vals []string
		for _, l := range i.Labels {
			if v, ok := strings.CutPrefix(l, prefix); ok {
				vals = append(vals, v)
			}
		}
		return vals
	}
	listed := func(name string, vals, allowed []string) {
		for _, v := range vals {
			if len(allowed) > 0 && !slices.Contains(allowed, v) {
				reasons = append(reasons, fmt.Sprintf("%s::%s is not a listed %s", name, v, name))
			}
		}
	}

	planes := axis("plane::")
	if len(planes) != 1 {
		reasons = append(reasons, fmt.Sprintf("needs exactly one plane:: label, has %d", len(planes)))
	}
	listed("plane", planes, p.Planes)

	types := axis("type::")
	if len(types) > 1 {
		reasons = append(reasons, fmt.Sprintf("has %d type:: labels, at most one", len(types)))
	}
	listed("type", types, p.Types)
	sprint := slices.Contains(types, "sprint")

	components := axis("component::")
	if !sprint && len(components) != 1 {
		reasons = append(reasons, fmt.Sprintf("needs exactly one component:: label, has %d", len(components)))
	}
	listed("component", components, p.Components)

	kinds := axis("kind::")
	data := slices.Contains(planes, "data")
	if data && len(types) == 0 && len(kinds) != 1 {
		reasons = append(reasons, fmt.Sprintf("a data issue needs exactly one kind:: label, has %d", len(kinds)))
	}
	if !data && len(kinds) > 0 {
		reasons = append(reasons, "only data issues carry a kind:: label")
	}
	listed("kind", kinds, p.Kinds)

	if data && len(types) == 0 && p.RequireMilestone && i.Milestone == "" {
		reasons = append(reasons, "a data issue needs a sprint milestone")
	}

	cites := links.ParseCites(i.Description)
	needAC := len(cites) > 0
	for _, k := range kinds {
		needAC = needAC || slices.Contains(p.RequireAC, k)
	}
	reqs, haveReqs := section(i.Description, "Requirements")
	criteria, haveAC := section(i.Description, "Acceptance Criteria")
	if needAC && (!haveReqs || len(reqs) == 0) {
		reasons = append(reasons, "needs a ## Requirements section with at least one entry")
	}
	if needAC && (!haveAC || len(criteria) == 0) {
		reasons = append(reasons, "needs a ## Acceptance Criteria section with at least one entry")
	}
	for _, c := range criteria {
		if !evidencePattern.MatchString(c) {
			reasons = append(reasons, fmt.Sprintf("criterion %q names no (test: T; environment: E)", shorten(c)))
		}
	}
	for _, c := range cites {
		if err := links.CheckPinned(c.URL); err != nil {
			reasons = append(reasons, "Cites line is not pinned: "+err.Error())
		}
	}

	if slices.Contains(planes, "management") || slices.Contains(planes, "control") {
		if len(p.ManagementFilers) > 0 && p.ManagementFilers[0] != "leads" && !slices.Contains(p.ManagementFilers, i.Author) {
			reasons = append(reasons, fmt.Sprintf("%s may not file a %s issue", i.Author, planes[0]))
		}
	}
	return reasons
}

// section returns the list entries under a "## name" heading and whether
// the heading exists.
func section(body, name string) ([]string, bool) {
	var entries []string
	in, found := false, false
	s := bufio.NewScanner(strings.NewReader(body))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if strings.HasPrefix(line, "## ") {
			in = strings.TrimSpace(strings.TrimPrefix(line, "## ")) == name
			found = found || in
			continue
		}
		if in && (strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || numberedEntry.MatchString(line)) {
			entries = append(entries, line)
		}
	}
	return entries, found
}

func shorten(s string) string {
	if len(s) > 60 {
		return s[:57] + "..."
	}
	return s
}

// LintResult is what lint reports for one issue.
type LintResult struct {
	Project  string   `json:"project"`
	IID      int      `json:"iid"`
	Admitted bool     `json:"admitted"`
	Reasons  []string `json:"reasons,omitempty"`
	Changed  bool     `json:"changed"`
}

// Apply sets needs-triage with a comment listing the reasons on a failing
// issue and clears it on a passing one. It changes no other label and never
// the assignee (srd013 R1.2, srd041 R9.4). A dry run writes nothing.
func Apply(t gitlab.Tracker, i gitlab.Issue, reasons []string, p Policy, dryRun bool) (LintResult, error) {
	r := LintResult{Project: i.Project, IID: i.IID, Admitted: len(reasons) == 0, Reasons: reasons}
	triaged := i.HasLabel(p.TriageLabel)
	if r.Admitted {
		if triaged {
			r.Changed = true
			if !dryRun {
				_, err := t.UpdateIssue(i.Project, i.IID, gitlab.IssueUpdate{RemoveLabels: []string{p.TriageLabel}})
				return r, err
			}
		}
		return r, nil
	}
	body := lintMarker + "\nAdmission failed:\n"
	for _, reason := range reasons {
		body += "- " + reason + "\n"
	}
	notes, err := t.ListNotes(i.Project, i.IID)
	if err != nil {
		return r, err
	}
	posted := false
	for _, n := range notes {
		if strings.TrimSpace(n.Body) == strings.TrimSpace(body) {
			posted = true
		}
	}
	r.Changed = !triaged || !posted
	if dryRun {
		return r, nil
	}
	if !triaged {
		if _, err := t.UpdateIssue(i.Project, i.IID, gitlab.IssueUpdate{AddLabels: []string{p.TriageLabel}}); err != nil {
			return r, err
		}
	}
	if !posted {
		return r, t.CreateNote(i.Project, i.IID, body)
	}
	return r, nil
}
