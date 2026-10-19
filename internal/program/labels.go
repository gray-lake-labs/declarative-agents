package program

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/corpus"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// DefaultLabels are the label values the labels config format lists, used
// when the program repository has no control/labels.md. Component labels
// are not here: they come from the subsystem SRDs (srd001 R21.1).
func DefaultLabels() []tracker.Label {
	return []tracker.Label{
		{Name: "plane::management", Color: "#5843ad", Description: "Changes intent - road-map release, system or subsystem SRD, objective document, sprint"},
		{Name: "plane::control", Color: "#8e44ad", Description: "Changes a rule - policy files, label schema, state machine"},
		{Name: "plane::data", Color: "#1f75cb", Description: "Work"},
		{Name: "type::tracking", Color: "#6699cc", Description: "Task list of member issues; never popped"},
		{Name: "type::sprint", Color: "#009966", Description: "The sprint's exit checklist"},
		{Name: "type::recurring", Color: "#00b140", Description: "Subscription to one artifact"},
		{Name: "kind::build", Color: "#428bca", Description: "Builds or changes software"},
		{Name: "kind::experiment", Color: "#5cb85c", Description: "Produces a measurement"},
		{Name: "kind::research", Color: "#69d100", Description: "Produces a written result"},
		{Name: "kind::spec", Color: "#8e7cc3", Description: "Changes requirements or design documents under docs/"},
		{Name: "kind::review", Color: "#f0ad4e", Description: "Responds to another person's artifact"},
		{Name: "kind::consume", Color: "#ff8c00", Description: "Adapts to a released artifact"},
		{Name: "kind::handoff", Color: "#ad8d43", Description: "Decides what an arriving artifact means"},
		{Name: "kind::release", Color: "#34495e", Description: "Publishes a version"},
		{Name: "kind::defect", Color: "#d9534f", Description: "An artifact is wrong"},
		{Name: "kind::request", Color: "#d10069", Description: "An artifact lacks something"},
		{Name: "kind::question", Color: "#a8d695", Description: "Needs an answer, not a change"},
		{Name: "needs-triage", Color: "#cc338b", Description: "Failed admission; not ready"},
		{Name: "status::in-progress", Color: "#428bca", Description: "The assignee started it"},
		{Name: "waiting", Color: "#e6e6fa", Description: "A link is unsatisfied; set by track-goals"},
		{Name: "blocked", Color: "#d9534f", Description: "The assignee states a blocker"},
		{Name: "needs-run", Color: "#ff8c00", Description: "A breaking release reached this subscription"},
	}
}

type labelsFileValues struct {
	Axes map[string]struct {
		Prefix string `yaml:"prefix"`
		Values []struct {
			Name        string `yaml:"name"`
			Color       string `yaml:"color"`
			Description string `yaml:"description"`
		} `yaml:"values"`
	} `yaml:"axes"`
}

