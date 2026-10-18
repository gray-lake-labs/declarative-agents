// Package gitlabtest is an in-memory Tracker for unit tests.
package gitlabtest

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
)

// Fake holds projects' files, tags, issues, notes, and milestones in memory.
type Fake struct {
	Files      map[string][]byte // "project@ref:path"
	Tags       map[string]bool   // "project@tag"
	Issues     map[string][]*gitlab.Issue
	Notes      map[string][]gitlab.Note // "project#iid"
	Milestones map[string]gitlab.Milestone
	Awards     map[string][]gitlab.Award // "project#iid"
	Labels     map[string][]gitlab.Label // group -> labels
	Me         gitlab.User
	Branches   map[string]string // "project@branch" -> ref it was created from
	Default    string            // default branch of every project
	// Ahead maps "project@branch" to the commits it has over the default
	// branch and the paths they change.
	Ahead map[string]Commits
	MRs   map[string][]*gitlab.MergeRequest // project -> merge requests
	// Writes counts every mutating call, so a dry run can prove it wrote nothing.
	Writes int
}

var _ gitlab.Tracker = (*Fake)(nil)

// New returns an empty fake.
func New() *Fake {
	return &Fake{Files: map[string][]byte{}, Tags: map[string]bool{}, Issues: map[string][]*gitlab.Issue{},
		Notes: map[string][]gitlab.Note{}, Milestones: map[string]gitlab.Milestone{}, Awards: map[string][]gitlab.Award{}, Labels: map[string][]gitlab.Label{},
		Me: gitlab.User{ID: 1, Username: "me"}, Branches: map[string]string{}, Default: "main",
		Ahead: map[string]Commits{}, MRs: map[string][]*gitlab.MergeRequest{}}
}

// AddFile stores content at project, ref, path.
func (f *Fake) AddFile(project, ref, path string, content []byte) {
	f.Files[project+"@"+ref+":"+path] = content
}

// AddTag records a tag.
func (f *Fake) AddTag(project, tag string) { f.Tags[project+"@"+tag] = true }

func (f *Fake) GetFile(project, path, ref string) ([]byte, error) {
	b, ok := f.Files[project+"@"+ref+":"+path]
	if !ok {
		return nil, gitlab.ErrNotFound
	}
	return b, nil
}

func (f *Fake) ListDir(project, dir, ref string) ([]string, error) {
	prefix := project + "@" + ref + ":" + strings.TrimSuffix(dir, "/") + "/"
	var out []string
	for k := range f.Files {
		if rest, ok := strings.CutPrefix(k, prefix); ok && !strings.Contains(rest, "/") {
			out = append(out, strings.TrimPrefix(k, project+"@"+ref+":"))
		}
	}
	slices.Sort(out)
	return out, nil
}

func (f *Fake) TagExists(project, tag string) (bool, error) { return f.Tags[project+"@"+tag], nil }

func (f *Fake) find(project string, iid int) (*gitlab.Issue, error) {
	for _, i := range f.Issues[project] {
		if i.IID == iid {
			return i, nil
		}
	}
	return nil, gitlab.ErrNotFound
}

func (f *Fake) GetIssue(project string, iid int) (gitlab.Issue, error) {
	i, err := f.find(project, iid)
	if err != nil {
		return gitlab.Issue{}, err
	}
	return *i, nil
}

func (f *Fake) ListIssues(project string, q gitlab.IssueQuery) ([]gitlab.Issue, error) {
	var out []gitlab.Issue
	for _, i := range f.Issues[project] {
		if q.State != "" && q.State != "all" && i.State != q.State {
			continue
		}
		if !hasAll(i.Labels, q.Labels) {
			continue
		}
		if q.Search != "" && !strings.Contains(i.Title+" "+i.Description, q.Search) {
			continue
		}
		if q.UpdatedAfter != "" && i.UpdatedAt != "" && i.UpdatedAt < q.UpdatedAfter {
			continue
		}
		out = append(out, *i)
	}
	return out, nil
}

func hasAll(have, want []string) bool {
	for _, w := range want {
		if !slices.Contains(have, w) {
			return false
		}
	}
	return true
}

func (f *Fake) CreateIssue(project string, in gitlab.NewIssue) (gitlab.Issue, error) {
	f.Writes++
	i := &gitlab.Issue{Project: project, IID: len(f.Issues[project]) + 1, Title: in.Title, Description: in.Description,
		Labels: slices.Clone(in.Labels), State: "opened", DueDate: in.DueDate, Milestone: in.Milestone, Author: f.Me.Username}
	i.WebURL = fmt.Sprintf("https://gitlab.example.com/%s/-/issues/%d", project, i.IID)
	f.Issues[project] = append(f.Issues[project], i)
	return *i, nil
}

func (f *Fake) UpdateIssue(project string, iid int, u gitlab.IssueUpdate) (gitlab.Issue, error) {
	f.Writes++
	i, err := f.find(project, iid)
	if err != nil {
		return gitlab.Issue{}, err
	}
	if u.Description != nil {
		i.Description = *u.Description
	}
	if u.State != nil {
		i.State = map[string]string{"close": "closed", "reopen": "opened"}[*u.State]
	}
	if u.DueDate != nil {
		i.DueDate = *u.DueDate
	}
	for _, l := range u.AddLabels {
		if !slices.Contains(i.Labels, l) {
			i.Labels = append(i.Labels, l)
		}
	}
	i.Labels = slices.DeleteFunc(i.Labels, func(l string) bool { return slices.Contains(u.RemoveLabels, l) })
	return *i, nil
}

