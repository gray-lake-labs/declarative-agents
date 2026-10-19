// Package events writes and reads comment events: comments whose body holds
// one <!-- tpm program event=NAME key=value ... --> marker (srd006 R1).
// Events inform; posting one never changes an issue's state (R7.3).
package events

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
)

// KV is one key and value of a marker, in the order written.
type KV struct{ Key, Value string }

// Event is a parsed marker. A key may repeat (requirement on planned).
type Event struct {
	Name string
	Keys []KV
}

// Values returns every value of key, in order.
func (e Event) Values(key string) []string {
	var out []string
	for _, kv := range e.Keys {
		if kv.Key == key {
			out = append(out, kv.Value)
		}
	}
	return out
}

var markerPattern = regexp.MustCompile(`<!--\s*tpm program (event=\S+(?:\s+[a-z_]+=\S+)*)\s*-->`)

// Marker renders the event's marker. Values must be free of spaces (R1.1).
func (e Event) Marker() (string, error) {
	if e.Name == "" || strings.ContainsAny(e.Name, " \t\n") {
		return "", fmt.Errorf("event name %q is empty or has spaces", e.Name)
	}
	parts := []string{"event=" + e.Name}
	for _, kv := range e.Keys {
		if kv.Value == "" || strings.ContainsAny(kv.Value, " \t\n") {
			return "", fmt.Errorf("event %s: value of %s is empty or has spaces: %q", e.Name, kv.Key, kv.Value)
		}
		parts = append(parts, kv.Key+"="+kv.Value)
	}
	return "<!-- tpm program " + strings.Join(parts, " ") + " -->", nil
}

// Parse returns the marker in a comment body, if it holds exactly one.
func Parse(body string) (Event, bool) {
	m := markerPattern.FindAllStringSubmatch(body, -1)
	if len(m) != 1 {
		return Event{}, false
	}
	var e Event
	for i, field := range strings.Fields(m[0][1]) {
		k, v, _ := strings.Cut(field, "=")
		if i == 0 {
			e.Name = v
			continue
		}
		e.Keys = append(e.Keys, KV{k, v})
	}
	return e, true
}

// Same reports whether two events have the same name and the same values
// for each required key, the duplicate rule of srd006 R7.2.
func Same(a, b Event, required []string) bool {
	if a.Name != b.Name {
		return false
	}
	for _, k := range required {
		av, bv := slices.Clone(a.Values(k)), slices.Clone(b.Values(k))
		slices.Sort(av)
		slices.Sort(bv)
		if !slices.Equal(av, bv) {
			return false
		}
	}
	return true
}

// Has reports whether any note already carries the event.
func Has(notes []gitlab.Note, e Event, required []string) bool {
	for _, n := range notes {
		if got, ok := Parse(n.Body); ok && Same(got, e, required) {
			return true
		}
	}
	return false
}

// Comment is the full comment: one or two sentences for a person, then the
// marker.
func Comment(e Event, text string) (string, error) {
	marker, err := e.Marker()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text) + "\n\n" + marker, nil
}

// Post posts the event on an issue unless an equal one is already there.
// It returns whether it posted; a dry run reports what it would do and
// writes nothing.
func Post(t gitlab.Tracker, project string, iid int, e Event, required []string, text string, dryRun bool) (bool, error) {
	body, err := Comment(e, text)
	if err != nil {
		return false, err
	}
	notes, err := t.ListNotes(project, iid)
	if err != nil {
		return false, err
	}
	if Has(notes, e, required) {
		return false, nil
	}
	if dryRun {
		return true, nil
	}
	return true, t.CreateNote(project, iid, body)
}

// EvidenceRequired are the evidence event's required keys (srd006 R9.2),
// with the run that posted it, so a later run is never taken for a
// duplicate of an earlier one (R9.3).
var EvidenceRequired = []string{"requirement", "test", "environment", "version", "result", "run"}

// Evidence is the event a release gate posts for one check (srd015 R1.1):
// what it proves, the check, where it ran, the candidate tag, the result
// (pass, fail, or pending), and extra keys naming the artifact, the
// revision, and the run.
func Evidence(requirement, test, environment, version, result string, extra ...KV) Event {
	e := Event{Name: "evidence", Keys: []KV{{"requirement", requirement}, {"test", test}, {"environment", environment},
		{"version", version}, {"result", result}}}
	e.Keys = append(e.Keys, extra...)
	return e
}

// PlannedRequired are the planned event's required keys (srd006 R8.2).
var PlannedRequired = []string{"release", "requirement", "environment"}

// Planned is the event tasks and plan post on an objective
// issue: the release it counts toward, each requirement it satisfies as a
// pinned URL, and the environment where its evidence counts (srd006 R8).
func Planned(release string, requirements []string, environment string) (Event, string) {
	e := Event{Name: "planned", Keys: []KV{{"release", release}}}
	for _, r := range requirements {
		e.Keys = append(e.Keys, KV{"requirement", r})
	}
	e.Keys = append(e.Keys, KV{"environment", environment})
	text := fmt.Sprintf("Planned for release %s; evidence counts in %s for %d requirement(s).", release, environment, len(requirements))
	return e, text
}
