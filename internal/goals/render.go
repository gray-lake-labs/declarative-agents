package goals

import (
	"fmt"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/objectives"
	"github.com/gray-lake-labs/scrum-master/internal/sprintgoal"
)

// CiteURL pins an initiative item to the release tag (srd007 R5.1), at the
// path the initiative was read from.
func CiteURL(webBase, system, tag, docPath, item string) string {
	return fmt.Sprintf("%s/%s/-/blob/%s/%s#%s", strings.TrimSuffix(webBase, "/"), system, tag, docPath, item)
}

// Render builds a team's Sprint Goal for the release entirely from the
// initiatives and the confirmed plan (srd034 R3.2-R3.5): it serves every
// outcome the plan confirmed for the team's repository. The Requirements
// section lists the steps in order, so the issue is its own instructions;
// the Notes and Sprint Backlog sections are empty for the team.
func Render(webBase string, p Plan, team []PlannedObjective, docs map[string]*objectives.Document) (title, body string, cites []string) {
	var titles, serves []string
	for _, o := range team {
		group := docs[o.Document].Requirements[o.Group]
		title := strings.TrimSpace(group.Title)
		// A team serving part of a split group names its items, so the
		// teams' Sprint Goals for one outcome read apart (srd034 R3.2).
		if len(o.Items) < len(group.ItemsOf()) {
			title += " (" + strings.Join(o.Items, ", ") + ")"
		}
		titles = append(titles, title)
		serves = append(serves, o.Key)
	}
	goal := strings.Join(titles, "; ")
	title = fmt.Sprintf("Sprint Goal: %s (release %s)", goal, p.Release)

	var b strings.Builder
	fmt.Fprintf(&b, "Sprint Goal for release %s: %s\n\n", p.Release, goal)
	for _, o := range team {
		d := docs[o.Document]
		for _, item := range o.Items {
			u := CiteURL(webBase, p.System, p.Tag, d.Path, item)
			cites = append(cites, u)
			fmt.Fprintf(&b, "Cites: %s\n", u)
		}
	}
	fmt.Fprintf(&b, "Environment: %s\n\n## Requirements\n\n", team[0].Environment)
	n := 0
	for _, o := range team {
		for _, it := range docs[o.Document].Requirements[o.Group].ItemsOf() {
			if contains(o.Items, it.ID) {
				n++
				fmt.Fprintf(&b, "%d. %s (%s)\n", n, it.Text, it.ID)
			}
		}
	}
	b.WriteString("\n## Acceptance Criteria\n\n")
	for _, o := range team {
		d := docs[o.Document]
		for _, c := range d.CriteriaFor(o.Group) {
			if !tracesAny(c, o.Items) {
				continue
			}
			// A criterion that also traces other teams' items says which
			// part is this Sprint Goal's, naming only its own items: the
			// other teams' work is theirs to name (srd022 R3.2).
			part := ""
			var mine []string
			for _, t := range c.Traces {
				if contains(o.Items, t) {
					mine = append(mine, t)
				}
			}
			if len(mine) < len(c.Traces) {
				part = " [this Sprint Goal: " + strings.Join(mine, ", ") + "]"
			}
			fmt.Fprintf(&b, "- %s%s (test: %s; environment: %s)\n", strings.TrimSpace(c.Criterion), part, strings.Join(c.Tests, ", "), c.Environment)
		}
	}
	b.WriteString("\n## Notes\n\n## Sprint Backlog\n\n")
	b.WriteString(sprintgoal.Marker(p.Release, serves) + "\n")
	return title, b.String(), cites
}

// notesHeading starts the team's sections, Notes and the Sprint Backlog
// Sprint Planning fills; push never rewrites them.
const notesHeading = "\n## Notes\n"

// KeepNotes returns the rendered body with the existing issue's Notes
// section, and everything after it, the Sprint Backlog included, carried
// over unchanged.
func KeepNotes(rendered, existing string) string {
	i := strings.Index(existing, notesHeading)
	j := strings.Index(rendered, notesHeading)
	if i < 0 || j < 0 {
		return rendered
	}
	return rendered[:j] + existing[i:]
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func tracesAny(c objectives.Criterion, items []string) bool {
	for _, t := range c.Traces {
		if contains(items, t) {
			return true
		}
	}
	return false
}
