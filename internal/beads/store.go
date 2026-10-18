package beads

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// pushAttempts bounds how often a write re-synchronizes after its push is
// rejected by a newer tracker repository.
const pushAttempts = 5

// Store is one working clone of the tracker repository, used on behalf of
// one person. Its files are a cache; the tracker repository is the source
// of truth (srd042 R3.1).
type Store struct {
	Dir  string
	User string
	Now  func() time.Time
	// beforeWrite runs between a write's synchronization and its commit, so
	// a test can land another clone's write inside that window.
	beforeWrite func()
	// last is the most recent synchronization, what the last read reflects.
	last *Freshness
}

// Last is the most recent synchronization, if any.
func (s *Store) Last() (Freshness, bool) {
	if s.last == nil {
		return Freshness{}, false
	}
	return *s.last, true
}

// NewStore returns the store for the clone at dir, acting as user.
func NewStore(dir, user string) *Store {
	return &Store{Dir: dir, User: user, Now: time.Now}
}

// Freshness names the tracker repository commit a read reflects. Offline
// means the clone could not synchronize and served its cache, last pulled
// at At (srd042 R3.3).
type Freshness struct {
	Commit  string    `json:"commit"`
	At      time.Time `json:"at"`
	Offline bool      `json:"offline,omitempty"`
}

// PopResult is what a pop reports. A pop taken offline is Provisional until
// Reconcile confirms it or reports it lost; an online pop whose push lost to
// an earlier taker is Lost (eng04-beads-sync).
type PopResult struct {
	ID          string    `json:"id"`
	Assignee    string    `json:"assignee"`
	Provisional bool      `json:"provisional,omitempty"`
	Lost        bool      `json:"lost,omitempty"`
	LostTo      string    `json:"lost_to,omitempty"`
	Read        Freshness `json:"read"`
}

// CloseResult is what a close reports; a repeated close changes nothing.
type CloseResult struct {
	ID      string    `json:"id"`
	Changed bool      `json:"changed"`
	Read    Freshness `json:"read"`
}

// Reconciled is the outcome of one provisional pop once the clone is back
// online: "confirmed" or "lost to <taker>".
type Reconciled struct {
	ID      string `json:"id"`
	Outcome string `json:"outcome"`
}

// syncState is what a clone remembers between commands, kept under its git
// directory so it is never tracked.
type syncState struct {
	Commit      string        `json:"commit"`
	At          time.Time     `json:"at"`
	Provisional []Provisional `json:"provisional,omitempty"`
}

// Provisional is a pop taken while offline.
type Provisional struct {
	ID   string `json:"id"`
	User string `json:"user"`
}

// Issues pulls the tracker repository and returns every issue with the
// commit it reflects.
func (s *Store) Issues() ([]Issue, Freshness, error) {
	fr, err := s.Sync()
	if err != nil {
		return nil, fr, err
	}
	issues, err := s.read()
	return issues, fr, err
}

// Pop takes a ready issue for the store's user (srd041 R9.1). A second pop
// by the same person changes nothing; a pop of someone else's issue is
// refused.
func (s *Store) Pop(id string) (PopResult, error) {
	r := PopResult{ID: id}
	fr, written, queued, err := s.mutate(id, "pop", func(i *Issue) (bool, error) {
		if i.Assignee == s.User {
			return false, nil
		}
		if i.Assignee != "" {
			return false, fmt.Errorf("%s is assigned to %s; only that person pops it", id, i.Assignee)
		}
		return true, transition(i, ActorPop, Taken, s.User)
	})
	r.Read = fr
	if err != nil {
		return r, err
	}
	if written && queued {
		r.Provisional = true
		r.Assignee = s.User
		return r, s.update(func(st *syncState) {
			st.Provisional = append(st.Provisional, Provisional{ID: id, User: s.User})
		})
	}
	i, err := s.issue(id)
	if err != nil {
		return r, err
	}
	r.Assignee = i.Assignee
	if i.Assignee != s.User {
		r.Lost, r.LostTo = true, i.Assignee
	}
	return r, nil
}

// Close closes an issue on its observed merge, recording the merge commit
// as the closing comment (srd042 R2.3). An issue its assignee merged
// without first marking it started is started on their behalf, so the
// close is still the table's in-progress to closed by merge.
func (s *Store) Close(id, commit string) (CloseResult, error) {
	fr, written, _, err := s.mutate(id, "close", func(i *Issue) (bool, error) {
		if i.State == Closed && i.CloseReason == commit {
			return false, nil
		}
		if i.State == Taken {
			if err := transition(i, ActorPerson, InProgress, s.User); err != nil {
				return false, err
			}
		}
		if err := transition(i, ActorMerge, Closed, s.User); err != nil {
			return false, err
		}
		now := s.stamp()
		i.ClosedAt, i.CloseReason = now, commit
		i.Comments = append(i.Comments, Comment{ID: newID(), IssueID: i.ID, Author: s.User, CreatedAt: now,
			Text: "Closed by merge commit " + commit + "."})
		return true, nil
	})
	return CloseResult{ID: id, Changed: written, Read: fr}, err
}

