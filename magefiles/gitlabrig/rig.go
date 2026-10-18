package gitlabrig

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// StateDir holds one directory per run: state, provider data, and outputs.
	StateDir   = ".regression"
	moduleDir  = "infra/regression"
	outputs    = "outputs.json"
	runFile    = "run.json"
	currentRun = "current"

	applyAttempts = 3
)

// Rig is one configured rig in one repository checkout.
type Rig struct {
	Root   string
	Config Config
	Runner Runner
	API    API
	Token  string
	Now    func() time.Time
}

// Run is what up records for down and for the integration targets.
type Run struct {
	ID       string `json:"run_id"`
	Group    string `json:"group"`
	POCount  int    `json:"po_repos"`
	Reports  bool   `json:"reports"`
	Accounts bool   `json:"service_accounts"`
}

// Outputs is the infra/regression rig output the integration targets read.
type Outputs struct {
	RunID string `json:"run_id"`
	Group struct {
		ID       ID     `json:"id"`
		FullPath string `json:"full_path"`
		WebURL   string `json:"web_url"`
	} `json:"group"`
	Projects map[string]struct {
		ID                ID     `json:"id"`
		PathWithNamespace string `json:"path_with_namespace"`
		WebURL            string `json:"web_url"`
	} `json:"projects"`
}

// ID is a resource id Terraform outputs as a string or a number.
type ID string

func (i *ID) UnmarshalJSON(b []byte) error {
	*i = ID(strings.Trim(string(b), `"`))
	return nil
}

// NewRunID is the UTC timestamp and a short random suffix.
func NewRunID(now time.Time) string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return now.UTC().Format("20060102-150405") + "-" + hex.EncodeToString(b)
}

func (g Rig) runDir(id string) string { return filepath.Join(g.Root, StateDir, id) }

func (g Rig) env(id string) []string {
	return []string{
		"GITLAB_TOKEN=" + g.Token,
		"GITLAB_BASE_URL=" + g.Config.APIURL(),
		"TF_DATA_DIR=" + filepath.Join(g.runDir(id), ".terraform"),
		"TF_IN_AUTOMATION=1",
	}
}

func (r Run) vars() []string {
	return []string{
		"-var", "group=" + r.Group,
		"-var", "run_id=" + r.ID,
		"-var", "po_repos=" + strconv.Itoa(r.POCount),
		"-var", "reports=" + strconv.FormatBool(r.Reports),
		"-var", "service_accounts_enabled=" + strconv.FormatBool(r.Accounts),
	}
}

func (g Rig) terraform(id string, args ...string) (string, error) {
	return g.Runner.Run(filepath.Join(g.Root, moduleDir), g.env(id), "terraform", args...)
}

// Up applies infra/regression for a new run, writes its outputs, and
// records it as the current run. A failed apply is destroyed before
// returning, so a broken up leaves nothing behind.
func (g Rig) Up() (Run, error) {
	now := time.Now
	if g.Now != nil {
		now = g.Now
	}
	run := Run{ID: NewRunID(now()), Group: g.Config.Group, POCount: g.Config.POCount, Reports: *g.Config.Reports, Accounts: g.Config.ServiceAccounts}
	dir := g.runDir(run.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return run, err
	}
	if err := writeJSON(filepath.Join(dir, runFile), run); err != nil {
		return run, err
	}
	state := "-backend-config=path=" + filepath.Join(dir, "terraform.tfstate")
	if _, err := g.terraform(run.ID, "init", "-input=false", "-reconfigure", state); err != nil {
		return run, err
	}
	// The instance sometimes times out creating a project; apply again
	// creates only what is still missing.
	apply := append([]string{"apply", "-input=false", "-auto-approve"}, run.vars()...)
	var err error
	for attempt := 1; attempt <= applyAttempts; attempt++ {
		if _, err = g.terraform(run.ID, apply...); err == nil {
			break
		}
	}
	if err != nil {
		if derr := g.Down(run.ID); derr != nil {
			return run, errors.Join(err, fmt.Errorf("cleanup after failed apply: %w", derr))
		}
		return run, err
	}
	out, err := g.terraform(run.ID, "output", "-json", "rig")
	if err != nil {
		return run, err
	}
	if err := os.WriteFile(filepath.Join(dir, outputs), []byte(out), 0o644); err != nil {
		return run, err
	}
	if err := g.ensureInitialCommits(run, dir); err != nil {
		if derr := g.Down(run.ID); derr != nil {
			return run, errors.Join(err, fmt.Errorf("cleanup after failed repair: %w", derr))
		}
		return run, err
	}
	return run, os.WriteFile(filepath.Join(g.Root, StateDir, currentRun), []byte(run.ID+"\n"), 0o644)
}

