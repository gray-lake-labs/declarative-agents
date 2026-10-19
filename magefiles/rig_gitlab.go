package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/beads"
)

// The opt-in GitLab rig (docs/engineering/eng01-regression-rig.md): a
// disposable group per run, provisioned by the Terraform root
// infra/regression, against which beads runs under real concurrency. It
// never joins integrationTargets, so mage regression stays offline.

const (
	gitlabRoot      = "infra/regression"
	gitlabRunPrefix = "gitlab-"
	gitlabRunFile   = "run.json"
	gitlabWorkers   = 4
)

// gitlabRun is one run's record, kept beside its Terraform state.
type gitlabRun struct {
	ID    string `json:"run_id"`
	Group string `json:"group"`
	Dir   string `json:"dir"`
}

// gitlabRig is what infra/regression's rig output names.
type gitlabRig struct {
	Group struct {
		FullPath string `json:"full_path"`
	} `json:"group"`
	Projects map[string]struct {
		Path string `json:"path_with_namespace"`
	} `json:"projects"`
}

// gitlabEnv is a provisioned run: the rig, the built tpm, and how git
// reaches the instance.
type gitlabEnv struct {
	run      gitlabRun
	rig      gitlabRig
	tpm      string
	base     string   // web root, https://gitlab.com
	auth     []string // git environment carrying the token as a header, never in a file
	tracker  string   // the tracker repository's clone URL
	projects []string // code projects by key, sorted: system, reports, product-owners/po-N
	// busy holds each worker's lock: a person at one machine runs one
	// command at a time against their one clone of the tracker.
	busy sync.Map
}

// Gitlab runs the opt-in GitLab rig: it provisions a disposable group, runs
// the concurrent workload against it, and destroys the group, including
// when the workload fails. Without GITLAB_TOKEN it skips.
func (Regression) Gitlab() error {
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		fmt.Println("regression:gitlab skipped: set GITLAB_TOKEN and GITLAB_RIG_GROUP to run it (eng01-regression-rig)")
		return nil
	}
	group := os.Getenv("GITLAB_RIG_GROUP")
	if group == "" {
		return errors.New("regression:gitlab needs GITLAB_RIG_GROUP, the existing group each run creates its own group under")
	}
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	id := time.Now().UTC().Format("20060102-150405")
	run := gitlabRun{ID: id, Group: group, Dir: filepath.Join(root, rigStateDir, gitlabRunPrefix+id)}
	if err := os.MkdirAll(run.Dir, 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(run, "", "  ")
	if err := os.WriteFile(filepath.Join(run.Dir, gitlabRunFile), b, 0o644); err != nil {
		return err
	}
	fmt.Printf("regression:gitlab run %s under %s\n", id, group)
	err = func() error {
		if err := terraform(root, run, "apply", "-auto-approve"); err != nil {
			return fmt.Errorf("provision: %w", err)
		}
		env, err := newGitlabEnv(root, run, token)
		if err != nil {
			return err
		}
		return gitlabChecks(env)
	}()
	if derr := destroyGitlabRun(root, run); derr != nil {
		return errors.Join(err, fmt.Errorf("destroy run %s: %w; run mage regression:gitlabSweep", id, derr))
	}
	if err != nil {
		return fmt.Errorf("regression:gitlab failed: %w", err)
	}
	fmt.Println("regression:gitlab passed")
	return nil
}

// GitlabSweep destroys every GitLab rig run an interrupted regression left
// behind under .regression.
func (Regression) GitlabSweep() error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	dirs, _ := filepath.Glob(filepath.Join(root, rigStateDir, gitlabRunPrefix+"*"))
	var failed []error
	for _, dir := range dirs {
		var run gitlabRun
		b, err := os.ReadFile(filepath.Join(dir, gitlabRunFile))
		if err != nil || json.Unmarshal(b, &run) != nil {
			continue
		}
		if err := destroyGitlabRun(root, run); err != nil {
			failed = append(failed, fmt.Errorf("%s: %w", run.ID, err))
			continue
		}
		fmt.Println("destroyed", run.ID)
	}
	return errors.Join(failed...)
}

// destroyGitlabRun destroys a run's group and, once that succeeds, removes
// the run's directory with its state.
func destroyGitlabRun(root string, run gitlabRun) error {
	if _, err := os.Stat(filepath.Join(run.Dir, "terraform.tfstate")); err != nil {
		return os.RemoveAll(run.Dir)
	}
	if err := terraform(root, run, "destroy", "-auto-approve"); err != nil {
		return err
	}
	return os.RemoveAll(run.Dir)
}

