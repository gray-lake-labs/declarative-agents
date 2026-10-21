package beads

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// statusInProgress is the label commands use for in-progress; the
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
	// fetched records the projects whose origin this command has fetched,
	// so files read reflect what other clones pushed, at one fetch each.
	fetched map[string]bool
}

var (
	_ tracker.Tracker = (*Tracker)(nil)
	_ tracker.Reader  = (*Tracker)(nil)
	_ tracker.Triager = (*Tracker)(nil)
)

// LastRead names the tracker commit the last read reflected.
func (t *Tracker) LastRead() (tracker.Read, bool) {
	fr, ok := t.Store.Last()
	if !ok {
		return tracker.Read{}, false
	}
	return tracker.Read{Commit: fr.Commit, At: fr.At.UTC().Format(time.RFC3339), Offline: fr.Offline}, true
}

// NewTracker serves the tracker repository cloned at clone, as user, with
// the code half answered by the checkout at code.
func NewTracker(clone, user, code string) *Tracker {
	return &Tracker{Store: NewStore(clone, user), code: code}
}

// Configured reports whether TPM_TRACKER names a tracker repository.
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
	i, ok, err := t.Store.Find(project, iid)
	if err != nil {
		return Issue{}, err
	}
	if !ok {
		return Issue{}, fmt.Errorf("%s#%d: %w", project, iid, tracker.ErrNotFound)
	}
	return i, nil
}

func (t *Tracker) view(i Issue) tracker.Issue {
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
	return tracker.Issue{Project: i.Repository, IID: i.IID, Title: i.Title, Description: i.Description, Labels: labels,
		State: state, WebURL: "beads:" + i.ID, DueDate: i.DueDate, Milestone: i.Release, Assignees: assignees,
		Author: i.CreatedBy, CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt}
}

func (t *Tracker) GetIssue(project string, iid int) (tracker.Issue, error) {
	i, err := t.find(project, iid)
	if err != nil {
		return tracker.Issue{}, err
	}
	return t.view(i), nil
}

func (t *Tracker) ListIssues(project string, q tracker.IssueQuery) ([]tracker.Issue, error) {
	issues, _, err := t.Store.Issues()
	if err != nil {
		return nil, err
	}
	var out []tracker.Issue
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
	slices.SortFunc(out, func(a, b tracker.Issue) int { return a.IID - b.IID })
	return out, nil
}

// CreateAdmitted creates the issue and records lint's verdict on it in one
// tracker write: the srd041 triage rows and, on a failing issue, admit's
// note as a comment.
func (t *Tracker) CreateAdmitted(project string, in tracker.NewIssue, admit func(tracker.Issue) ([]string, string)) (tracker.Issue, error) {
	me := t.Store.User
	i, err := t.Store.CreateWith(Issue{Repository: project, Title: in.Title, Description: in.Description,
		Labels: slices.Clone(in.Labels), DueDate: in.DueDate, Release: in.Milestone}, func(i *Issue) error {
		reasons, note := admit(t.view(*i))
		if _, err := triage(i, len(reasons) == 0, me); err != nil {
			return err
		}
		if note != "" {
			i.Comments = append(i.Comments, Comment{ID: newID(), IssueID: i.ID, Author: me, CreatedAt: t.Store.stamp(), Text: note})
		}
		return nil
	})
	if err != nil {
		return tracker.Issue{}, err
	}
	return t.view(i), nil
}

func (t *Tracker) CreateIssue(project string, in tracker.NewIssue) (tracker.Issue, error) {
	i, err := t.Store.Create(Issue{Repository: project, Title: in.Title, Description: in.Description,
		Labels: slices.Clone(in.Labels), DueDate: in.DueDate, Release: in.Milestone})
	if err != nil {
		return tracker.Issue{}, err
	}
	return t.view(i), nil
}