// Fire makes one srd041 transition as actor on behalf of the store's user,
// or refuses it.
func (s *Store) Fire(id, actor, to string) error {
	_, _, _, err := s.mutate(id, actor+" "+to, func(i *Issue) (bool, error) {
		return true, transition(i, actor, to, s.User)
	})
	return err
}

// Comment appends a comment by the store's user.
func (s *Store) Comment(id, text string) error {
	_, _, _, err := s.mutate(id, "comment", func(i *Issue) (bool, error) {
		i.Comments = append(i.Comments, Comment{ID: newID(), IssueID: i.ID, Author: s.User, CreatedAt: s.stamp(), Text: text})
		return true, nil
	})
	return err
}

// Reconcile is the first synchronization back online: it merges, pushes the
// writes queued offline, and reports each provisional pop as confirmed or
// lost to the taker who pushed first.
func (s *Store) Reconcile() ([]Reconciled, error) {
	fr, err := s.Sync()
	if err != nil {
		return nil, err
	}
	if fr.Offline {
		return nil, fmt.Errorf("the tracker repository is still unreachable; last pulled %s at %s", fr.Commit, fr.At.Format(time.RFC3339))
	}
	if queued, err := s.push(); err != nil {
		return nil, err
	} else if queued {
		return nil, errors.New("the tracker repository became unreachable during reconcile")
	}
	st, err := s.state()
	if err != nil {
		return nil, err
	}
	var out []Reconciled
	for _, p := range st.Provisional {
		i, err := s.issue(p.ID)
		if err != nil {
			return out, err
		}
		outcome := "confirmed"
		if i.Assignee != p.User {
			outcome = "lost to " + i.Assignee
		}
		out = append(out, Reconciled{ID: p.ID, Outcome: outcome})
	}
	return out, s.update(func(st *syncState) { st.Provisional = nil })
}

// Sync pulls the tracker repository into the clone, merging by the eng04
// rules when both have moved. A clone that cannot reach the tracker
// repository is not an error: it reports Offline with the last pull.
func (s *Store) Sync() (Freshness, error) {
	st, err := s.state()
	if err != nil {
		return Freshness{}, err
	}
	if _, err := s.git("fetch", "-q", "origin"); err != nil {
		fr := Freshness{Commit: st.Commit, At: st.At, Offline: true}
		s.last = &fr
		return fr, nil
	}
	remote, err := s.git("rev-parse", "--verify", "-q", "origin/main")
	if err != nil {
		return Freshness{}, fmt.Errorf("the tracker repository has no main branch: %w", err)
	}
	head, err := s.git("rev-parse", "HEAD")
	if err != nil {
		return Freshness{}, err
	}
	base, err := s.git("merge-base", "HEAD", remote)
	if err != nil {
		return Freshness{}, err
	}
	switch {
	case head == remote, base == remote:
	case base == head:
		if _, err := s.git("merge", "-q", "--ff-only", remote); err != nil {
			return Freshness{}, err
		}
	default:
		if err := s.merge(base, remote); err != nil {
			return Freshness{}, err
		}
	}
	at := s.Now()
	if err := s.update(func(st *syncState) { st.Commit, st.At = remote, at }); err != nil {
		return Freshness{}, err
	}
	fr := Freshness{Commit: remote, At: at}
	s.last = &fr
	return fr, nil
}

// merge records a merge with the tracker repository whose content is the
// eng04 merge of the tracker file, not git's line merge.
func (s *Store) merge(base, remote string) error {
	b, err := s.show(base)
	if err != nil {
		return err
	}
	o, err := s.show("HEAD")
	if err != nil {
		return err
	}
	t, err := s.show(remote)
	if err != nil {
		return err
	}
	merged := mergeIssues(b, o, t, s.stamp())
	bm, err := s.showMilestones(base)
	if err != nil {
		return err
	}
	om, err := s.showMilestones("HEAD")
	if err != nil {
		return err
	}
	tm, err := s.showMilestones(remote)
	if err != nil {
		return err
	}
	if _, err := s.git("merge", "-q", "--no-ff", "--no-commit", "-s", "ours", remote); err != nil {
		return err
	}
	if err := writeIssues(filepath.Join(s.Dir, IssuesFile), merged); err != nil {
		return err
	}
	if ms := mergeMilestones(bm, om, tm); len(ms) > 0 {
		if err := writeMilestones(filepath.Join(s.Dir, MilestonesFile), ms); err != nil {
			return err
		}
	}
	if _, err := s.git("add", "-A", "--", ".beads"); err != nil {
		return err
	}
	_, err = s.git("commit", "-q", "-m", "tpm: merge the tracker repository")
	return err
}

