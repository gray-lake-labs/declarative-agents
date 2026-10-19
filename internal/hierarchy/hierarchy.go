// Package hierarchy reads the planning documents of the agile hierarchy,
// theme, initiative, epic, feature, and story, and checks them as the corpus
// gate does (design.md hierarchy_links, srd045). Each is an SRD in the
// pinned parent release's Markdown format whose context items carry its
// owner, its parent, its links, and, for a story, its statement, points,
// rank, and citations. An objective document is read as an initiative.
package hierarchy

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/corpus"
)

// Kind is a level of the hierarchy.
type Kind string

const (
	Theme      Kind = "theme"
	Initiative Kind = "initiative"
	Epic       Kind = "epic"
	Feature    Kind = "feature"
	Story      Kind = "story"
)

// Level is one kind with the directory and id prefix its documents use.
type Level struct {
	Kind   Kind
	Dir    string
	Prefix string
}

// Levels lists the hierarchy top-down (design.md schema).
var Levels = []Level{
	{Theme, "docs/specs/themes", "th"},
	{Initiative, "docs/specs/initiatives", "in"},
	{Epic, "docs/specs/epics", "ep"},
	{Feature, "docs/specs/features", "ft"},
	{Story, "docs/specs/stories", "st"},
}

// ObjectiveDir holds objective documents, read as initiatives (srd034 R1.1).
const ObjectiveDir = "docs/specs/objectives"

// parentKinds names the kinds a document's parent may have (srd007 R7.2).
var parentKinds = map[Kind][]Kind{
	Initiative: {Theme},
	Epic:       {Initiative},
	Feature:    {Epic},
	Story:      {Feature, Epic},
}

// LinkKeys are the horizontal links a document carries as context items.
var LinkKeys = []string{"blocked_by", "depends_on", "relates_to", "duplicates"}

// Doc is one hierarchy document.
type Doc struct {
	Path     string
	ID       string
	Kind     Kind
	Title    string
	Status   string
	Owner    string
	Horizon  string
	Parent   string
	Links    map[string][]string
	Problem  string
	Goals    []string
	NonGoals []string
	// Groups maps each requirement group to its item ids, in document order.
	Groups   map[string][]string
	Due      map[string]string
	Criteria []Criterion
	// The story's statement and sizing (design.md story).
	AsA    string
	IWant  string
	SoThat string
	Points string
	Rank   string
	Cites  []string
}

// Criterion is one acceptance criterion with the evidence it names (D8).
type Criterion struct {
	ID          string
	Traces      []string
	Tests       []string
	Environment string
}

// Statuses are the values every hierarchy document may carry.
var Statuses = []string{"to_be_agreed", "agreed", "retired"}

// PointsValue is a story's points as a number, or 0 when they are missing
// or not a number.
func (d *Doc) PointsValue() int {
	n, err := strconv.Atoi(strings.TrimSpace(d.Points))
	if err != nil {
		return 0
	}
	return n
}

// RankValue is a story's rank as a number; a story without a rank sorts
// after every ranked one.
func (d *Doc) RankValue() int {
	n, err := strconv.Atoi(strings.TrimSpace(d.Rank))
	if err != nil {
		return int(^uint(0) >> 1)
	}
	return n
}

// KindOf returns the kind a document id's prefix names; objNNN is an
// initiative.
func KindOf(id string) (Kind, bool) {
	if strings.HasPrefix(id, "obj") {
		return Initiative, true
	}
	for _, l := range Levels {
		if strings.HasPrefix(id, l.Prefix) && len(id) > len(l.Prefix) && id[len(l.Prefix)] >= '0' && id[len(l.Prefix)] <= '9' {
			return l.Kind, true
		}
	}
	return "", false
}

// Load reads every hierarchy document of the repository at root, in level
// order and then by name. Absent directories hold no documents.
func Load(root string) ([]*Doc, error) {
	var docs []*Doc
	for _, l := range Levels {
		dirs := [][2]string{{l.Dir, l.Prefix}}
		if l.Kind == Initiative {
			dirs = append(dirs, [2]string{ObjectiveDir, "obj"})
		}
		for _, d := range dirs {
			paths, err := filepath.Glob(filepath.Join(root, d[0], d[1]+"[0-9]*.md"))
			if err != nil {
				return nil, err
			}
			sort.Strings(paths)
			for _, p := range paths {
				data, err := os.ReadFile(p)
				if err != nil {
					return nil, err
				}
				doc, err := Parse(p, data)
				if err != nil {
					return nil, err
				}
				docs = append(docs, doc)
			}
		}
	}
	return docs, nil
}

// Parse reads one hierarchy document; its kind comes from its file name.
func Parse(path string, data []byte) (*Doc, error) {
	f, err := corpus.Decode(path, data)
	if err != nil {
		return nil, err
	}
	md := f.Markdown
	d := &Doc{Path: path, ID: md.ID(), Links: map[string][]string{}, Groups: map[string][]string{}, Due: map[string]string{}}
	d.Kind, _ = KindOf(strings.TrimSuffix(filepath.Base(path), ".md"))
	d.Title, _ = md.Front["title"].(string)
	d.Status, _ = md.Front["status"].(string)
	context := md.Field("context").Tree()
	d.Owner = corpus.TreeString(context, "owner")
	d.Horizon = corpus.TreeString(context, "horizon")
	d.Parent = corpus.TreeString(context, "parent")
	for _, k := range LinkKeys {
		if v := corpus.TreeList(context, k); len(v) > 0 {
			d.Links[k] = v
		} else if s := corpus.TreeString(context, k); s != "" {
			d.Links[k] = []string{s}
		}
	}
	d.AsA = corpus.TreeString(context, "as_a")
	d.IWant = corpus.TreeString(context, "i_want")
	d.SoThat = corpus.TreeString(context, "so_that")
	d.Points = corpus.TreeString(context, "points")
	d.Rank = corpus.TreeString(context, "rank")
	d.Cites = corpus.TreeList(context, "cites")
	d.Problem = md.Field("problem").Text()
	d.Goals = md.Field("goals").List()
	d.NonGoals = md.Field("non_goals").List()
	for _, group := range md.Sections {
		if group.Kind != "req" {
			continue
		}
		var items []string
		for _, item := range group.Children {
			if item.Kind == "req" && strings.TrimSpace(item.Text()) != "" {
				items = append(items, item.Key)
			}
		}
		d.Groups[group.Key] = items
		if due := corpus.TreeString(group.Field("applicability").Tree(), "due"); due != "" {
			d.Due[group.Key] = due
		}
	}
	criteria := md.Field("acceptance_criteria")
	if criteria == nil {
		return d, nil
	}
	for _, ac := range criteria.Children {
		if ac.Kind != "ac" {
			continue
		}
		d.Criteria = append(d.Criteria, Criterion{
			ID:          ac.Key,
			Traces:      ac.Field("traces").List(),
			Tests:       ac.Field("tests").List(),
			Environment: strings.TrimSpace(ac.Field("applicability").Text()),
		})
	}
	return d, nil
}
