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

// IssuesDir holds one file per issue, <id>.json, in a tracker whose config
// sets layout: files. A write rewrites only the issues it changes, so its
// cost does not grow with the tracker, and writers of different issues
// change different files, which git merges by itself (eng04).
const IssuesDir = ".beads/issues"

// LayoutFiles is the config's layout value for one file per issue. Without
// it the tracker keeps every issue in IssuesFile, bd's JSONL.
const LayoutFiles = "files"

// files reports whether the clone's tracker keeps one file per issue.
func (s *Store) files() bool {
	c, err := s.config()
	return err == nil && c.Layout == LayoutFiles
}

func issuePath(dir, id string) string { return filepath.Join(dir, IssuesDir, id+".json") }

// encodeIssue is one issue as its file holds it: one line of JSON, so that
// two writers changing the same issue always conflict in git and the merge
// goes through mergeIssue, never through git's line merge.
func encodeIssue(i Issue) ([]byte, error) {
	if i.Type == "" {
		i.Type = "issue"
	}
	b, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func decodeIssue(path string, b []byte) (Issue, error) {
	var i Issue
	if err := json.Unmarshal(bytes.TrimSpace(b), &i); err != nil {
		return i, fmt.Errorf("%s: %w", path, err)
	}
	return i, nil
}

// readIssueDir reads every issue file under dir, in ID order.
func readIssueDir(dir string) ([]Issue, error) {
	entries, err := os.ReadDir(filepath.Join(dir, IssuesDir))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var issues []Issue
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, IssuesDir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		i, err := decodeIssue(path, b)
		if err != nil {
			return nil, err
		}
		issues = append(issues, i)
	}
	return issues, nil
}

// topIID is the highest IID among dir's issues in repository, reading only
// each issue's repository and IID.
func topIID(dir, repository string) (int, error) {
	entries, err := os.ReadDir(filepath.Join(dir, IssuesDir))
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	top := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, IssuesDir, e.Name()))
		if err != nil {
			return 0, err
		}
		var head struct {
			Repository string `json:"repository"`
			IID        int    `json:"iid"`
		}
		if err := json.Unmarshal(b, &head); err != nil {
			return 0, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if head.Repository == repository {
			top = max(top, head.IID)
		}
	}
	return top, nil
}

// encodedAll is each issue's file content by ID, what a write compares
// against to rewrite only the issues it changed.
func encodedAll(issues []Issue) (map[string][]byte, error) {
	out := make(map[string][]byte, len(issues))
	for _, i := range issues {
		b, err := encodeIssue(i)
		if err != nil {
			return nil, err
		}
		out[i.ID] = b
	}
	return out, nil
}

// writeChanged writes the files of the issues whose content differs from
// before, and reports whether it wrote any.
func writeChanged(dir string, before map[string][]byte, issues []Issue) (bool, error) {
	if err := os.MkdirAll(filepath.Join(dir, IssuesDir), 0o755); err != nil {
		return false, err
	}
	wrote := false
	for _, i := range issues {
		b, err := encodeIssue(i)
		if err != nil {
			return false, err
		}
		if bytes.Equal(before[i.ID], b) {
			continue
		}
		if err := os.WriteFile(issuePath(dir, i.ID), b, 0o644); err != nil {
			return false, err
		}
		wrote = true
	}
	return wrote, nil
}

