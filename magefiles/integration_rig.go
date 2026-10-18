package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	gl "gitlab.com/gitlab-org/api/client-go"

	"github.com/gray-lake-labs/scrum-master/magefiles/gitlabrig"
)

// seedBranch holds the fixtures in rig repositories: main is protected
// against pushes, as in production.
const seedBranch = "seed"

// rigEnv is what an integration target needs from the current rig run.
type rigEnv struct {
	root    string
	out     gitlabrig.Outputs
	cfg     gitlabrig.Config
	token   string
	api     *gl.Client
	tpm     string
	webBase string
}

func newRigEnv() (*rigEnv, error) {
	root, err := repositoryRoot()
	if err != nil {
		return nil, err
	}
	out, err := gitlabrig.LoadOutputs(root)
	if err != nil {
		return nil, err
	}
	cfg, err := gitlabrig.LoadConfig(root)
	if err != nil {
		return nil, err
	}
	token, err := gitlabrig.Token(gitlabrig.ExecRunner{}, cfg.Host())
	if err != nil {
		return nil, err
	}
	api, err := gl.NewClient(token, gitlab.ClientOptions(cfg.APIURL(), 60*time.Second)...)
	if err != nil {
		return nil, err
	}
	bin := filepath.Join(root, gitlabrig.StateDir, out.RunID, "tpm")
	if err := runStreaming(root, "go", "build", "-o", bin, "./cmd/tpm"); err != nil {
		return nil, fmt.Errorf("build tpm: %w", err)
	}
	return &rigEnv{root: root, out: out, cfg: cfg, token: token, api: api, tpm: bin, webBase: cfg.BaseURL}, nil
}

func (r *rigEnv) project(key string) (string, error) {
	p, ok := r.out.Projects[key]
	if !ok {
		return "", fmt.Errorf("rig has no %s repository; the suite needs it", key)
	}
	return p.PathWithNamespace, nil
}

// runTPM runs the built tpm against the rig's instance and returns stdout.
func (r *rigEnv) runTPM(args ...string) ([]byte, error) {
	cmd := exec.Command(r.tpm, args...)
	cmd.Env = append(os.Environ(), "GITLAB_TOKEN="+r.token, "GITLAB_BASE_URL="+r.cfg.APIURL())
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return out.Bytes(), fmt.Errorf("tpm %s: %w: %s", strings.Join(args, " "), err, errb.String())
	}
	return out.Bytes(), nil
}

func (r *rigEnv) tpmJSON(into any, args ...string) error {
	out, err := r.runTPM(append(args, "--json")...)
	if err != nil {
		return err
	}
	return json.Unmarshal(out, into)
}

// seed creates the seed branch from main and commits files to it, once.
func (r *rigEnv) seed(project string, files map[string]string) error {
	if _, resp, err := r.api.Branches.GetBranch(project, seedBranch); err != nil {
		if resp == nil || resp.StatusCode != http.StatusNotFound {
			return err
		}
		if _, _, err := r.api.Branches.CreateBranch(project, &gl.CreateBranchOptions{Branch: gl.Ptr(seedBranch), Ref: gl.Ptr("main")}); err != nil {
			return fmt.Errorf("create %s in %s: %w", seedBranch, project, err)
		}
	}
	var actions []*gl.CommitActionOptions
	for path, content := range files {
		action := gl.FileCreate
		if _, resp, err := r.api.RepositoryFiles.GetFileMetaData(project, path, &gl.GetFileMetaDataOptions{Ref: gl.Ptr(seedBranch)}); err == nil {
			action = gl.FileUpdate
		} else if resp == nil || resp.StatusCode != http.StatusNotFound {
			return err
		}
		actions = append(actions, &gl.CommitActionOptions{Action: gl.Ptr(action), FilePath: gl.Ptr(path), Content: gl.Ptr(content)})
	}
	_, _, err := r.api.Commits.CreateCommit(project, &gl.CreateCommitOptions{Branch: gl.Ptr(seedBranch), CommitMessage: gl.Ptr("Seed regression fixtures"), Actions: actions})
	return err
}

func (r *rigEnv) tag(project, tag string) error {
	if _, _, err := r.api.Tags.GetTag(project, tag); err == nil {
		return nil
	}
	_, _, err := r.api.Tags.CreateTag(project, &gl.CreateTagOptions{TagName: gl.Ptr(tag), Ref: gl.Ptr(seedBranch)})
	return err
}

// milestone ensures a group milestone on the rig group, started yesterday,
// so it is the active sprint tpm program file defaults to.
func (r *rigEnv) milestone(title, due string) error {
	ms, _, err := r.api.GroupMilestones.ListGroupMilestones(r.out.Group.FullPath, &gl.ListGroupMilestonesOptions{Title: gl.Ptr(title)})
	if err != nil || len(ms) > 0 {
		return err
	}
	d, _ := time.Parse("2006-01-02", due)
	iso := gl.ISOTime(d)
	start := gl.ISOTime(time.Now().AddDate(0, 0, -1))
	_, _, err = r.api.GroupMilestones.CreateGroupMilestone(r.out.Group.FullPath, &gl.CreateGroupMilestoneOptions{Title: gl.Ptr(title), StartDate: &start, DueDate: &iso})
	return err
}

func (r *rigEnv) issues(project string) ([]*gl.Issue, error) {
	is, _, err := r.api.Issues.ListProjectIssues(project, &gl.ListProjectIssuesOptions{State: gl.Ptr("opened")})
	return is, err
}

func (r *rigEnv) notes(project string, iid int64) ([]*gl.Note, error) {
	ns, _, err := r.api.Notes.ListIssueNotes(project, iid, &gl.ListIssueNotesOptions{})
	var out []*gl.Note
	for _, n := range ns {
		if !n.System {
			out = append(out, n)
		}
	}
	return out, err
}

// check collects failed expectations so a target reports all of them.
type check struct{ failures []string }

func (c *check) expect(ok bool, format string, args ...any) {
	if !ok {
		c.failures = append(c.failures, fmt.Sprintf(format, args...))
	}
}

func (c *check) err(name string) error {
	if len(c.failures) == 0 {
		fmt.Printf("integration:%s passed\n", name)
		return nil
	}
	return fmt.Errorf("integration:%s failed:\n  %s", name, strings.Join(c.failures, "\n  "))
}
