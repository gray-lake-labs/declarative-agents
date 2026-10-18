package gitlabrig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// recorder captures every command and answers from a script keyed on a
// command prefix, so the tests prove the sequences without terraform, glab,
// or a GitLab instance.
type recorder struct {
	calls   []string
	answers map[string]answer
	missing map[string]bool
	// failFirst makes a command prefix fail its first n calls.
	failFirst map[string]int
}

type answer struct {
	out string
	err error
}

func (r *recorder) Run(dir string, env []string, name string, args ...string) (string, error) {
	call := strings.TrimSpace(name + " " + strings.Join(args, " "))
	r.calls = append(r.calls, call)
	for prefix, n := range r.failFirst {
		if strings.HasPrefix(call, prefix) && n > 0 {
			r.failFirst[prefix] = n - 1
			return "", errors.New("request timed out")
		}
	}
	for prefix, a := range r.answers {
		if strings.HasPrefix(call, prefix) {
			return a.out, a.err
		}
	}
	return "", nil
}

func (r *recorder) LookPath(name string) error {
	if r.missing[name] {
		return errors.New("not found")
	}
	return nil
}

func (r *recorder) called(prefix string) int {
	n := 0
	for _, c := range r.calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

type fakeAPI struct {
	user, level int
	userErr     error
	subgroups   []Subgroup
	groups      map[string]int
	deleteErr   error
	deleted     []int
	empty       map[string]bool
	repairErr   error
	repaired    []string
}

func (f *fakeAPI) EnsureInitialCommit(id string) (bool, error) {
	if f.repairErr != nil {
		return false, f.repairErr
	}
	if f.empty[id] {
		f.repaired = append(f.repaired, id)
		f.empty[id] = false
		return true, nil
	}
	return false, nil
}

func (f *fakeAPI) CurrentUserID() (int, error)          { return f.user, f.userErr }
func (f *fakeAPI) AccessLevel(string, int) (int, error) { return f.level, nil }
func (f *fakeAPI) Subgroups(string) ([]Subgroup, error) { return f.subgroups, nil }
func (f *fakeAPI) GroupID(path string) (int, error) {
	if id, ok := f.groups[path]; ok {
		return id, nil
	}
	return 99, nil
}
func (f *fakeAPI) DeleteGroup(id int) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func config(group string) Config {
	c := Defaults()
	c.Group = group
	return c
}

func TestEmptyGroupIsRefusedByName(t *testing.T) {
	_, _, err := Preflight(config(""), &recorder{}, nil)
	if err == nil || !strings.Contains(err.Error(), "names no group") {
		t.Fatalf("err = %v", err)
	}
}

func TestPreflightNamesMissingTerraformAndToken(t *testing.T) {
	_, _, err := Preflight(config("g"), &recorder{missing: map[string]bool{"terraform": true}}, nil)
	if err == nil || !strings.Contains(err.Error(), "terraform is not on PATH") {
		t.Fatalf("terraform: err = %v", err)
	}
	t.Setenv("GITLAB_TOKEN", "")
	_, _, err = Preflight(config("g"), &recorder{missing: map[string]bool{"glab": true}}, nil)
	if err == nil || !strings.Contains(err.Error(), "set GITLAB_TOKEN") {
		t.Fatalf("token: err = %v", err)
	}
}

func TestPreflightRefusesAUserBelowMaintainer(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "tok")
	api := &fakeAPI{user: 7, level: 30}
	_, _, err := Preflight(config("g"), &recorder{}, func(string) API { return api })
	if err == nil || !strings.Contains(err.Error(), "need maintainer or owner") {
		t.Fatalf("err = %v", err)
	}
	api.level = 50
	if _, tok, err := Preflight(config("g"), &recorder{}, func(string) API { return api }); err != nil || tok != "tok" {
		t.Fatalf("owner: tok=%q err=%v", tok, err)
	}
}