// terraform runs one command on infra/regression with the run's own state
// and data directory, so nothing lands in the repository.
func terraform(root string, run gitlabRun, command string, args ...string) error {
	env := append(os.Environ(), "TF_DATA_DIR="+filepath.Join(run.Dir, ".terraform"), "TF_IN_AUTOMATION=1")
	tf := func(args ...string) error {
		cmd := exec.Command("terraform", append([]string{"-chdir=" + filepath.Join(root, gitlabRoot)}, args...)...)
		cmd.Env, cmd.Stdout, cmd.Stderr = env, os.Stdout, os.Stderr
		return cmd.Run()
	}
	if err := tf("init", "-input=false", "-backend-config=path="+filepath.Join(run.Dir, "terraform.tfstate")); err != nil {
		return err
	}
	return tf(append(append([]string{command, "-input=false"}, args...), "-var", "group="+run.Group, "-var", "run_id="+run.ID)...)
}

// newGitlabEnv reads the provisioned rig, builds tpm, seeds the tracker's
// first commit and the program's labels, and installs the skills.
func newGitlabEnv(root string, run gitlabRun, token string) (*gitlabEnv, error) {
	cmd := exec.Command("terraform", "-chdir="+filepath.Join(root, gitlabRoot), "output", "-json", "rig")
	cmd.Env = append(os.Environ(), "TF_DATA_DIR="+filepath.Join(run.Dir, ".terraform"))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("read the rig output: %w", err)
	}
	env := &gitlabEnv{run: run, base: gitlabWebBase(), tpm: filepath.Join(run.Dir, "tpm")}
	if err := json.Unmarshal(out, &env.rig); err != nil {
		return nil, err
	}
	header := "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("oauth2:"+token))
	env.auth = []string{"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=http.extraHeader", "GIT_CONFIG_VALUE_0=" + header, "GIT_TERMINAL_PROMPT=0"}
	for key, p := range env.rig.Projects {
		if key == "tracker" {
			env.tracker = env.url(p.Path)
			continue
		}
		env.projects = append(env.projects, key)
	}
	sort.Strings(env.projects)
	if env.tracker == "" {
		return nil, errors.New("the rig has no tracker project")
	}
	if err := runStreaming(root, "go", "build", "-o", env.tpm, "./cmd/tpm"); err != nil {
		return nil, fmt.Errorf("build tpm: %w", err)
	}
	seed := filepath.Join(run.Dir, "seed")
	if err := env.clone(env.tracker, seed); err != nil {
		return nil, err
	}
	writeFileTo(filepath.Join(seed, beads.IssuesFile), "")
	for _, args := range [][]string{
		{"-c", "user.name=operator", "-c", "user.email=operator@example.invalid", "add", beads.IssuesFile},
		{"-c", "user.name=operator", "-c", "user.email=operator@example.invalid", "commit", "-q", "-m", "regression:gitlab: the tracker's first commit"},
		{"push", "-q", "origin", "HEAD:main"},
	} {
		if err := env.git(seed, args...); err != nil {
			return nil, err
		}
	}
	w, err := env.worker(0)
	if err != nil {
		return nil, err
	}
	if _, err := w.tpm("issue", "labels", "--group", env.rig.Group.FullPath); err != nil {
		return nil, err
	}
	return env, nil
}

// gitlabWebBase is the instance's web root: GITLAB_BASE_URL less its API
// path, or gitlab.com.
func gitlabWebBase() string {
	if b := strings.TrimSpace(os.Getenv("GITLAB_BASE_URL")); b != "" {
		return strings.TrimSuffix(strings.TrimSuffix(b, "/"), "/api/v4")
	}
	return "https://gitlab.com"
}

func (e *gitlabEnv) url(project string) string { return e.base + "/" + project + ".git" }

// projectPath is a rig project's path on the instance, by its key.
func (e *gitlabEnv) projectPath(key string) string { return e.rig.Projects[key].Path }

func (e *gitlabEnv) git(dir string, args ...string) error {
	_, err := e.gitOut(dir, args...)
	return err
}