// UpdateIssue applies an update. Closing is a person's decline and
// reopening a person's reopen (srd041); the in-progress label fires the
// assignee's start and stop.
func (t *Tracker) UpdateIssue(project string, iid int, u tracker.IssueUpdate) (tracker.Issue, error) {
	found, err := t.find(project, iid)
	if err != nil {
		return tracker.Issue{}, err
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
		if u.Milestone != nil {
			i.Release = *u.Milestone
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
		return tracker.Issue{}, err
	}
	return t.GetIssue(project, iid)
}

func (t *Tracker) ListNotes(project string, iid int) ([]tracker.Note, error) {
	i, err := t.find(project, iid)
	if err != nil {
		return nil, err
	}
	notes := make([]tracker.Note, 0, len(i.Comments))
	for k, c := range i.Comments {
		notes = append(notes, tracker.Note{ID: k + 1, Body: c.Text, Author: c.Author, System: c.Author == "tpm"})
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

func (t *Tracker) CurrentUser() (tracker.User, error) {
	return tracker.User{Username: t.Store.User}, nil
}

// AssignIssue is pop's assignment (srd041 R9.1): only the person running
// it, and a pop that lost to an earlier taker is an error naming them.
func (t *Tracker) AssignIssue(project string, iid int, user tracker.User) error {
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

// Triage is admission as srd041's lint rows: an admitted open issue becomes
// ready, an admitted issue in needs-triage becomes ready or, already taken
// by its author, taken; a failing open issue goes to needs-triage. Only a
// data issue enters the backlog (srd041 R3.1), so admission moves only a
// data issue's state; for any other issue, and for an issue already past
// admission that later fails, for which srd041 has no row, lint sets or
// clears only the label.
func (t *Tracker) Triage(project string, iid int, admitted bool) (bool, error) {
	found, err := t.find(project, iid)
	if err != nil {
		return false, err
	}
	me := t.Store.User
	_, written, _, err := t.Store.mutate(found.ID, "lint", func(i *Issue) (bool, error) { return triage(i, admitted, me) })
	return written, err
}

// triage is lint's srd041 rows on one issue: an admitted open data issue
// becomes ready, an admitted one in needs-triage ready or, already taken by
// its author, taken; a failing open data issue goes to needs-triage, and an
// issue on another plane carries the needs-triage label instead.
func triage(i *Issue, admitted bool, me string) (bool, error) {
	data := i.HasLabel("plane::data")
	switch {
	case !data && admitted && i.HasLabel(NeedsTriage):
		i.Labels = slices.DeleteFunc(i.Labels, func(l string) bool { return l == NeedsTriage })
		return true, nil
	case !data && !admitted && !i.HasLabel(NeedsTriage):
		i.Labels = append(i.Labels, NeedsTriage)
		return true, nil
	case !data:
		return false, nil
	case admitted && (i.State == Open || i.State == NeedsTriage) && i.Assignee == "":
		return true, transition(i, ActorLint, Ready, me)
	case admitted && i.State == NeedsTriage:
		return true, transition(i, ActorLint, Taken, me)
	case admitted && i.HasLabel(NeedsTriage):
		i.Labels = slices.DeleteFunc(i.Labels, func(l string) bool { return l == NeedsTriage })
		return true, nil
	case !admitted && i.State == Open:
		return true, transition(i, ActorLint, NeedsTriage, me)
	case !admitted && !i.HasLabel(NeedsTriage):
		i.Labels = append(i.Labels, NeedsTriage)
		return true, nil
	}
	return false, nil
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

// The code half: each project's checkout and its origin. A project's
// checkout is the sibling of the command's checkout named for the project's
// last path segment, as repositories sit side by side under their owner;
// a project with no such sibling is the command's own checkout.

func (t *Tracker) checkout(project string) string {
	if project != "" {
		sib := filepath.Join(filepath.Dir(t.code), path.Base(project))
		if _, err := os.Stat(filepath.Join(sib, ".git")); err == nil {
			return sib
		}
	}
	return t.code
}

func (t *Tracker) git(project string, args ...string) (string, error) {
	out, err := t.raw(project, args...)
	return strings.TrimSpace(string(out)), err
}

func (t *Tracker) raw(project string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = t.checkout(project)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

func (t *Tracker) DefaultBranch(project string) (string, error) {
	ref, err := t.git(project, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err != nil {
		return "main", nil
	}
	return strings.TrimPrefix(ref, "origin/"), nil
}

func (t *Tracker) BranchExists(project, branch string) (bool, error) {
	out, err := t.git(project, "ls-remote", "--heads", "origin", "refs/heads/"+branch)
	return out != "", err
}

func (t *Tracker) CreateBranch(project, branch, ref string) error {
	if _, err := t.git(project, "fetch", "-q", "origin", ref); err != nil {
		return err
	}
	_, err := t.git(project, "push", "-q", "origin", "refs/remotes/origin/"+ref+":refs/heads/"+branch)
	return err
}

func (t *Tracker) rev(project, ref string) string {
	if !t.fetched[project] {
		if t.fetched == nil {
			t.fetched = map[string]bool{}
		}
		t.fetched[project] = true
		t.git(project, "fetch", "-q", "origin")
	}
	if _, err := t.git(project, "rev-parse", "--verify", "-q", "refs/remotes/origin/"+ref); err == nil {
		return "origin/" + ref
	}
	return ref
}

// GetFile reads p at ref as text: through the repository's textconv, so a
// git-crypt repository's files come back decrypted, and as stored where no
// textconv is configured.
func (t *Tracker) GetFile(project, p, ref string) ([]byte, error) {
	out, err := t.raw(project, "cat-file", "--textconv", t.rev(project, ref)+":"+p)
	if err != nil {
		return nil, fmt.Errorf("%s at %s: %w", p, ref, tracker.ErrNotFound)
	}
	return out, nil
}

func (t *Tracker) ListDir(project, dir, ref string) ([]string, error) {
	out, err := t.git(project, "ls-tree", "--name-only", t.rev(project, ref), strings.TrimSuffix(dir, "/")+"/")
	if err != nil || out == "" {
		return nil, fmt.Errorf("%s at %s: %w", dir, ref, tracker.ErrNotFound)
	}
	return strings.Split(out, "\n"), nil
}

func (t *Tracker) TagExists(project, tag string) (bool, error) {
	out, err := t.git(project, "ls-remote", "--tags", "origin", "refs/tags/"+tag)
	return out != "", err
}

// FindMilestone looks in the project, then in its ancestor groups.
func (t *Tracker) FindMilestone(project, title string) (tracker.Milestone, error) {
	ms, _, err := t.Store.Milestones()
	if err != nil {
		return tracker.Milestone{}, err
	}
	for _, scope := range ancestors(project) {
		for _, m := range ms {
			if m.Scope == scope && m.Title == title {
				return toTracker(m), nil
			}
		}
	}
	return tracker.Milestone{}, fmt.Errorf("milestone %s for %s: %w", title, project, tracker.ErrNotFound)
}

// ActiveMilestones lists the milestones of the project and its ancestor
// groups by due date. A beads milestone is never closed, so all are active.
func (t *Tracker) ActiveMilestones(project string) ([]tracker.Milestone, error) {
	ms, _, err := t.Store.Milestones()
	if err != nil {
		return nil, err
	}
	scopes := ancestors(project)
	var out []tracker.Milestone
	for _, m := range ms {
		if slices.Contains(scopes, m.Scope) {
			out = append(out, toTracker(m))
		}
	}
	slices.SortStableFunc(out, func(a, b tracker.Milestone) int {
		return strings.Compare(a.DueDate+"\x00"+a.Title, b.DueDate+"\x00"+b.Title)
	})
	return out, nil
}

func (t *Tracker) EnsureMilestone(group string, m tracker.Milestone) (string, error) {
	return t.Store.EnsureMilestone(Milestone{Scope: group, Title: m.Title, Description: m.Description,
		StartDate: m.StartDate, DueDate: m.DueDate})
}

func toTracker(m Milestone) tracker.Milestone {
	return tracker.Milestone{Title: m.Title, Description: m.Description, StartDate: m.StartDate, DueDate: m.DueDate}
}

// IssueAwards is always empty: beads has no award emoji, and a peer read
// is an approval comment instead (srd015 R1.2).
func (t *Tracker) IssueAwards(string, int) ([]tracker.Award, error) { return nil, nil }

// GroupLabels lists the labels defined on the group itself.
func (t *Tracker) GroupLabels(group string) ([]tracker.Label, error) {
	ls, _, err := t.Store.Labels()
	if err != nil {
		return nil, err
	}
	var out []tracker.Label
	for _, l := range ls {
		if l.Scope == group {
			out = append(out, tracker.Label{Name: l.Name, Color: l.Color, Description: l.Description})
		}
	}
	return out, nil
}

func (t *Tracker) CreateGroupLabel(group string, l tracker.Label) error {
	return t.Store.PutLabel(Label{Scope: group, Name: l.Name, Color: l.Color, Description: l.Description}, false)
}

func (t *Tracker) UpdateGroupLabel(group string, l tracker.Label) error {
	err := t.Store.PutLabel(Label{Scope: group, Name: l.Name, Color: l.Color, Description: l.Description}, true)
	if errors.Is(err, ErrNotFound) {
		return fmt.Errorf("%w: %v", tracker.ErrNotFound, err)
	}
	return err
}
