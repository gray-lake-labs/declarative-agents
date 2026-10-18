package beads

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// MilestonesFile is where the tracker repository holds milestones, beside
// the issues; bd does not read it.
const MilestonesFile = ".beads/milestones.jsonl"

// Milestone is a group's or project's milestone (srd042 R2.2): Scope is the
// group or project path it belongs to, and an issue names it by title in its
// Release field.
type Milestone struct {
	Scope       string `json:"scope"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	StartDate   string `json:"start_date,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

func (m Milestone) key() string { return m.Scope + "\x00" + m.Title }

// sameAs reports whether m carries o's content, ignoring when it changed.
func (m Milestone) sameAs(o Milestone) bool {
	m.UpdatedAt, o.UpdatedAt = "", ""
	return m == o
}

func parseMilestones(b []byte) ([]Milestone, error) {
	var ms []Milestone
	for n, line := range bytes.Split(b, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var m Milestone
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", MilestonesFile, n+1, err)
		}
		ms = append(ms, m)
	}
	return ms, nil
}

func writeMilestones(path string, ms []Milestone) error {
	sorted := slices.Clone(ms)
	slices.SortFunc(sorted, func(a, b Milestone) int { return strings.Compare(a.key(), b.key()) })
	var buf bytes.Buffer
	for _, m := range sorted {
		line, err := json.Marshal(m)
		if err != nil {
			return err
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// mergeMilestones merges two clones' milestones over their merge base, a
// milestone at a time: the side that changed it wins, and when both did,
// the later writer.
func mergeMilestones(base, ours, theirs []Milestone) []Milestone {
	index := func(ms []Milestone) map[string]Milestone {
		m := make(map[string]Milestone, len(ms))
		for _, x := range ms {
			m[x.key()] = x
		}
		return m
	}
	b, o, t := index(base), index(ours), index(theirs)
	out := slices.Clone(theirs)
	for k, om := range o {
		tm, inTheirs := t[k]
		bm, inBase := b[k]
		switch {
		case !inTheirs:
			out = append(out, om)
		case inBase && om.sameAs(bm):
		case inBase && tm.sameAs(bm), om.UpdatedAt > tm.UpdatedAt:
			out[slices.IndexFunc(out, func(x Milestone) bool { return x.key() == k })] = om
		}
	}
	return out
}

// ancestors returns project and every group above it, nearest first.
func ancestors(project string) []string {
	var out []string
	for p := project; p != ""; {
		out = append(out, p)
		i := strings.LastIndex(p, "/")
		if i < 0 {
			break
		}
		p = p[:i]
	}
	return out
}

// Milestones pulls the tracker repository and returns every milestone.
func (s *Store) Milestones() ([]Milestone, Freshness, error) {
	fr, err := s.Sync()
	if err != nil {
		return nil, fr, err
	}
	ms, err := s.readMilestones()
	return ms, fr, err
}

// EnsureMilestone makes the milestone titled m.Title in m.Scope carry m's
// description and dates, creating it when the scope has none, and reports
// created, updated, or unchanged as GitLab's EnsureMilestone does.
func (s *Store) EnsureMilestone(m Milestone) (string, error) {
	result := "unchanged"
	_, _, _, err := s.writeFiles("milestone "+m.Scope+" "+m.Title, func() (bool, error) {
		ms, err := s.readMilestones()
		if err != nil {
			return false, err
		}
		m.UpdatedAt = s.stamp()
		n := slices.IndexFunc(ms, func(x Milestone) bool { return x.key() == m.key() })
		switch {
		case n < 0:
			ms, result = append(ms, m), "created"
		case ms[n].sameAs(m):
			return false, nil
		default:
			ms[n], result = m, "updated"
		}
		return true, writeMilestones(filepath.Join(s.Dir, MilestonesFile), ms)
	})
	return result, err
}

func (s *Store) readMilestones() ([]Milestone, error) {
	b, err := os.ReadFile(filepath.Join(s.Dir, MilestonesFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parseMilestones(b)
}

func (s *Store) showMilestones(rev string) ([]Milestone, error) {
	out, err := s.git("show", rev+":"+MilestonesFile)
	if err != nil {
		return nil, nil
	}
	return parseMilestones([]byte(out))
}
