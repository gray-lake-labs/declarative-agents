package plan

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/events"
	"github.com/gray-lake-labs/scrum-master/internal/goals"
	"github.com/gray-lake-labs/scrum-master/internal/objectives"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// SprintEndLabels are a sprint-end issue's labels besides its component: a
// recurring data issue whose work is a release (srd038 R1.1).
var SprintEndLabels = []string{"plane::data", "type::recurring", "kind::release"}

// PushInput is one tpm plan push: the release the road map in Dir holds,
// dated by End, pushed to the system repository and the repositories its
// subsystem SRDs declare.
type PushInput struct {
	Dir     string
	System  string
	Group   string // the program group the milestone lives on
	Version string
	End     string // the milestone's end date, YYYY-MM-DD
	WebBase string
	Policy  program.Policy
	Ref     string
	DryRun  bool
}

// PushResult is what tpm plan push did.
type PushResult struct {
	Release    Release          `json:"release"`
	Milestone  string           `json:"milestone"` // created, updated, unchanged
	Sprint     IssueAction      `json:"sprint_issue"`
	SprintEnds []IssueAction    `json:"sprint_end_issues"`
	Planned    []PlannedOnIssue `json:"planned_events"`
	Skipped    []string         `json:"skipped,omitempty"`
}

// IssueAction is one issue push filed, updated, or found unchanged.
type IssueAction struct {
	Project  string   `json:"project"`
	IID      int      `json:"iid,omitempty"`
	Action   string   `json:"action"` // filed, updated, unchanged
	Admitted bool     `json:"admitted"`
	Reasons  []string `json:"reasons,omitempty"`
}

// PlannedOnIssue is a planned event push posted, or found already posted.
type PlannedOnIssue struct {
	Issue  string `json:"issue"`
	Posted bool   `json:"posted"`
}

var weekRange = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// Push dates the release's milestone, sets the sprint issue's checklist,
// posts a planned event on each open objective issue the release moves,
// and files one sprint-end issue per releasing repository, each found by
// its marker so a second push updates and creates nothing (srd031 R4.2,
// R4.3, R4.6; srd038 R1.1; srd001 R7).
func Push(t tracker.Tracker, in PushInput) (PushResult, error) {
	var res PushResult
	if !weekRange.MatchString(in.End) {
		return res, fmt.Errorf("--end must be YYYY-MM-DD, not %q", in.End)
	}
	end, _ := time.Parse(time.DateOnly, in.End)
	start := end.AddDate(0, 0, 1-periodDays)
	releases, err := readRoadMap(in.Dir)
	if err != nil {
		return res, err
	}
	i := slices.IndexFunc(releases, func(r goals.Release) bool { return r.Version == in.Version })
	if i < 0 {
		return res, fmt.Errorf("%s names no release %s; write it with tpm plan write first", RoadMap, in.Version)
	}
	r := Release{Version: in.Version, Items: releases[i].Objectives, Milestone: releases[i].Milestone, Tag: releases[i].Tag}
	res.Release = r
	if r.Milestone == "" || r.Tag == "" {
		return res, fmt.Errorf("release %s has no milestone or tag in release.publication", r.Version)
	}
	docs, err := objectives.LoadDir(filepath.Join(in.Dir, objectives.Dir))
	if err != nil {
		return res, err
	}
	groups := groupsOf(r.Items, docs)

	m := tracker.Milestone{Title: r.Milestone, StartDate: start.Format(time.DateOnly), DueDate: in.End,
		Description: fmt.Sprintf("Release %s, %s to %s.", r.Version, isoWeek(start), isoWeek(end))}
	if in.DryRun {
		res.Milestone = "dry-run"
	} else if res.Milestone, err = t.EnsureMilestone(in.Group, m); err != nil {
		return res, err
	}

	if res.Sprint, err = pushSprint(t, in, r, groups); err != nil {
		return res, err
	}
	subs, err := program.LoadSubsystems(t, in.System, in.Ref)
	if err != nil {
		return res, err
	}
	repos := subs.Repositories()
	if !slices.Contains(repos, program.ProjectPath(in.System)) {
		res.Skipped = append(res.Skipped, in.System+": no subsystem SRD declares it as a component, so it has no sprint-end issue")
	}
	sort.Strings(repos)
	for _, repo := range slices.Compact(repos) {
		a, err := pushSprintEnd(t, in, r, repo)
		if err != nil {
			return res, err
		}
		res.SprintEnds = append(res.SprintEnds, a)
	}
	if res.Planned, err = postPlanned(t, in, r, repos, docs, groups); err != nil {
		return res, err
	}
	return res, nil
}