// LoadLabels reads control/labels.md, a config document (design D11), from
// the program repository at ref, or returns DefaultLabels when it is absent.
func LoadLabels(t tracker.Tracker, project, ref string) ([]tracker.Label, error) {
	if project == "" {
		return DefaultLabels(), nil
	}
	b, err := t.GetFile(project, LabelsFile, ref)
	if errors.Is(err, tracker.ErrNotFound) {
		return DefaultLabels(), nil
	}
	if err != nil {
		return nil, err
	}
	var f labelsFileValues
	if err := corpus.DecodeConfig(b, &f); err != nil {
		return nil, fmt.Errorf("%s %s: %w", project, LabelsFile, err)
	}
	var out []tracker.Label
	for _, axis := range []string{"plane", "type", "kind", "state"} {
		a, ok := f.Axes[axis]
		if !ok {
			continue
		}
		for _, v := range a.Values {
			out = append(out, tracker.Label{Name: a.Prefix + v.Name, Color: v.Color, Description: v.Description})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s %s lists no label values", project, LabelsFile)
	}
	return out, nil
}

// LabelResult is what labels reports for one label.
type LabelResult struct {
	Name   string `json:"name"`
	Action string `json:"action"` // created, updated, unchanged, differs
}

// SyncLabels creates every wanted label the group lacks (srd013 R6). A label
// that exists with another color or description is reported as differs and
// left alone unless update is set, so a group whose labels another tool
// manages is not fought over. No label is ever deleted or renamed, and a
// dry run writes nothing.
func SyncLabels(t tracker.Tracker, group string, want []tracker.Label, update, dryRun bool) ([]LabelResult, error) {
	have, err := t.GroupLabels(group)
	if err != nil {
		return nil, err
	}
	existing := map[string]tracker.Label{}
	for _, l := range have {
		existing[l.Name] = l
	}
	var out []LabelResult
	for _, w := range want {
		e, ok := existing[w.Name]
		switch {
		case !ok:
			if !dryRun {
				if err := t.CreateGroupLabel(group, w); err != nil {
					return out, fmt.Errorf("create %s: %w", w.Name, err)
				}
			}
			out = append(out, LabelResult{w.Name, "created"})
		case strings.EqualFold(e.Color, w.Color) && e.Description == w.Description:
			out = append(out, LabelResult{w.Name, "unchanged"})
		case update:
			if !dryRun {
				if err := t.UpdateGroupLabel(group, w); err != nil {
					return out, fmt.Errorf("update %s: %w", w.Name, err)
				}
			}
			out = append(out, LabelResult{w.Name, "updated"})
		default:
			out = append(out, LabelResult{w.Name, "differs"})
		}
	}
	return out, nil
}

// TypeMigration is one issue --migrate relabelled, or would on a dry run.
type TypeMigration struct {
	Project string `json:"project"`
	IID     int    `json:"iid"`
	From    string `json:"from"`
	To      string `json:"to"`
}

// MigrateTypes relabels every issue, open or closed, in the projects that
// carries a type value's earlier name, so it carries the current one
// instead (srd013 R6.3). It is idempotent, a dry run writes nothing, and no
// group label is deleted or renamed.
func MigrateTypes(t tracker.Tracker, projects []string, dryRun bool) ([]TypeMigration, error) {
	var out []TypeMigration
	for _, r := range typeRenames {
		from, to := "type::"+r[0], "type::"+r[1]
		for _, p := range projects {
			issues, err := t.ListIssues(p, tracker.IssueQuery{State: "all", Labels: []string{from}})
			if err != nil {
				return out, err
			}
			for _, i := range issues {
				if !dryRun {
					if _, err := t.UpdateIssue(p, i.IID, tracker.IssueUpdate{AddLabels: []string{to}, RemoveLabels: []string{from}}); err != nil {
						return out, fmt.Errorf("relabel %s#%d: %w", p, i.IID, err)
					}
				}
				out = append(out, TypeMigration{Project: p, IID: i.IID, From: from, To: to})
			}
		}
	}
	return out, nil
}

// StaleTypes lists the earlier type names control/labels.md in the program
// repository at ref still lists. tpm never edits a control file; a
// plane::control issue changes it (srd013 R6.3).
func StaleTypes(t tracker.Tracker, project, ref string) ([]string, error) {
	b, err := t.GetFile(project, LabelsFile, ref)
	if errors.Is(err, tracker.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f labelsFile
	if err := corpus.DecodeConfig(b, &f); err != nil {
		return nil, fmt.Errorf("%s %s: %w", project, LabelsFile, err)
	}
	var stale []string
	for _, v := range f.Axes["type"].Values {
		if CanonicalType(v.Name) != v.Name {
			stale = append(stale, v.Name)
		}
	}
	return stale, nil
}

// MigrateResult is what labels --migrate reports.
type MigrateResult struct {
	Labels     []LabelResult   `json:"labels"`
	Relabelled []TypeMigration `json:"relabelled"`
	// StaleTypes are earlier type names the program's control/labels.md
	// still lists.
	StaleTypes []string `json:"stale_types,omitempty"`
}

// migrate relabels issues in the system repository and every repository its
// subsystem SRDs declare, and reports what its control/labels.md still
// lists (srd013 R6.3).
func migrate(t tracker.Tracker, system, ref string, dryRun bool) (MigrateResult, error) {
	subs, err := LoadSubsystems(t, system, ref)
	if err != nil && !errors.Is(err, tracker.ErrNotFound) {
		return MigrateResult{}, err
	}
	projects := []string{ProjectPath(system)}
	for _, r := range subs.Repositories() {
		if !slices.Contains(projects, r) {
			projects = append(projects, r)
		}
	}
	relabelled, err := MigrateTypes(t, projects, dryRun)
	if err != nil {
		return MigrateResult{Relabelled: relabelled}, err
	}
	stale, err := StaleTypes(t, system, ref)
	return MigrateResult{Relabelled: relabelled, StaleTypes: stale}, err
}