// mergeFiles records a merge with the tracker repository in the files
// layout. git merges the issues only one side changed; an issue both sides
// changed conflicts, as its file is one line, and merges by mergeIssue; an
// issue each side created with the same IID in the same repository is
// renumbered on ours, as mergeIssues does; the record files merge by their
// own rules (eng04).
func (s *Store) mergeFiles(base, remote string) error {
	if _, err := s.git("merge", "-q", "--no-ff", "--no-commit", remote); err != nil {
		if _, herr := s.git("rev-parse", "-q", "--verify", "MERGE_HEAD"); herr != nil {
			return err
		}
	}
	conflicted, err := s.git("diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return err
	}
	for _, p := range strings.Fields(conflicted) {
		if !strings.HasPrefix(p, IssuesDir+"/") {
			continue // a record file, merged below
		}
		var sides [3]Issue
		for n := range sides {
			out, err := s.git("show", fmt.Sprintf(":%d:%s", n+1, p))
			if err != nil {
				continue // the side has no version: an issue new on the others
			}
			if sides[n], err = decodeIssue(p, []byte(out)); err != nil {
				return err
			}
		}
		b, err := encodeIssue(mergeIssue(sides[0], sides[1], sides[2], s.stamp()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(s.Dir, p), b, 0o644); err != nil {
			return err
		}
	}
	if err := s.renumberNew(base, remote); err != nil {
		return err
	}
	if err := mergeRecordFile[Milestone](s, MilestonesFile, base, remote); err != nil {
		return err
	}
	if err := mergeRecordFile[Label](s, LabelsFile, base, remote); err != nil {
		return err
	}
	if _, err := s.git("add", "-A", "--", ".beads"); err != nil {
		return err
	}
	_, err = s.git("commit", "-q", "-m", "tpm: merge the tracker repository")
	return err
}

// renumberNew gives each issue ours created since base that collides with
// one theirs created the next free IID in its repository. Only issues
// created on both sides can collide: an existing issue's IID is at most
// base's highest, and a new one is above it.
func (s *Store) renumberNew(base, remote string) error {
	added := func(rev string) ([]string, error) {
		out, err := s.git("diff", "--name-only", "--diff-filter=A", base, rev, "--", IssuesDir)
		return strings.Fields(out), err
	}
	ours, err := added("HEAD")
	if err != nil || len(ours) == 0 {
		return err
	}
	theirs, err := added(remote)
	if err != nil || len(theirs) == 0 {
		return err
	}
	var issues []Issue
	for _, p := range append(slices.Clone(theirs), ours...) {
		b, err := os.ReadFile(filepath.Join(s.Dir, p))
		if err != nil {
			return err
		}
		i, err := decodeIssue(p, b)
		if err != nil {
			return err
		}
		issues = append(issues, i)
	}
	before, err := encodedAll(issues)
	if err != nil {
		return err
	}
	oursNew := map[string]bool{}
	for _, p := range ours {
		oursNew[strings.TrimSuffix(filepath.Base(p), ".json")] = true
	}
	renumber(issues, func(id string) bool { return oursNew[id] })
	_, err = writeChanged(s.Dir, before, issues)
	return err
}

// MigrateResult is what tpm tracker migrate did.
type MigrateResult struct {
	Issues  int    `json:"issues"`
	Commit  string `json:"tracker_commit,omitempty"`
	Already bool   `json:"already"`
}

// Migrate moves a tracker from IssuesFile to one file per issue, in one
// commit that also sets the config's layout. It is meant to run once, when
// no one is writing: a tracker that moved while it ran is left as it was,
// and the command asks to be run again.
func (s *Store) Migrate() (MigrateResult, error) {
	var r MigrateResult
	fr, err := s.Sync()
	if err != nil {
		return r, err
	}
	if fr.Offline {
		return r, errors.New("the tracker repository is unreachable; migrate needs it")
	}
	if s.files() {
		r.Already = true
		return r, nil
	}
	issues, err := s.read()
	if err != nil {
		return r, err
	}
	before, err := s.git("rev-parse", "HEAD")
	if err != nil {
		return r, err
	}
	if _, err := writeChanged(s.Dir, nil, issues); err != nil {
		return r, err
	}
	if err := os.Remove(filepath.Join(s.Dir, IssuesFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return r, err
	}
	c, err := s.config()
	if err != nil {
		return r, err
	}
	c.Layout = LayoutFiles
	if err := WriteConfig(s.Dir, c); err != nil {
		return r, err
	}
	if _, err := s.git("add", "-A", "--", ".beads"); err != nil {
		return r, err
	}
	if _, err := s.git("commit", "-q", "-m", fmt.Sprintf("tpm: migrate the tracker to one file per issue, %d issues (%s)", len(issues), s.User)); err != nil {
		return r, err
	}
	if _, err := s.git("push", "-q", "origin", "HEAD:main"); err != nil {
		if _, rerr := s.git("reset", "-q", "--hard", before); rerr != nil {
			return r, errors.Join(err, rerr)
		}
		return r, fmt.Errorf("the tracker repository moved while migrating, so nothing was written; run tpm tracker migrate again when no one is writing: %w", err)
	}
	r.Issues = len(issues)
	r.Commit, err = s.git("rev-parse", "HEAD")
	return r, err
}

// Export writes every issue as bd's JSONL, one line per issue sorted by
// ID, whichever layout the tracker keeps, so bd can import a tracker that
// keeps one file per issue.
func (s *Store) Export() ([]byte, error) {
	if _, err := s.Sync(); err != nil {
		return nil, err
	}
	issues, err := s.read()
	if err != nil {
		return nil, err
	}
	return encodeJSONL(issues)
}
