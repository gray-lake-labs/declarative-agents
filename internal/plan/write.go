package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// Release is the release plan writes into the road map: its
// objective items in deliverables and its milestone and tag in
// release.publication (design.md schema.roadmap, srd031 R4.1).
type Release struct {
	Version   string   `json:"version"`
	Items     []string `json:"items"`
	Milestone string   `json:"milestone"`
	Tag       string   `json:"tag"`
}

// Returned is an item plan would not place, and why (srd031 R4.4).
type Returned struct {
	ID      string   `json:"id"`
	Missing []string `json:"missing"`
}

// Written is what tpm plan write reports.
type Written struct {
	Path     string     `json:"path"`
	Release  Release    `json:"release"`
	Action   string     `json:"action"` // added or replaced
	Returned []Returned `json:"returned,omitempty"`
}

var releaseMarker = regexp.MustCompile(`(?m)^<!-- release: ([^ |]+)( \|[^>]*)? -->$`)

// WriteRelease puts r into the road map in dir, replacing the release of
// the same version or adding it after the last one. An item the
// candidates hold back for a missing test or environment is not placed:
// it comes back in Returned, and on issue, when given, a comment names it
// and what it lacks so its owner can fix the objective document.
func WriteRelease(dir string, r Release, c Candidates, t tracker.Tracker, issueProject string, issueIID int) (Written, error) {
	w := Written{Path: RoadMap}
	held := map[string][]string{}
	for _, it := range c.WithoutEvidence {
		held[it.ID] = it.Missing
	}
	var place []string
	for _, id := range r.Items {
		switch {
		case held[id] != nil:
			w.Returned = append(w.Returned, Returned{id, held[id]})
		case !c.agreed[id]:
			return w, fmt.Errorf("%s is not a candidate objective item of an agreed objective document", id)
		default:
			place = append(place, id)
		}
	}
	if len(place) == 0 {
		return w, fmt.Errorf("release %s has no placeable objective item", r.Version)
	}
	if r.Version == "" || r.Milestone == "" || r.Tag == "" {
		return w, fmt.Errorf("a release needs a version, a milestone, and a tag")
	}
	r.Items = place
	w.Release = r
	path := filepath.Join(dir, RoadMap)
	b, err := os.ReadFile(path)
	if err != nil {
		return w, err
	}
	out, action := putRelease(string(b), r)
	w.Action = action
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return w, err
	}
	if len(w.Returned) > 0 && t != nil && issueProject != "" {
		var body strings.Builder
		body.WriteString("plan did not place these objective items; their acceptance criteria need a test and an environment (srd031 R4.4):\n\n")
		for _, x := range w.Returned {
			fmt.Fprintf(&body, "- %s: missing %s\n", x.ID, strings.Join(x.Missing, " and "))
		}
		if err := t.CreateNote(issueProject, issueIID, body.String()); err != nil {
			return w, err
		}
	}
	return w, nil
}

