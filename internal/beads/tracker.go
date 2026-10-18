package beads

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
)

// ErrNotServed is returned by the tracker methods the beads backend does not
// serve yet; the commands that need them still run on GitLab (GAP10).
var ErrNotServed = errors.New("not served by the beads tracker yet (GAP10)")

// statusInProgress is the label the GitLab tracker uses for in-progress; the
// beads tracker shows it from the state and fires the transition when a
// command adds or removes it.
const statusInProgress = "status::in-progress"

// Tracker serves tpm's tracker interface over beads (srd042 R1.1): the issue
// half from the tracker repository, the code half from the git checkout the
// command runs in. A project path is an issue's Repository; the code half
// serves its one checkout whatever project it is asked about.
type Tracker struct {
	Store *Store
	code  string
}

var _ gitlab.Tracker = (*Tracker)(nil)

// NewTracker serves the tracker repository cloned at clone, as user, with
// the code half answered by the checkout at code.
func NewTracker(clone, user, code string) *Tracker {
	return &Tracker{Store: NewStore(clone, user), code: code}
}

// Configured reports whether TPM_TRACKER names a tracker repository, which
// selects the beads backend over GitLab.
func Configured() bool { return strings.TrimSpace(os.Getenv("TPM_TRACKER")) != "" }

// FromEnv opens the beads tracker TPM_TRACKER names. Its clone is
// TPM_TRACKER_CLONE, else a directory under the user cache, cloned on first
// use; the person is TPM_USER, else git's user.name; the code half serves
// the checkout around the working directory.
func FromEnv() (*Tracker, error) {
	remote := strings.TrimSpace(os.Getenv("TPM_TRACKER"))
	clone := strings.TrimSpace(os.Getenv("TPM_TRACKER_CLONE"))
	if clone == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256([]byte(remote))
		clone = filepath.Join(cache, "tpm", "trackers", hex.EncodeToString(sum[:6]))
	}
	if _, err := os.Stat(filepath.Join(clone, ".git")); err != nil {
		if err := os.MkdirAll(filepath.Dir(clone), 0o755); err != nil {
			return nil, err
		}
		if out, err := exec.Command("git", "clone", "-q", remote, clone).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("clone the tracker repository %s: %w: %s", remote, err, strings.TrimSpace(string(out)))
		}
	}
	user := strings.TrimSpace(os.Getenv("TPM_USER"))
	if user == "" {
		out, err := exec.Command("git", "config", "user.name").Output()
		if user = strings.TrimSpace(string(out)); err != nil || user == "" {
			return nil, errors.New("no person for the beads tracker: set TPM_USER or git config user.name")
		}
	}
	code, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return NewTracker(clone, user, code), nil
}

func (t *Tracker) find(project string, iid int) (Issue, error) {
	issues, _, err := t.Store.Issues()
	if err != nil {
		return Issue{}, err
	}
	for _, i := range issues {
		if i.Repository == project && i.IID == iid {
			return i, nil
		}
	}
	return Issue{}, fmt.Errorf("%s#%d: %w", project, iid, gitlab.ErrNotFound)
}

func (t *Tracker) view(i Issue) gitlab.Issue {
	labels := slices.Clone(i.Labels)
	if i.State == InProgress {
		labels = append(labels, statusInProgress)
	}
	state := "opened"
	if i.Status == StatusClosed {
		state = "closed"
	}
	var assignees []string
	if i.Assignee != "" {
		assignees = []string{i.Assignee}
	}
	return gitlab.Issue{Project: i.Repository, IID: i.IID, Title: i.Title, Description: i.Description, Labels: labels,
		State: state, WebURL: "beads:" + i.ID, DueDate: i.DueDate, Milestone: i.Release, Assignees: assignees,
		Author: i.CreatedBy, UpdatedAt: i.UpdatedAt}
}

func (t *Tracker) GetIssue(project string, iid int) (gitlab.Issue, error) {
	i, err := t.find(project, iid)
	if err != nil {
		return gitlab.Issue{}, err
	}
	return t.view(i), nil
}