func TestTokenFallsBackToGlab(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	r := &recorder{answers: map[string]answer{"sh -c glab auth status": {out: "gitlab.com\n  ✓ Token found: glpat-abc123\n"}}}
	tok, err := Token(r, "gitlab.com")
	if err != nil || tok != "glpat-abc123" {
		t.Fatalf("tok=%q err=%v", tok, err)
	}
}

func newTestRig(t *testing.T, r *recorder) Rig {
	t.Helper()
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	api := &fakeAPI{groups: map[string]int{}}
	return Rig{Root: t.TempDir(), Config: config("parent/group"), Runner: r, API: api, Token: "tok",
		Now: func() time.Time { return now }}
}

func TestUpWritesOutputsAndRecordsTheRun(t *testing.T) {
	r := &recorder{answers: map[string]answer{"terraform output -json rig": {out: `{"run_id":"x","group":{"full_path":"parent/group/rig-x"}}`}}}
	g := newTestRig(t, r)
	run, err := g.Up()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(run.ID, "20261015-120000-") {
		t.Fatalf("run id %q", run.ID)
	}
	if r.called("terraform init -input=false -reconfigure -backend-config=path=") != 1 || r.called("terraform apply -input=false -auto-approve -var group=parent/group -var run_id="+run.ID) != 1 {
		t.Fatalf("calls: %q", r.calls)
	}
	o, err := LoadOutputs(g.Root)
	if err != nil || o.Group.FullPath != "parent/group/rig-x" {
		t.Fatalf("outputs=%+v err=%v", o, err)
	}
}

const twoProjects = `{"run_id":"x","group":{"id":"7","full_path":"parent/group/rig-x"},"projects":{"system":{"id":"11"},"product-owners/po-1":{"id":12}}}`

func TestUpRepairsAnEmptyRepository(t *testing.T) {
	r := &recorder{answers: map[string]answer{"terraform output -json rig": {out: twoProjects}}}
	g := newTestRig(t, r)
	api := g.API.(*fakeAPI)
	api.empty = map[string]bool{"12": true}
	if _, err := g.Up(); err != nil {
		t.Fatal(err)
	}
	if len(api.repaired) != 1 || api.repaired[0] != "12" {
		t.Fatalf("repaired = %v", api.repaired)
	}
	if _, err := CurrentRunID(g.Root); err != nil {
		t.Fatalf("run not recorded: %v", err)
	}
}

func TestFailedRepairRemovesTheRig(t *testing.T) {
	r := &recorder{answers: map[string]answer{"terraform output -json rig": {out: twoProjects}}}
	g := newTestRig(t, r)
	api := g.API.(*fakeAPI)
	api.repairErr = errors.New("commit refused")
	if _, err := g.Up(); err == nil || !strings.Contains(err.Error(), "commit refused") {
		t.Fatalf("err = %v", err)
	}
	if len(api.deleted) != 1 {
		t.Fatalf("rig not removed: %v", api.deleted)
	}
	if _, err := CurrentRunID(g.Root); err == nil {
		t.Fatal("failed run recorded as current")
	}
}

func TestDownRunsAfterAFailingTarget(t *testing.T) {
	r := &recorder{answers: map[string]answer{"terraform output": {out: "{}"}}}
	g := newTestRig(t, r)
	targetErr := errors.New("target failed")
	err := g.Regression(func() error { return nil }, func() error { return targetErr }, func() error { t.Fatal("ran after failure"); return nil })
	if !errors.Is(err, targetErr) {
		t.Fatalf("err = %v", err)
	}
	if len(g.API.(*fakeAPI).deleted) != 1 || r.called("terraform destroy") != 0 {
		t.Fatalf("rig group not deleted through the API: deleted=%v calls=%q", g.API.(*fakeAPI).deleted, r.calls)
	}
	if entries, _ := os.ReadDir(filepath.Join(g.Root, StateDir)); len(entries) != 0 {
		t.Fatalf("run directory left behind: %v", entries)
	}
}