func (f *Fake) ListNotes(project string, iid int) ([]gitlab.Note, error) {
	return f.Notes[fmt.Sprintf("%s#%d", project, iid)], nil
}

func (f *Fake) CreateNote(project string, iid int, body string) error {
	f.Writes++
	key := fmt.Sprintf("%s#%d", project, iid)
	f.Notes[key] = append(f.Notes[key], gitlab.Note{ID: len(f.Notes[key]) + 1, Body: body, Author: f.Me.Username})
	return nil
}

func (f *Fake) GroupLabels(group string) ([]gitlab.Label, error) {
	return slices.Clone(f.Labels[group]), nil
}

func (f *Fake) CreateGroupLabel(group string, l gitlab.Label) error {
	f.Writes++
	for _, x := range f.Labels[group] {
		if x.Name == l.Name {
			return fmt.Errorf("label %q already exists", l.Name)
		}
	}
	f.Labels[group] = append(f.Labels[group], l)
	return nil
}

func (f *Fake) UpdateGroupLabel(group string, l gitlab.Label) error {
	f.Writes++
	for i, x := range f.Labels[group] {
		if x.Name == l.Name {
			f.Labels[group][i] = l
			return nil
		}
	}
	return gitlab.ErrNotFound
}

func (f *Fake) CurrentUser() (gitlab.User, error) { return f.Me, nil }

func (f *Fake) AssignIssue(project string, iid int, user gitlab.User) error {
	f.Writes++
	i, err := f.find(project, iid)
	if err != nil {
		return err
	}
	i.Assignees = []string{user.Username}
	return nil
}

func (f *Fake) DefaultBranch(string) (string, error) { return f.Default, nil }

func (f *Fake) BranchExists(project, branch string) (bool, error) {
	_, ok := f.Branches[project+"@"+branch]
	return ok, nil
}

func (f *Fake) CreateBranch(project, branch, ref string) error {
	f.Writes++
	if _, ok := f.Branches[project+"@"+branch]; ok {
		return fmt.Errorf("branch %s exists", branch)
	}
	f.Branches[project+"@"+branch] = ref
	return nil
}

// Commits is what a branch has over the default branch.
type Commits struct {
	Count int
	Paths []string
}

func (f *Fake) Compare(project, base, head string) (int, []string, error) {
	c := f.Ahead[project+"@"+head]
	return c.Count, c.Paths, nil
}

func (f *Fake) OpenMergeRequest(project, branch string) (*gitlab.MergeRequest, error) {
	for _, m := range f.MRs[project] {
		if m.SourceBranch == branch && m.State == "opened" {
			c := *m
			return &c, nil
		}
	}
	return nil, nil
}

func (f *Fake) CreateMergeRequest(project string, mr gitlab.MergeRequest) (gitlab.MergeRequest, error) {
	f.Writes++
	mr.IID = len(f.MRs[project]) + 1
	mr.State = "opened"
	mr.WebURL = fmt.Sprintf("https://gitlab.example.com/%s/-/merge_requests/%d", project, mr.IID)
	c := mr
	f.MRs[project] = append(f.MRs[project], &c)
	return mr, nil
}

// MergedMergeRequests returns the fake's merge requests in the project
// that are merged, by author (any when empty), on or after since.
func (f *Fake) MergedMergeRequests(project, author, since string) ([]gitlab.MergeRequest, error) {
	var out []gitlab.MergeRequest
	for _, m := range f.MRs[project] {
		if m.State == "merged" && (author == "" || m.Author == author) && m.MergedAt >= since {
			out = append(out, *m)
		}
	}
	return out, nil
}

func (f *Fake) IssueAwards(project string, iid int) ([]gitlab.Award, error) {
	return f.Awards[fmt.Sprintf("%s#%d", project, iid)], nil
}

// ActiveMilestones returns every milestone the fake holds, by title.
func (f *Fake) ActiveMilestones(project string) ([]gitlab.Milestone, error) {
	var out []gitlab.Milestone
	for _, title := range slices.Sorted(maps.Keys(f.Milestones)) {
		out = append(out, f.Milestones[title])
	}
	return out, nil
}

// EnsureMilestone keeps one milestone per title, as the client does for a
// group, and counts a write only when it creates or changes one.
func (f *Fake) EnsureMilestone(group string, m gitlab.Milestone) (string, error) {
	cur, ok := f.Milestones[m.Title]
	switch {
	case !ok:
		f.Milestones[m.Title] = m
		f.Writes++
		return gitlab.MilestoneCreated, nil
	case cur == m:
		return gitlab.MilestoneUnchanged, nil
	}
	f.Milestones[m.Title] = m
	f.Writes++
	return gitlab.MilestoneUpdated, nil
}

func (f *Fake) FindMilestone(project, title string) (gitlab.Milestone, error) {
	m, ok := f.Milestones[title]
	if !ok {
		return gitlab.Milestone{}, gitlab.ErrNotFound
	}
	return m, nil
}
