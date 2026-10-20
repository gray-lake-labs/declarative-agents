package program

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/gray-lake-labs/scrum-master"
	"github.com/gray-lake-labs/scrum-master/internal/install"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// Git runs git in a directory and returns its standard output. Tests use a
// recording implementation.
type Git interface {
	Run(dir string, args ...string) (string, error)
}

// ExecGit runs the git on PATH.
type ExecGit struct{}

func (ExecGit) Run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

var slugJunk = regexp.MustCompile(`[^a-z0-9]+`)

// Slug turns an issue title into a branch slug: lowercase words joined by
// hyphens, at most 40 characters.
func Slug(title string) string {
	s := strings.Trim(slugJunk.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if len(s) > 40 {
		s = strings.TrimRight(s[:40], "-")
	}
	if s == "" {
		s = "issue"
	}
	return s
}

// PopResult is what pop reports.
type PopResult struct {
	Project  string `json:"project"`
	IID      int    `json:"iid"`
	Title    string `json:"title"`
	WebURL   string `json:"web_url"`
	Branch   string `json:"branch"`
	Worktree string `json:"worktree"`
	Checkout string `json:"checkout"`
	Assigned bool   `json:"assigned"` // pop assigned the caller now
	// Provisional is an assignment taken while the tracker repository was
	// unreachable; tpm tracker reconcile confirms it or reports it lost.
	Provisional bool `json:"provisional,omitempty"`
	Created     bool `json:"branch_created"` // pop created the branch now
	Added       bool `json:"worktree_added"` // pop added the worktree now
	DryRun      bool `json:"dry_run"`
	// Installed are the agents whose skills were installed into the worktree
	// (srd036 R2.3); InstallError says why that failed, without undoing pop.
	Installed    []string `json:"installed,omitempty"`
	InstallError string   `json:"install_error,omitempty"`
	// NotAdmitted are the reasons a management or control issue fails
	// admission. They do not block pop, which refuses only data issues
	// admission has not admitted (srd013 R7.1).
	NotAdmitted []string `json:"not_admitted,omitempty"`
}

// Pop claims an issue for the caller and opens its own branch and worktree
// (srd013 R7). It refuses a closed issue, an issue assigned to anyone else,
// and a data issue admission does not admit. Everything it finds already in
// place is reused, so a second pop changes nothing. dir is any directory in
// the local checkout of the project.
func Pop(t tracker.Tracker, git Git, project string, iid int, dir string, p Policy, dryRun bool) (PopResult, error) {
	i, err := t.GetIssue(project, iid)
	if err != nil {
		return PopResult{}, err
	}
	r := PopResult{Project: project, IID: iid, Title: i.Title, WebURL: i.WebURL, DryRun: dryRun}
	if i.State == "closed" {
		return r, fmt.Errorf("%s#%d is closed", project, iid)
	}
	me, err := t.CurrentUser()
	if err != nil {
		return r, err
	}
	for _, a := range i.Assignees {
		if a != me.Username {
			return r, fmt.Errorf("%s#%d is assigned to %s; only that person pops it", project, iid, a)
		}
	}
	if reasons := Lint(i, p); len(reasons) > 0 {
		if i.HasLabel("plane::data") {
			return r, fmt.Errorf("%s#%d is not admitted: %s", project, iid, strings.Join(reasons, "; "))
		}
		r.NotAdmitted = reasons
	}
	if !slices.Contains(i.Assignees, me.Username) {
		r.Assigned = true
		if !dryRun {
			if err := t.AssignIssue(project, iid, me); err != nil {
				return r, err
			}
			if reader, ok := t.(tracker.Reader); ok {
				read, _ := reader.LastRead()
				r.Provisional = read.Offline
			}
		}
	}

	r.Branch = fmt.Sprintf("%d-%s", iid, Slug(i.Title))
	base, err := t.DefaultBranch(project)
	if err != nil {
		return r, err
	}
	exists, err := t.BranchExists(project, r.Branch)
	if err != nil {
		return r, err
	}
	if !exists {
		r.Created = true
		if !dryRun {
			if err := t.CreateBranch(project, r.Branch, base); err != nil {
				return r, err
			}
		}
	}

	top, err := git.Run(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return r, fmt.Errorf("pop needs a local checkout of %s: %w", project, err)
	}
	root := strings.TrimSpace(top)
	r.Checkout = root
	r.Worktree = filepath.Join(filepath.Dir(root), filepath.Base(root)+"-"+r.Branch)
	if _, err := os.Stat(r.Worktree); err == nil {
		return r, nil
	}
	r.Added = true
	if dryRun {
		return r, nil
	}
	if _, err := git.Run(root, "fetch", "origin", r.Branch); err != nil {
		return r, err
	}
	if _, err := git.Run(root, "rev-parse", "--verify", "--quiet", "refs/heads/"+r.Branch); err == nil {
		_, err = git.Run(root, "worktree", "add", r.Worktree, r.Branch)
		return r, err
	}
	_, err = git.Run(root, "worktree", "add", "--track", "-b", r.Branch, r.Worktree, "origin/"+r.Branch)
	return r, err
}

// InstallSkills installs into worktree the skills of every agent installed
// in checkout (srd036 R2.3). The installed files are untracked, so a new
// worktree has none, and an agent started there could not finish the issue.
// A checkout with no installed agent installs nothing.
func InstallSkills(checkout, worktree string) ([]string, error) {
	agents := install.InstalledAgents(checkout)
	if len(agents) == 0 {
		return nil, nil
	}
	skills, err := install.Skills(scrummaster.Plugin, scrummaster.SkillsDir)
	if err != nil {
		return nil, err
	}
	if _, err := install.Install(worktree, agents, skills, install.Version(), false, false); err != nil {
		return nil, err
	}
	return agents, nil
}