// Create files a new issue in repository with the next IID there, and
// returns it as the tracker repository holds it after the push: a create
// that raced another clone's has been renumbered by the merge.
func (s *Store) Create(i Issue) (Issue, error) {
	i.ID = "tpm-" + newID()[:8]
	_, _, _, err := s.write("create "+i.ID, func(issues *[]Issue) (bool, error) {
		for _, x := range *issues {
			if x.Repository == i.Repository {
				i.IID = max(i.IID, x.IID)
			}
		}
		i.IID++
		now := s.stamp()
		i.CreatedAt, i.UpdatedAt, i.CreatedBy = now, now, s.User
		if i.State == "" {
			i.State, i.Status = Open, StatusOpen
		}
		*issues = append(*issues, i)
		return true, nil
	})
	if err != nil {
		return Issue{}, err
	}
	return s.issue(i.ID)
}

// mutate is every write to one issue: pull, apply fn, commit, push.
func (s *Store) mutate(id, what string, fn func(*Issue) (bool, error)) (fr Freshness, written, queued bool, err error) {
	return s.write(what+" "+id, func(issues *[]Issue) (bool, error) {
		for k := range *issues {
			if (*issues)[k].ID == id {
				changed, err := fn(&(*issues)[k])
				if changed && err == nil {
					(*issues)[k].UpdatedAt = s.stamp()
				}
				return changed, err
			}
		}
		return false, fmt.Errorf("issue %s: %w", id, ErrNotFound)
	})
}

// write is every write to the issues: pull, apply fn, commit, push.
func (s *Store) write(what string, fn func(*[]Issue) (bool, error)) (fr Freshness, written, queued bool, err error) {
	return s.writeFiles(what, func() (bool, error) {
		issues, err := s.read()
		if err != nil {
			return false, err
		}
		changed, err := fn(&issues)
		if err != nil || !changed {
			return false, err
		}
		return true, writeIssues(filepath.Join(s.Dir, IssuesFile), issues)
	})
}

// writeFiles is every write: pull, let fn change the tracker files, commit,
// push. A write made offline is committed and queued for the next
// synchronization.
func (s *Store) writeFiles(what string, fn func() (bool, error)) (fr Freshness, written, queued bool, err error) {
	if fr, err = s.Sync(); err != nil {
		return fr, false, false, err
	}
	if s.beforeWrite != nil {
		s.beforeWrite()
	}
	changed, err := fn()
	if err != nil || !changed {
		return fr, false, false, err
	}
	if _, err := s.git("add", "-A", "--", ".beads"); err != nil {
		return fr, false, false, err
	}
	if _, err := s.git("commit", "-q", "-m", fmt.Sprintf("tpm: %s (%s)", what, s.User)); err != nil {
		return fr, false, false, err
	}
	if fr.Offline {
		return fr, true, true, nil
	}
	queued, err = s.push()
	return fr, true, queued, err
}

// push lands the clone's commits in the tracker repository, merging and
// retrying when the repository moved since the last pull. It reports queued
// when the repository turned out to be unreachable.
func (s *Store) push() (queued bool, err error) {
	for range pushAttempts {
		if _, err := s.git("push", "-q", "origin", "HEAD:main"); err == nil {
			return false, nil
		}
		fr, err := s.Sync()
		if err != nil {
			return false, err
		}
		if fr.Offline {
			return true, nil
		}
	}
	return false, fmt.Errorf("push to the tracker repository rejected %d times", pushAttempts)
}

func (s *Store) read() ([]Issue, error) {
	b, err := os.ReadFile(filepath.Join(s.Dir, IssuesFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parseIssues(b)
}

func (s *Store) issue(id string) (Issue, error) {
	issues, err := s.read()
	if err != nil {
		return Issue{}, err
	}
	for _, i := range issues {
		if i.ID == id {
			return i, nil
		}
	}
	return Issue{}, fmt.Errorf("issue %s: %w", id, ErrNotFound)
}

func (s *Store) show(rev string) ([]Issue, error) {
	out, err := s.git("show", rev+":"+IssuesFile)
	if err != nil {
		return nil, nil
	}
	return parseIssues([]byte(out))
}

func (s *Store) stamp() string { return s.Now().UTC().Format(time.RFC3339) }

func (s *Store) statePath() (string, error) {
	dir, err := s.git("rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tpm-beads-sync.json"), nil
}

func (s *Store) state() (syncState, error) {
	var st syncState
	path, err := s.statePath()
	if err != nil {
		return st, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	return st, json.Unmarshal(b, &st)
}

func (s *Store) update(fn func(*syncState)) error {
	st, err := s.state()
	if err != nil {
		return err
	}
	fn(&st)
	path, err := s.statePath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (s *Store) git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = s.Dir
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