// group is one objective of the release and the items it places.
type group struct {
	key, title, docPath, environment string
	items                            []string
}

func groupsOf(items []string, docs []*objectives.Document) []group {
	var out []group
	for _, id := range items {
		key, item, _ := strings.Cut(id, "#")
		gid, _, _ := strings.Cut(item, ".")
		k := key + "#" + gid
		if j := slices.IndexFunc(out, func(g group) bool { return g.key == k }); j >= 0 {
			out[j].items = append(out[j].items, item)
			continue
		}
		g := group{key: k, environment: "dev", items: []string{item}}
		for _, d := range docs {
			if d.ID == key {
				g.title, g.docPath = d.Requirements[gid].Title, filepath.ToSlash(filepath.Join(objectives.Dir, filepath.Base(d.Path)))
				if c := d.CriteriaFor(gid); len(c) > 0 && c[0].Environment != "" {
					g.environment = c[0].Environment
				}
			}
		}
		out = append(out, g)
	}
	return out
}

func sprintMarker(version string) string {
	return fmt.Sprintf("<!-- tpm plan sprint release=%s -->", version)
}
func sprintEndMarker(version string) string {
	return fmt.Sprintf("<!-- tpm plan sprint-end release=%s -->", version)
}

var checked = regexp.MustCompile(`(?m)^- \[x\] (\S+?):`)