// putRelease replaces the release record of r's version, keeping its
// heading line, or appends a new one after the last release record.
func putRelease(doc string, r Release) (string, string) {
	var body strings.Builder
	body.WriteString("\n<!-- field: deliverables -->\n### Deliverables\n\n")
	for _, id := range r.Items {
		body.WriteString("- " + id + "\n")
	}
	fmt.Fprintf(&body, "\n<!-- field: release -->\n### Release\n\n<!-- field: publication | form: \"tree\" -->\n#### Publication\n\n- tag: %s\n- milestone: %s\n", treeValue(r.Tag), treeValue(r.Milestone))
	marks := releaseMarker.FindAllStringSubmatchIndex(doc, -1)
	for i, m := range marks {
		if doc[m[2]:m[3]] != r.Version {
			continue
		}
		head := doc[m[0]:]
		headEnd := m[0] + strings.Index(head, "\n") + 1
		headEnd += strings.Index(doc[headEnd:], "\n") + 1 // the heading line
		end := len(doc)
		if i+1 < len(marks) {
			end = marks[i+1][0]
		} else if next := nextLevel2(doc, headEnd); next >= 0 {
			end = next
		}
		tail := doc[end:]
		if tail != "" {
			tail = "\n" + strings.TrimLeft(tail, "\n")
		}
		return doc[:headEnd] + kept(doc[headEnd:end]) + body.String() + tail, "replaced"
	}
	record := fmt.Sprintf("<!-- release: %s | status: specified -->\n## %s - Release %s\n\nThe technical program manager writes here what release %s proves.\n%s%s",
		r.Version, r.Version, r.Version, r.Version, lastSRDs(doc), body.String())
	if len(marks) == 0 {
		return strings.TrimRight(doc, "\n") + "\n\n" + record, "added"
	}
	last := marks[len(marks)-1]
	end := len(doc)
	if next := nextLevel2(doc, last[1]); next >= 0 {
		end = next
	}
	before := strings.TrimRight(doc[:end], "\n") + "\n\n" + record
	if rest := strings.TrimLeft(doc[end:], "\n"); rest != "" {
		return before + "\n" + rest, "added"
	}
	return before, "added"
}

var recordField = regexp.MustCompile(`(?m)^<!-- field: ([a-z_]+)[^\n]*-->\n### `)

// kept is what a release record's body keeps when plan rewrites it: its
// prose and every field but deliverables and release, which plan owns
// (srd031 R4.1). The owned fields follow, deliverables before release
// (design.md roadmap program_release_fields).
func kept(record string) string {
	marks := recordField.FindAllStringSubmatchIndex(record, -1)
	end := len(record)
	if len(marks) > 0 {
		end = marks[0][0]
	}
	out := strings.Trim(record[:end], "\n")
	for i, m := range marks {
		stop := len(record)
		if i+1 < len(marks) {
			stop = marks[i+1][0]
		}
		if name := record[m[2]:m[3]]; name == "deliverables" || name == "release" {
			continue
		}
		if out != "" {
			out += "\n\n"
		}
		out += strings.Trim(record[m[0]:stop], "\n")
	}
	if out == "" {
		return ""
	}
	return "\n" + out + "\n"
}

// lastSRDs is the srds field of the road map's last release record, which a
// new record starts from, as the SRDs a release names change by decision,
// not by default; empty when there is none.
func lastSRDs(doc string) string {
	marks := releaseMarker.FindAllStringIndex(doc, -1)
	if len(marks) == 0 {
		return ""
	}
	record := doc[marks[len(marks)-1][0]:]
	if next := nextLevel2(record, len(record[:strings.Index(record, "\n")+1])); next >= 0 {
		record = record[:next]
	}
	for i, m := range recordField.FindAllStringSubmatchIndex(record, -1) {
		if record[m[2]:m[3]] != "srds" {
			continue
		}
		end := len(record)
		if all := recordField.FindAllStringIndex(record, -1); i+1 < len(all) {
			end = all[i+1][0]
		}
		return "\n" + strings.Trim(record[m[0]:end], "\n") + "\n"
	}
	return ""
}

// treeValue writes a tree item's value so it reads back as the same text:
// a value YAML would read as a number or a date, such as 01.1, is quoted,
// as the release's printer quotes it.
func treeValue(v string) string {
	b, err := yaml.Marshal(v)
	if err != nil {
		return v
	}
	return strings.TrimSuffix(string(b), "\n")
}

// nextLevel2 is the offset of the next marker above a level 2 heading at
// or after from, or -1: where the record that started before from ends.
func nextLevel2(doc string, from int) int {
	re := regexp.MustCompile(`(?m)^<!-- [a-z_]+: [^\n]*-->\n## `)
	if loc := re.FindStringIndex(doc[from:]); loc != nil {
		return from + loc[0]
	}
	return -1
}