func (t *Tracker) ListIssues(project string, q gitlab.IssueQuery) ([]gitlab.Issue, error) {
	issues, _, err := t.Store.Issues()
	if err != nil {
		return nil, err
	}
	var out []gitlab.Issue
	for _, i := range issues {
		v := t.view(i)
		switch {
		case i.Repository != project,
			q.State == "opened" && v.State != "opened",
			q.State == "closed" && v.State != "closed",
			slices.ContainsFunc(q.Labels, func(l string) bool { return !v.HasLabel(l) }),
			q.Search != "" && !strings.Contains(strings.ToLower(v.Title+"\n"+v.Description), strings.ToLower(q.Search)),
			q.UpdatedAfter != "" && v.UpdatedAt < q.UpdatedAfter:
			continue
		}
		out = append(out, v)
	}
	slices.SortFunc(out, func(a, b gitlab.Issue) int { return a.IID - b.IID })
	return out, nil
}

func (t *Tracker) CreateIssue(project string, in gitlab.NewIssue) (gitlab.Issue, error) {
	i, err := t.Store.Create(Issue{Repository: project, Title: in.Title, Description: in.Description,
		Labels: slices.Clone(in.Labels), DueDate: in.DueDate, Release: in.Milestone})
	if err != nil {
		return gitlab.Issue{}, err
	}
	return t.view(i), nil
}

// UpdateIssue applies an update. Closing is a person's decline and
// reopening a person's reopen (srd041); the in-progress label fires the
// assignee's start and stop.
func (t *Tracker) UpdateIssue(project string, iid int, u gitlab.IssueUpdate) (gitlab.Issue, error) {
	found, err := t.find(project, iid)
	if err != nil {
		return gitlab.Issue{}, err
	}
	me := t.Store.User
	_, _, _, err = t.Store.mutate(found.ID, "update", func(i *Issue) (bool, error) {
		if u.State != nil {
			to := map[string]string{"close": Closed, "reopen": Open}[*u.State]
			if to == "" {
				return false, fmt.Errorf("state %q is neither close nor reopen", *u.State)
			}
			if err := transition(i, ActorPerson, to, me); err != nil {
				return false, err
			}
		}
		if u.Description != nil {
			i.Description = *u.Description
		}
		if u.DueDate != nil {
			i.DueDate = *u.DueDate
		}
		for _, l := range u.AddLabels {
			if l == statusInProgress {
				if i.State != InProgress {
					if err := transition(i, ActorPerson, InProgress, me); err != nil {
						return false, err
					}
				}
			} else if !i.HasLabel(l) {
				i.Labels = append(i.Labels, l)
			}
		}
		for _, l := range u.RemoveLabels {
			if l == statusInProgress {
				if i.State == InProgress {
					if err := transition(i, ActorPerson, Taken, me); err != nil {
						return false, err
					}
				}
			} else {
				i.Labels = slices.DeleteFunc(i.Labels, func(x string) bool { return x == l })
			}
		}
		return true, nil
	})
	if err != nil {
		return gitlab.Issue{}, err
	}
	return t.GetIssue(project, iid)
}

func (t *Tracker) ListNotes(project string, iid int) ([]gitlab.Note, error) {
	i, err := t.find(project, iid)
	if err != nil {
		return nil, err
	}
	notes := make([]gitlab.Note, 0, len(i.Comments))
	for k, c := range i.Comments {
		notes = append(notes, gitlab.Note{ID: k + 1, Body: c.Text, Author: c.Author, System: c.Author == "tpm"})
	}
	return notes, nil
}

func (t *Tracker) CreateNote(project string, iid int, body string) error {
	i, err := t.find(project, iid)
	if err != nil {
		return err
	}
	return t.Store.Comment(i.ID, body)
}

func (t *Tracker) CurrentUser() (gitlab.User, error) {
	return gitlab.User{Username: t.Store.User}, nil
}

// AssignIssue is pop's assignment (srd041 R9.1): only the person running
// it, and a pop that lost to an earlier taker is an error naming them.
func (t *Tracker) AssignIssue(project string, iid int, user gitlab.User) error {
	if user.Username != t.Store.User {
		return fmt.Errorf("%w: pop assigns only the person who runs it, not %s (srd041 R9.1)", ErrRefused, user.Username)
	}
	i, err := t.find(project, iid)
	if err != nil {
		return err
	}
	r, err := t.Store.Pop(i.ID)
	if err != nil {
		return err
	}
	if r.Lost {
		return fmt.Errorf("%s#%d was taken by %s first", project, iid, r.LostTo)
	}
	return nil
}