// pushSprint keeps one sprint issue for the release with one checklist item
// per objective; an item a person ticked stays ticked (srd001 R7.2, R7.4).
func pushSprint(t tracker.Tracker, in PushInput, r Release, groups []group) (IssueAction, error) {
	a := IssueAction{Project: in.System}
	existing, err := find(t, in.System, "type::sprint", sprintMarker(r.Version))
	if err != nil {
		return a, err
	}
	ticked := map[string]bool{}
	if existing != nil {
		for _, m := range checked.FindAllStringSubmatch(existing.Description, -1) {
			ticked[m[1]] = true
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "The sprint for release %s, ending %s. The program lead ticks each objective at the period's end; no script ticks or closes this issue (srd001 R7.4).\n\n## Exit checklist\n\n", r.Version, in.End)
	for _, g := range groups {
		box := " "
		if ticked[g.key] {
			box = "x"
		}
		fmt.Fprintf(&b, "- [%s] %s: %s\n", box, g.key, g.title)
	}
	b.WriteString("\n" + sprintMarker(r.Version) + "\n")
	return upsert(t, in, a, existing, tracker.NewIssue{Title: "Sprint " + r.Version, Description: b.String(),
		Labels: []string{"plane::management", "type::sprint"}, DueDate: in.End, Milestone: r.Milestone})
}

// pushSprintEnd keeps one sprint-end issue for the release in repo.
func pushSprintEnd(t tracker.Tracker, in PushInput, r Release, repo string) (IssueAction, error) {
	a := IssueAction{Project: repo}
	existing, err := find(t, repo, "type::recurring", sprintEndMarker(r.Version))
	if err != nil {
		return a, err
	}
	l, err := program.Derive(t, repo, in.System, in.Ref, nil, program.Labelling{Plane: "data", Kind: "release"})
	if err != nil {
		return a, err
	}
	labels := append([]string{"type::recurring"}, program.Labels(l.Plane, "", l.Kind, l.Component)...)
	return upsert(t, in, a, existing, tracker.NewIssue{Title: "Recurring: sprint end " + r.Version,
		Description: SprintEndBody(r.Version, in.End), Labels: labels, DueDate: in.End, Milestone: r.Milestone})
}

// SprintEndBody is the sprint-end issue's description: its Requirements are
// the steps srd038 R1.2 lists, in order.
func SprintEndBody(version, end string) string {
	return fmt.Sprintf(`The end of release %[1]s's period, due %[2]s (srd038 R1).

## Requirements

1. Run `+"`tpm sprint-end facts --json`"+` in this repository's checkout for the period's aggregate, each objective issue's ledger and reported status, and the recommended version.
2. Model step (combine): draft the period's summary from the period's reports and that output.
3. Ask the owner whether anything in the summary, the objectives, or the release should change, and apply what the owner names in the worktree.
4. Commit the summary at docs/sprint-ends/%[1]s.md and merge it with `+"`tpm issue merge`"+`, adding `+"`--closes`"+` for each objective the owner reports complete; the merge closes this issue.
5. Ask the owner to confirm the release and its version. With no confirmation, or nothing to release, stop: the merge alone closes this issue.
6. Once the summary merges, run `+"`tpm release gate`"+` and `+"`tpm release tag`"+` on main with this issue as the release issue.

## Acceptance Criteria

- The summary merges, and a confirmed release is tagged on main after a passing gate (test: integration_sprint_end; environment: dev)

## Recurrence

Once per period. tpm plan push for the next release files the next sprint-end issue.

## Previous Runs

- none

%[3]s
`, version, end, sprintEndMarker(version))
}

// find returns the open issue in project carrying marker, if any.
func find(t tracker.Tracker, project, label, marker string) (*tracker.Issue, error) {
	issues, err := t.ListIssues(project, tracker.IssueQuery{State: "opened", Labels: []string{label}})
	if err != nil {
		return nil, err
	}
	for i := range issues {
		if strings.Contains(issues[i].Description, marker) {
			return &issues[i], nil
		}
	}
	return nil, nil
}

// upsert files in when there is no existing issue, else updates its
// description and due date when they differ.
func upsert(t tracker.Tracker, in PushInput, a IssueAction, existing *tracker.Issue, n tracker.NewIssue) (IssueAction, error) {
	if existing == nil {
		r, err := program.File(t, a.Project, n, in.Policy, in.DryRun)
		if err != nil {
			return a, err
		}
		a.Action, a.IID, a.Admitted, a.Reasons = "filed", r.IID, r.Lint.Admitted, r.Lint.Reasons
		return a, nil
	}
	a.IID, a.Admitted = existing.IID, true
	// A tracker may store a description without its trailing newline, so
	// the comparison ignores the whitespace around it.
	if strings.TrimSpace(existing.Description) == strings.TrimSpace(n.Description) && existing.DueDate == n.DueDate {
		a.Action = "unchanged"
		return a, nil
	}
	a.Action = "updated"
	if in.DryRun {
		return a, nil
	}
	_, err := t.UpdateIssue(a.Project, existing.IID, tracker.IssueUpdate{Description: &n.Description, DueDate: &n.DueDate})
	return a, err
}

// postPlanned posts a planned event on each open objective issue for one of
// the release's objectives that tasks filed for another release
// (srd031 R4.3, srd006 R8).
func postPlanned(t tracker.Tracker, in PushInput, r Release, repos []string, docs []*objectives.Document, groups []group) ([]PlannedOnIssue, error) {
	var out []PlannedOnIssue
	for _, repo := range repos {
		issues, err := t.ListIssues(repo, tracker.IssueQuery{State: "opened", Labels: []string{"plane::management"}})
		if err != nil {
			return out, err
		}
		for _, issue := range issues {
			key := objectiveKey(issue.Description)
			j := slices.IndexFunc(groups, func(g group) bool { return g.key == key })
			if j < 0 || strings.Contains(issue.Description, "release="+r.Version+" -->") {
				continue
			}
			g := groups[j]
			var cites []string
			for _, item := range g.items {
				cites = append(cites, goals.CiteURL(in.WebBase, in.System, r.Tag, g.docPath, item))
			}
			e, text := events.Planned(r.Version, cites, g.environment)
			posted, err := events.Post(t, repo, issue.IID, e, events.PlannedRequired, text, in.DryRun)
			if err != nil {
				return out, err
			}
			out = append(out, PlannedOnIssue{Issue: fmt.Sprintf("%s#%d", repo, issue.IID), Posted: posted})
		}
	}
	return out, nil
}

// isoWeek names the ISO week of a date, such as 2026-W43.
func isoWeek(d time.Time) string {
	y, w := d.ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}