// ensureInitialCommits gives every rig repository an initial commit on main.
// GitLab sometimes creates a project empty although initialize_with_readme
// is set, and later targets branch from main.
func (g Rig) ensureInitialCommits(run Run, dir string) error {
	var o Outputs
	if err := readJSON(filepath.Join(dir, outputs), &o); err != nil {
		return err
	}
	keys := make([]string, 0, len(o.Projects))
	for k := range o.Projects {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		repaired, err := g.API.EnsureInitialCommit(string(o.Projects[k].ID))
		if err != nil {
			return err
		}
		if repaired {
			fmt.Printf("rig-%s: %s had no initial commit; added README.md on main\n", run.ID, k)
		}
	}
	return nil
}

// Down removes a run's rig subgroup and its directory. An empty id means
// the current run. The subgroup is deleted through the API, mark then
// permanent removal, which takes every project in it: terraform destroy of
// the projects fails on instances that require marking before permanent
// deletion. Service accounts live on the top-level group, outside the
// subgroup, so when the run has them terraform destroys those first.
func (g Rig) Down(id string) error {
	if id == "" {
		cur, err := CurrentRunID(g.Root)
		if err != nil {
			return err
		}
		id = cur
	}
	var run Run
	if err := readJSON(filepath.Join(g.runDir(id), runFile), &run); err != nil {
		return fmt.Errorf("run %s: %w", id, err)
	}
	if run.Accounts {
		destroy := append([]string{"destroy", "-input=false", "-auto-approve",
			"-target=module.program.gitlab_group_variable.bot_token",
			"-target=module.program.gitlab_group_service_account_access_token.bot",
			"-target=module.program.gitlab_group_membership.bot",
			"-target=module.program.gitlab_group_service_account.bot"}, run.vars()...)
		if _, err := g.terraform(id, destroy...); err != nil {
			return fmt.Errorf("destroy service accounts of run %s (state kept in %s): %w", id, g.runDir(id), err)
		}
	}
	path := run.Group + "/rig-" + run.ID
	gid, err := g.API.GroupID(path)
	if err == nil && gid != 0 {
		err = g.API.DeleteGroup(gid)
	}
	if err != nil {
		return fmt.Errorf("delete %s (state kept in %s): %w", path, g.runDir(id), err)
	}
	if cur, _ := CurrentRunID(g.Root); cur == id {
		_ = os.Remove(filepath.Join(g.Root, StateDir, currentRun))
	}
	return os.RemoveAll(g.runDir(id))
}

// Regression runs up, then each target, then down. down runs even when a
// target fails; the first target error is returned, joined with any
// teardown error.
func (g Rig) Regression(targets ...func() error) (err error) {
	run, err := g.Up()
	if err != nil {
		return err
	}
	defer func() {
		if derr := g.Down(run.ID); derr != nil {
			err = errors.Join(err, derr)
		}
	}()
	for _, t := range targets {
		if err := t(); err != nil {
			return err
		}
	}
	return nil
}

// CurrentRunID is the run up recorded last.
func CurrentRunID(root string) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, StateDir, currentRun))
	if err != nil {
		return "", fmt.Errorf("no current regression run; run mage regression:up first")
	}
	return strings.TrimSpace(string(b)), nil
}

// LoadOutputs reads the current run's outputs for an integration target.
func LoadOutputs(root string) (Outputs, error) {
	var o Outputs
	id, err := CurrentRunID(root)
	if err != nil {
		return o, err
	}
	return o, readJSON(filepath.Join(root, StateDir, id, outputs), &o)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
