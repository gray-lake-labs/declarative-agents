package goals

import (
	"fmt"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/objectives"
)

// CiteURL pins an objective item to the release tag (srd007 R5.1), at the
// path the objective document was read from.
func CiteURL(webBase, system, tag, docPath, item string) string {
	return fmt.Sprintf("%s/%s/-/blob/%s/%s#%s", strings.TrimSuffix(webBase, "/"), system, tag, docPath, item)
}

// Render builds the objective issue body entirely from the document and the
// confirmed plan (srd034 R3.2-R3.4). The Requirements section lists the
// steps in order, so the issue is its own instructions; the Notes section
// is empty for the owner.
func Render(webBase string, p Plan, o PlannedObjective, d *objectives.Document) (title, body string, cites []string) {
	obj := d.Requirements[o.Group]
	title = fmt.Sprintf("Objective %s: %s (release %s)", o.Key, obj.Title, p.Release)

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", strings.TrimSpace(obj.Title))
	for _, item := range o.Items {
		u := CiteURL(webBase, p.System, p.Tag, d.Path, item)
		cites = append(cites, u)
		fmt.Fprintf(&b, "Cites: %s\n", u)
	}
	fmt.Fprintf(&b, "Environment: %s\n\n## Requirements\n\n", o.Environment)
	n := 0
	for _, it := range obj.ItemsOf() {
		if contains(o.Items, it.ID) {
			n++
			fmt.Fprintf(&b, "%d. %s (%s)\n", n, it.Text, it.ID)
		}
	}
	b.WriteString("\n## Acceptance Criteria\n\n")
	for _, c := range d.CriteriaFor(o.Group) {
		if !tracesAny(c, o.Items) {
			continue
		}
		fmt.Fprintf(&b, "- %s (test: %s; environment: %s)\n", strings.TrimSpace(c.Criterion), strings.Join(c.Tests, ", "), c.Environment)
	}
	b.WriteString("\n## Notes\n\n")
	b.WriteString(marker(o.Key, p.Release) + "\n")
	return title, b.String(), cites
}

// notesHeading starts the owner's section; push never rewrites it.
const notesHeading = "\n## Notes\n"

// KeepNotes returns the rendered body with the existing issue's Notes
// section, and everything after it, carried over unchanged.
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