func TestFailedApplyRemovesThePartialRig(t *testing.T) {
	r := &recorder{answers: map[string]answer{"terraform apply": {err: errors.New("apply failed")}}}
	g := newTestRig(t, r)
	if _, err := g.Up(); err == nil {
		t.Fatal("expected apply error")
	}
	if d := g.API.(*fakeAPI).deleted; len(d) != 1 {
		t.Fatalf("partial rig not removed: %v", d)
	}
}

func TestUpRetriesATransientApplyFailure(t *testing.T) {
	r := &recorder{answers: map[string]answer{"terraform output": {out: "{}"}}, failFirst: map[string]int{"terraform apply": 2}}
	g := newTestRig(t, r)
	if _, err := g.Up(); err != nil {
		t.Fatal(err)
	}
	if r.called("terraform apply") != 3 || len(g.API.(*fakeAPI).deleted) != 0 {
		t.Fatalf("calls: %q", r.calls)
	}
}

func TestDownDestroysServiceAccountsBeforeTheGroup(t *testing.T) {
	r := &recorder{answers: map[string]answer{"terraform output": {out: "{}"}}}
	g := newTestRig(t, r)
	g.Config.ServiceAccounts = true
	if _, err := g.Up(); err != nil {
		t.Fatal(err)
	}
	if err := g.Down(""); err != nil {
		t.Fatal(err)
	}
	if r.called("terraform destroy -input=false -auto-approve -target=module.program.gitlab_group_variable.bot_token") != 1 {
		t.Fatalf("calls: %q", r.calls)
	}
}

func TestDownKeepsStateWhenDestroyFails(t *testing.T) {
	r := &recorder{answers: map[string]answer{"terraform output": {out: "{}"}}}
	g := newTestRig(t, r)
	g.API.(*fakeAPI).deleteErr = errors.New("boom")
	run, err := g.Up()
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Down(""); err == nil || !strings.Contains(err.Error(), "state kept") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(g.Root, StateDir, run.ID, runFile)); err != nil {
		t.Fatalf("state removed: %v", err)
	}
}

func TestSweepSelectsOnlyOldRigGroups(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	api := &fakeAPI{subgroups: []Subgroup{
		{ID: 1, Path: "rig-old", FullPath: "g/rig-old", CreatedAt: now.Add(-48 * time.Hour)},
		{ID: 2, Path: "rig-new", FullPath: "g/rig-new", CreatedAt: now.Add(-1 * time.Hour)},
		{ID: 3, Path: "platform", FullPath: "g/platform", CreatedAt: now.Add(-480 * time.Hour)},
	}}
	deleted, err := Sweep(api, "g", 24*time.Hour, now)
	if err != nil || len(deleted) != 1 || deleted[0] != "g/rig-old" || len(api.deleted) != 1 || api.deleted[0] != 1 {
		t.Fatalf("deleted=%v api=%v err=%v", deleted, api.deleted, err)
	}
}

func TestConfigFileOverridesDefaultsAndRejectsMalformed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ConfigFile), []byte("---\nid: regression\ntitle: Rig\n---\n\n<!-- field: group -->\n## Group\n\na/b\n\n<!-- field: po_repos -->\n## Po repos\n\n3\n\n<!-- field: sweep_age -->\n## Sweep age\n\n2h\n\n<!-- field: reports -->\n## Reports\n\nfalse\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(root)
	if err != nil || c.Group != "a/b" || c.POCount != 3 || c.SweepAge != 2*time.Hour || *c.Reports || c.BaseURL != "https://gitlab.com" {
		t.Fatalf("c=%+v err=%v", c, err)
	}
	if err := os.WriteFile(filepath.Join(root, ConfigFile), []byte("group: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(root); err == nil || !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("err = %v", err)
	}
}

// TestExampleConfigLoads reads the tracked template of regression.md.
func TestExampleConfigLoads(t *testing.T) {
	root := t.TempDir()
	b, err := os.ReadFile(filepath.Join("..", "..", "regression.example.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ConfigFile), b, 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(root)
	if err != nil || c.Group != "" || c.POCount != 2 || c.SweepAge != 24*time.Hour || !*c.Reports || c.ServiceAccounts {
		t.Fatalf("c=%+v err=%v", c, err)
	}
}
