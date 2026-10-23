// Package backlog is tpm backlog: the planning hierarchy of one repository,
// read and checked (srd045 R8). It reads only the directory it is given and
// writes nothing, so a skill can call it for facts (design.md D10).
package backlog

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/cli"
	"github.com/gray-lake-labs/scrum-master/internal/corpus"
	"github.com/gray-lake-labs/scrum-master/internal/hierarchy"
)

// Entry is one hierarchy document as tpm backlog list prints it.
type Entry struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Parent string `json:"parent,omitempty"`
	Path   string `json:"path"`
	Points string `json:"points,omitempty"`
	Rank   string `json:"rank,omitempty"`
}

// Report is what tpm backlog check prints: how many documents it read and
// every finding the gate reports on them. SpecSkipped says why the parent
// spec tool's shape check did not run, when it did not.
type Report struct {
	Documents   int                 `json:"documents"`
	Findings    []hierarchy.Finding `json:"findings"`
	SpecSkipped string              `json:"spec_check_skipped,omitempty"`
}

// List reads the hierarchy documents under dir, keeping one kind when kind
// is not empty; paths are relative to dir.
func List(dir, kind string) ([]Entry, error) {
	docs, err := hierarchy.Load(dir)
	if err != nil {
		return nil, err
	}
	entries := []Entry{}
	for _, d := range docs {
		if kind != "" && string(d.Kind) != kind {
			continue
		}
		e := Entry{ID: d.ID, Kind: string(d.Kind), Title: d.Title, Status: d.Status, Parent: d.Parent, Path: relative(dir, d.Path)}
		if d.Kind == hierarchy.Story {
			e.Points, e.Rank = d.Points, d.Rank
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// Check runs the hierarchy gate (srd045 R6.1) on the documents under dir,
// through hierarchy.Validate, the function the mage audit gate calls; it
// also reads the SRDs, refusing a requirement no criterion traces, and runs
// the parent spec tool's shape check, so the gate refuses what tpm doctor
// would (srd043 R4.4).
func Check(dir string) (Report, error) {
	docs, err := hierarchy.Load(dir)
	if err != nil {
		return Report{}, err
	}
	findings := hierarchy.Validate(docs)
	srds, untraced, err := checkSRDs(dir)
	if err != nil {
		return Report{}, err
	}
	statuses, err := releaseStatuses(dir)
	if err != nil {
		return Report{}, err
	}
	untraced = append(untraced, statuses...)
	shapes, skipped := checkShapes(dir)
	findings = append(append(append(findings, untraced...), shapes...), untracked(dir)...)
	for i := range findings {
		findings[i].File = relative(dir, findings[i].File)
	}
	if findings == nil {
		findings = []hierarchy.Finding{}
	}
	return Report{Documents: len(docs) + srds, Findings: findings, SpecSkipped: skipped}, nil
}

func relative(dir, path string) string {
	if rel, err := filepath.Rel(dir, path); err == nil {
		return rel
	}
	return path
}

func kinds() []string {
	var ks []string
	for _, l := range hierarchy.Levels {
		ks = append(ks, string(l.Kind))
	}
	return ks
}

func init() {
	var la struct{ dir, kind string }
	cli.Default.Register(&cli.Command{
		Group: "backlog", Name: "list", Summary: "the repository's hierarchy documents: id, kind, title, status, parent, and path",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&la.dir, "dir", ".", "the repository's checkout")
			fs.StringVar(&la.kind, "kind", "", "keep one kind: "+strings.Join(kinds(), ", "))
		},
		Run: func(ctx *cli.Context) error {
			if la.kind != "" && !strings.Contains(" "+strings.Join(kinds(), " ")+" ", " "+la.kind+" ") {
				return fmt.Errorf("%w: --kind is one of %s", cli.ErrUsage, strings.Join(kinds(), ", "))
			}
			entries, err := List(la.dir, la.kind)
			if err != nil {
				return err
			}
			var h strings.Builder
			for _, e := range entries {
				fmt.Fprintf(&h, "%-12s %-40s %-12s parent=%s  %s\n", e.Kind, e.ID, e.Status, orNone(e.Parent), e.Title)
			}
			if len(entries) == 0 {
				h.WriteString("no hierarchy documents\n")
			}
			return ctx.Emit(entries, h.String())
		},
	})

	var ca struct{ dir string }
	cli.Default.Register(&cli.Command{
		Group: "backlog", Name: "check", Summary: "the hierarchy gate on the repository's documents; exits non-zero on any finding",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&ca.dir, "dir", ".", "the repository's checkout")
		},
		Run: func(ctx *cli.Context) error {
			r, err := Check(ca.dir)
			if err != nil {
				return err
			}
			var h strings.Builder
			for _, f := range r.Findings {
				fmt.Fprintln(&h, f.String())
			}
			if len(r.Findings) == 0 {
				fmt.Fprintf(&h, "hierarchy gate: pass, %d documents\n", r.Documents)
			}
			if r.SpecSkipped != "" {
				fmt.Fprintf(&h, "spec shape check skipped: %s\n", r.SpecSkipped)
			}
			if err := ctx.Emit(r, h.String()); err != nil {
				return err
			}
			if len(r.Findings) > 0 {
				return fmt.Errorf("hierarchy gate: %d findings in %d documents", len(r.Findings), r.Documents)
			}
			return nil
		},
	})
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

// ReleaseStatuses are the values a road-map release record's status takes:
// planned, built and proven, and tagged and handed off (srd031 R4.1,
// design.md roadmap).
var ReleaseStatuses = []string{"specified", "implemented", "released"}

// releaseStatuses reports each road-map release record whose status is not
// one of ReleaseStatuses (srd045 R8.2). A repository without a road map has
// none.
func releaseStatuses(dir string) ([]hierarchy.Finding, error) {
	path := filepath.Join(dir, "docs", "road-map.md")
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d, err := corpus.ParseMarkdown(b)
	if err != nil {
		return nil, nil // the shape check reports a road map that does not parse
	}
	var out []hierarchy.Finding
	for _, r := range d.Records("release") {
		if status := r.Attrs["status"]; !slices.Contains(ReleaseStatuses, status) {
			out = append(out, hierarchy.Finding{File: path, Field: "release " + r.Key + " status",
				Message: fmt.Sprintf("%q is not a release status; use %s", status, strings.Join(ReleaseStatuses, ", "))})
		}
	}
	return out, nil
}