// CloseOnMerge closes an issue on the local merge that delivered it, citing
// the merge commit (srd042 R2.3). It reports whether anything changed.
func (t *Tracker) CloseOnMerge(project string, iid int, commit string) (bool, error) {
	i, err := t.find(project, iid)
	if err != nil {
		return false, err
	}
	r, err := t.Store.Close(i.ID, commit)
	return r.Changed, err
}

// The code half: the checkout's own git and its origin.

func (t *Tracker) git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = t.code
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (t *Tracker) DefaultBranch(string) (string, error) {
	ref, err := t.git("symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err != nil {
		return "main", nil
	}
	return strings.TrimPrefix(ref, "origin/"), nil
}

func (t *Tracker) BranchExists(_, branch string) (bool, error) {
	out, err := t.git("ls-remote", "--heads", "origin", "refs/heads/"+branch)
	return out != "", err
}

func (t *Tracker) CreateBranch(_, branch, ref string) error {
	if _, err := t.git("fetch", "-q", "origin", ref); err != nil {
		return err
	}
	_, err := t.git("push", "-q", "origin", "refs/remotes/origin/"+ref+":refs/heads/"+branch)
	return err
}

func (t *Tracker) rev(ref string) string {
	if _, err := t.git("rev-parse", "--verify", "-q", "refs/remotes/origin/"+ref); err == nil {
		return "origin/" + ref
	}
	return ref
}

func (t *Tracker) GetFile(_, path, ref string) ([]byte, error) {
	cmd := exec.Command("git", "show", t.rev(ref)+":"+path)
	cmd.Dir = t.code
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s at %s: %w", path, ref, gitlab.ErrNotFound)
	}
	return out, nil
}

func (t *Tracker) ListDir(_, dir, ref string) ([]string, error) {
	out, err := t.git("ls-tree", "--name-only", t.rev(ref), strings.TrimSuffix(dir, "/")+"/")
	if err != nil || out == "" {
		return nil, fmt.Errorf("%s at %s: %w", dir, ref, gitlab.ErrNotFound)
	}
	return strings.Split(out, "\n"), nil
}

func (t *Tracker) TagExists(_, tag string) (bool, error) {
	out, err := t.git("ls-remote", "--tags", "origin", "refs/tags/"+tag)
	return out != "", err
}

func (t *Tracker) Compare(_, base, head string) (int, []string, error) {
	if _, err := t.git("fetch", "-q", "origin"); err != nil {
		return 0, nil, err
	}
	b, h := t.rev(base), t.rev(head)
	count, err := t.git("rev-list", "--count", b+".."+h)
	if err != nil {
		return 0, nil, err
	}
	ahead, _ := strconv.Atoi(count)
	paths, err := t.git("diff", "--name-only", b+"..."+h)
	if err != nil || paths == "" {
		return ahead, nil, err
	}
	return ahead, strings.Split(paths, "\n"), nil
}

// Not served yet: milestones, merge requests, awards, and group labels
// belong to commands that still run on GitLab (GAP10).

func (t *Tracker) FindMilestone(string, string) (gitlab.Milestone, error) {
	return gitlab.Milestone{}, ErrNotServed
}

func (t *Tracker) ActiveMilestones(string) ([]gitlab.Milestone, error) { return nil, ErrNotServed }

func (t *Tracker) EnsureMilestone(string, gitlab.Milestone) (string, error) { return "", ErrNotServed }

func (t *Tracker) MergedMergeRequests(string, string, string) ([]gitlab.MergeRequest, error) {
	return nil, ErrNotServed
}

func (t *Tracker) IssueAwards(string, int) ([]gitlab.Award, error) { return nil, ErrNotServed }

func (t *Tracker) GroupLabels(string) ([]gitlab.Label, error) { return nil, ErrNotServed }

func (t *Tracker) CreateGroupLabel(string, gitlab.Label) error { return ErrNotServed }

func (t *Tracker) UpdateGroupLabel(string, gitlab.Label) error { return ErrNotServed }

func (t *Tracker) OpenMergeRequest(string, string) (*gitlab.MergeRequest, error) {
	return nil, ErrNotServed
}

func (t *Tracker) CreateMergeRequest(string, gitlab.MergeRequest) (gitlab.MergeRequest, error) {
	return gitlab.MergeRequest{}, ErrNotServed
}
