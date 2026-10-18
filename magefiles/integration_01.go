package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	gl "gitlab.com/gitlab-org/api/client-go"

	"github.com/gray-lake-labs/scrum-master/internal/program"
)

func init() {
	integrationTargets = append(integrationTargets, Integration{}.Pop)
}

// Pop files an admitted issue in po-1, pops it into a worktree of a local clone, proves the refusals, commits and pushes, opens the merge request, merges it, and proves the issue closed (test-rel00.1-pop).
func (Integration) Pop() error {
	r, err := newRigEnv()
	if err != nil {
		return err
	}
	po1, err := r.project("product-owners/po-1")
	if err != nil {
		return err
	}
	if err := r.milestone(rigMilestone, rigDue); err != nil {
		return err
	}
	if _, err := r.runTPM("program", "labels", "--group", r.out.Group.FullPath); err != nil {
		return err
	}
	c := &check{}
	run := filepath.Join(r.root, ".regression", r.out.RunID)
	good := "## Requirements\n\n1. Add the grader notes.\n\n## Acceptance Criteria\n\n- The notes exist (test: notes_exist; environment: local)\n"
	bad := "## Requirements\n\n1. Something.\n"
	file := func(title, body string) (program.FileResult, error) {
		path := filepath.Join(run, "pop-"+program.Slug(title)+".md")
		writeFileTo(path, body)
		var res program.FileResult
		return res, r.tpmJSON(&res, "program", "file", "--project", po1, "--title", title, "--body-file", path,
			"--plane", "data", "--kind", "build", "--component", "po-1", "--milestone", rigMilestone)
	}

	work, err := file("Add grader notes", good)
	if err != nil {
		return err
	}
	clone, err := r.clone(po1, "po-1")
	if err != nil {
		return err
	}
	if _, err := r.runTPM("install", "--agent", "claude", "--dir", clone); err != nil {
		return err
	}

	var popped program.PopResult
	if err := r.tpmJSON(&popped, "program", "pop", "--project", po1, "--issue", fmt.Sprint(work.IID), "--dir", clone); err != nil {
		return err
	}
	me, _, err := r.api.Users.CurrentUser()
	if err != nil {
		return err
	}
	i, _, err := r.api.Issues.GetIssue(po1, int64(work.IID))
	if err != nil {
		return err
	}
	c.expect(len(i.Assignees) == 1 && i.Assignees[0].Username == me.Username, "assignees after pop: %d", len(i.Assignees))
	_, _, berr := r.api.Branches.GetBranch(po1, popped.Branch)
	c.expect(berr == nil, "branch %s not on the remote: %v", popped.Branch, berr)
	_, serr := os.Stat(popped.Worktree)
	c.expect(serr == nil && strings.HasPrefix(popped.Worktree, run), "worktree %s missing or outside the run directory", popped.Worktree)
	_, ierr := os.Stat(filepath.Join(popped.Worktree, ".claude/skills/issue-pop/SKILL.md"))
	c.expect(slices.Equal(popped.Installed, []string{"claude"}) && ierr == nil, "skills not installed in the worktree: %v %q %v", popped.Installed, popped.InstallError, ierr)
	status, _ := exec.Command("git", "-C", popped.Worktree, "status", "--porcelain", "--untracked-files=all").Output()
	c.expect(len(status) == 0, "worktree not clean after the install: %q", status)

	var again program.PopResult
	if err := r.tpmJSON(&again, "program", "pop", "--project", po1, "--issue", fmt.Sprint(work.IID), "--dir", clone); err != nil {
		return err
	}
	c.expect(!again.Assigned && !again.Created && !again.Added, "second pop changed something: %+v", again)

	closed, err := file("Already closed", good)
	if err != nil {
		return err
	}
	if _, _, err := r.api.Issues.UpdateIssue(po1, int64(closed.IID), &gl.UpdateIssueOptions{StateEvent: gl.Ptr("close")}); err != nil {
		return err
	}
	_, err = r.runTPM("program", "pop", "--project", po1, "--issue", fmt.Sprint(closed.IID), "--dir", clone)
	c.expect(err != nil && strings.Contains(err.Error(), "is closed"), "pop of a closed issue: %v", err)
	triaged, err := file("No acceptance criteria", bad)
	if err != nil {
		return err
	}
	_, err = r.runTPM("program", "pop", "--project", po1, "--issue", fmt.Sprint(triaged.IID), "--dir", clone)
	c.expect(err != nil && strings.Contains(err.Error(), "not admitted"), "pop of a needs-triage issue: %v", err)

	writeFileTo(filepath.Join(popped.Worktree, "GRADER.md"), "# Grader notes\n")
	for _, args := range [][]string{{"add", "GRADER.md"}, {"commit", "--quiet", "-m", "Add grader notes"}, {"push", "--quiet"}} {
		if err := r.git(popped.Worktree, args...); err != nil {
			return err
		}
	}
	var opened program.OpenMergeRequestResult
	if err := r.tpmJSON(&opened, "program", "open-merge-request", "--project", po1, "--issue", fmt.Sprint(work.IID), "--branch", popped.Branch); err != nil {
		return err
	}
	c.expect(opened.Created && opened.MR > 0, "open-merge-request: %+v", opened)
	if err := r.merge(po1, opened.MR); err != nil {
		return err
	}
	c.expect(r.waitClosed(po1, work.IID), "issue #%d still open after its merge request merged", work.IID)
	return c.err("pop")
}

// clone clones a rig repository into the run directory, with a committer
// identity set, and returns its path.
func (r *rigEnv) clone(project, name string) (string, error) {
	run := filepath.Join(r.root, ".regression", r.out.RunID)
	dir := filepath.Join(run, name)
	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	}
	remote := strings.Replace(r.webBase, "https://", "https://oauth2:"+r.token+"@", 1) + "/" + project + ".git"
	if err := r.git(run, "clone", "--quiet", remote, dir); err != nil {
		return "", fmt.Errorf("clone %s: %w", project, err)
	}
	for _, kv := range [][2]string{{"user.name", "Regression Rig"}, {"user.email", "rig@example.invalid"}} {
		if err := r.git(dir, "config", kv[0], kv[1]); err != nil {
			return "", err
		}
	}
	return dir, nil
}

// git runs git for the rig; the token in a clone URL never reaches an
// error message.
func (r *rigEnv) git(dir string, args ...string) error {
	_, err := program.ExecGit{}.Run(dir, args...)
	if err != nil && r.token != "" {
		return fmt.Errorf("%s", strings.ReplaceAll(err.Error(), r.token, "<token>"))
	}
	return err
}

// merge accepts a merge request, retrying while GitLab still checks it.
func (r *rigEnv) merge(project string, iid int) error {
	var last error
	for attempt := 0; attempt < 20; attempt++ {
		_, _, err := r.api.MergeRequests.AcceptMergeRequest(project, int64(iid), nil)
		if err == nil {
			return nil
		}
		last = err
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("merge !%d in %s: %w", iid, project, last)
}

// waitClosed polls the issue until GitLab closes it from the merged
// merge request's Closes line.
func (r *rigEnv) waitClosed(project string, iid int) bool {
	for attempt := 0; attempt < 20; attempt++ {
		i, _, err := r.api.Issues.GetIssue(project, int64(iid))
		if err == nil && i.State == "closed" {
			return true
		}
		time.Sleep(3 * time.Second)
	}
	return false
}