// clone clones a rig repository, retrying while a project the run just
// created is not ready to serve yet.
func (e *gitlabEnv) clone(url, dir string) error {
	var err error
	for attempt := 1; attempt <= 6; attempt++ {
		if err = e.git(filepath.Dir(dir), "clone", "-q", url, dir); err == nil {
			return nil
		}
		os.RemoveAll(dir)
		time.Sleep(time.Duration(attempt) * 3 * time.Second)
	}
	return err
}

func (e *gitlabEnv) gitOut(dir string, args ...string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	cmd := exec.Command("git", args...)
	cmd.Dir, cmd.Env = dir, append(os.Environ(), e.auth...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// gitlabWorker is one person at one machine: their own clone of the tracker
// and their own checkouts, side by side with the work checkout tpm runs in.
type gitlabWorker struct {
	env  *gitlabEnv
	user string
	dir  string
}

// worker sets up worker i the first time and returns it: a tracker clone,
// a checkout of every code project with the skills installed, and work/.
func (e *gitlabEnv) worker(i int) (*gitlabWorker, error) {
	w := &gitlabWorker{env: e, user: fmt.Sprintf("worker%d", i), dir: filepath.Join(e.run.Dir, fmt.Sprintf("w%d", i))}
	if _, err := os.Stat(w.dir); err == nil {
		return w, nil
	}
	work := w.checkout("work")
	if err := e.git(work, "init", "-q", "-b", "main"); err != nil {
		return nil, err
	}
	for _, key := range e.projects {
		dir := w.checkout(path.Base(key))
		if err := e.clone(e.url(e.projectPath(key)), dir); err != nil {
			return nil, err
		}
		for _, kv := range [][2]string{{"user.name", w.user}, {"user.email", w.user + "@example.invalid"}} {
			if err := e.git(dir, "config", kv[0], kv[1]); err != nil {
				return nil, err
			}
		}
		if _, err := w.tpm("install", "--agent", "claude", "--dir", dir); err != nil {
			return nil, err
		}
	}
	return w, nil
}

func (w *gitlabWorker) checkout(name string) string { return filepath.Join(w.dir, "checkouts", name) }

func (w *gitlabWorker) trackerClone() string { return filepath.Join(w.dir, "tracker") }

// tpm runs the built tpm as this worker, from its work checkout, against
// its own clone of the tracker, one command at a time.
func (w *gitlabWorker) tpm(args ...string) ([]byte, error) {
	lock, _ := w.env.busy.LoadOrStore(w.user, &sync.Mutex{})
	lock.(*sync.Mutex).Lock()
	defer lock.(*sync.Mutex).Unlock()
	cmd := exec.Command(w.env.tpm, args...)
	cmd.Dir = w.checkout("work")
	cmd.Env = append(append(os.Environ(), w.env.auth...),
		"TPM_TRACKER="+w.env.tracker, "TPM_TRACKER_CLONE="+w.trackerClone(),
		"TPM_USER="+w.user, "TPM_WEB_BASE="+w.env.base,
		"GIT_AUTHOR_NAME="+w.user, "GIT_AUTHOR_EMAIL="+w.user+"@example.invalid",
		"GIT_COMMITTER_NAME="+w.user, "GIT_COMMITTER_EMAIL="+w.user+"@example.invalid")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return out.Bytes(), fmt.Errorf("%s: tpm %s: %w: %s", w.user, strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return out.Bytes(), nil
}

func (w *gitlabWorker) tpmJSON(into any, args ...string) error {
	out, err := w.tpm(append(args, "--json")...)
	if err != nil {
		return err
	}
	return json.Unmarshal(out, into)
}

// gitlabChecks proves the run provisioned what the workload needs: every
// project cloned with the skills installed, the tracker seeded, and the
// labels created.
func gitlabChecks(env *gitlabEnv) error {
	c := &check{}
	w, err := env.worker(0)
	if err != nil {
		return err
	}
	for _, key := range env.projects {
		_, err := os.Stat(filepath.Join(w.checkout(path.Base(key)), ".claude", "skills", "implement", "SKILL.md"))
		c.expect(err == nil, "%s: skills not installed: %v", key, err)
	}
	tr := beads.NewTracker(w.trackerClone(), w.user, w.checkout("work"))
	labels, err := tr.GroupLabels(env.rig.Group.FullPath)
	c.expect(err == nil && len(labels) > 0, "the tracker holds no labels for %s: %v", env.rig.Group.FullPath, err)
	if err := c.err("gitlab provisioning"); err != nil {
		return err
	}
	return gitlabWorkload(env)
}
