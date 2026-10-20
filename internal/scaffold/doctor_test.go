package scaffold

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// older is a system repository as an older tpm laid it down: the skeleton
// without the orientation files.
func older(t *testing.T) string {
	t.Helper()
	dir := repo(t)
	if _, err := Init(InitOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"README.md", AgentsFile, "docs/specs/themes/README.md"} {
		if err := os.Remove(filepath.Join(dir, f)); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestDoctorAddsMissing(t *testing.T) {
	dir := older(t)
	os.WriteFile(filepath.Join(dir, "docs", "VISION.md"), []byte("the owner's vision\n"), 0o644)
	r, err := Doctor(DoctorOptions{Dir: dir})
	if err != nil || r.Level != "system" {
		t.Fatalf("doctor: %+v, %v", r, err)
	}
	got := actions(r.Files)
	for p, a := range got {
		want := "kept"
		if p == "README.md" || p == AgentsFile || p == "docs/specs/themes/README.md" {
			want = "created"
		}
		if a != want {
			t.Errorf("%s: %s, want %s", p, a, want)
		}
	}
	if len(got) != len(skeleton) {
		t.Errorf("doctor reported %d files, want %d", len(got), len(skeleton))
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "docs", "VISION.md")); string(b) != "the owner's vision\n" {
		t.Errorf("doctor changed an edited VISION: %q", b)
	}
	for _, f := range r.Files {
		if f.Path == "docs/VISION.md" && f.Reason != "edited; doctor keeps it" {
			t.Errorf("VISION reason %q", f.Reason)
		}
	}
}

func TestDoctorRefreshesBlock(t *testing.T) {
	dir := repo(t)
	if _, err := Init(InitOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	olderBlock(t, filepath.Join(dir, "docs", "specs", "epics", "README.md"))
	r, err := Doctor(DoctorOptions{Dir: dir})
	if err != nil || actions(r.Files)["docs/specs/epics/README.md"] != "refreshed" {
		t.Fatalf("doctor: %v, %v", actions(r.Files), err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "docs", "specs", "epics", "README.md"))
	if !strings.Contains(string(b), "owner's note below") || !strings.Contains(string(b), "`epNNN-short-name.md`") {
		t.Fatalf("refreshed epics README:\n%s", b)
	}
}

func TestDoctorReportsStaleSkills(t *testing.T) {
	dir := repo(t)
	if _, err := Init(InitOptions{Dir: dir, Agents: []string{"claude"}, Version: "v0.1"}); err != nil {
		t.Fatal(err)
	}
	r, err := Doctor(DoctorOptions{Dir: dir, Version: "v0.2"})
	if err != nil || len(r.Skills) == 0 {
		t.Fatalf("doctor found no stale skills: %+v, %v", r.Skills, err)
	}
	for _, f := range r.Skills {
		if f.Agent != "claude" || f.Problem != "outdated" {
			t.Errorf("finding %+v", f)
		}
	}
	if h := humanDoctor(r); !strings.Contains(h, "run tpm install --agent claude") {
		t.Errorf("report does not name the fix:\n%s", h)
	}
	current, err := Doctor(DoctorOptions{Dir: dir, Version: "v0.1"})
	if err != nil || len(current.Skills) != 0 {
		t.Fatalf("current skills reported: %+v, %v", current.Skills, err)
	}
}

func TestDoctorDryRunWritesNothing(t *testing.T) {
	dir := older(t)
	r, err := Doctor(DoctorOptions{Dir: dir, DryRun: true})
	if err != nil || actions(r.Files)["README.md"] != "created" {
		t.Fatalf("dry run: %v, %v", actions(r.Files), err)
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err == nil {
		t.Fatal("a dry run wrote README.md")
	}
}

func TestDoctorSecondRunChangesNothing(t *testing.T) {
	dir := older(t)
	if _, err := Doctor(DoctorOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	r, err := Doctor(DoctorOptions{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for p, a := range actions(r.Files) {
		if a != "kept" {
			t.Errorf("second run: %s %s", a, p)
		}
	}
}

func TestDoctorRefusesNonProgram(t *testing.T) {
	if _, err := Doctor(DoctorOptions{Dir: repo(t)}); err == nil || !strings.Contains(err.Error(), "run tpm init or tpm onboard") {
		t.Fatalf("empty repository: %v", err)
	}
	if _, err := Doctor(DoctorOptions{Dir: t.TempDir()}); err == nil || !strings.Contains(err.Error(), "not a git repository") {
		t.Fatalf("plain directory: %v", err)
	}
}

func TestDoctorSubsystemGetsNoControl(t *testing.T) {
	dir := repo(t)
	if _, err := Onboard(system(t), joining(dir)); err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(dir, "docs", "specs", "stories", "README.md"))
	r, err := Doctor(DoctorOptions{Dir: dir})
	if err != nil || r.Level != "subsystem" || actions(r.Files)["docs/specs/stories/README.md"] != "created" {
		t.Fatalf("doctor: %s %v, %v", r.Level, actions(r.Files), err)
	}
	for p := range actions(r.Files) {
		if strings.HasPrefix(p, "control/") || strings.Contains(p, "themes/") || strings.Contains(p, "initiatives/") {
			t.Errorf("a subsystem repository got %s", p)
		}
	}
}

// The tests run offline: the spec tool finds nothing unless a test says
// otherwise, and a Go toolchain is on PATH.
func init() {
	specTool = func(string, string, string) ([]string, error) { return nil, nil }
	lookGo = func() error { return nil }
}

// stubTool replaces the spec tool for one test with one that answers each
// command with its lines and error, and records the commands run.
func stubTool(t *testing.T, answers map[string][]string, fails map[string]error) *[]string {
	t.Helper()
	var ran []string
	prev := specTool
	specTool = func(_, command, _ string) ([]string, error) {
		ran = append(ran, command)
		return answers[command], fails[command]
	}
	t.Cleanup(func() { specTool = prev })
	return &ran
}

func TestDoctorRepairRewritesCanonical(t *testing.T) {
	dir := repo(t)
	if _, err := Init(InitOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	ran := stubTool(t, map[string][]string{"fmt": {"formatted docs/VISION.md"}}, nil)
	r, err := Doctor(DoctorOptions{Dir: dir})
	if err != nil || len(r.Repair.Repaired) != 1 || r.Repair.Repaired[0] != "docs/VISION.md" || len(r.Repair.Findings) != 0 {
		t.Fatalf("repair: %+v, %v", r.Repair, err)
	}
	if strings.Join(*ran, ",") != "fmt,check" {
		t.Fatalf("ran %v, want fmt then check", *ran)
	}
	if h := humanDoctor(r); !strings.Contains(h, "repaired     docs/VISION.md") {
		t.Errorf("report:\n%s", h)
	}
}

func TestDoctorRepairReportsUnrepairable(t *testing.T) {
	dir := repo(t)
	if _, err := Init(InitOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	finding := "docs/specs/epics/ep001-x.md: missing-field: goals: add the goals field"
	stubTool(t, map[string][]string{"check": {finding}}, map[string]error{"check": errors.New("1 problem(s) in 9 document(s)")})
	r, err := Doctor(DoctorOptions{Dir: dir})
	if err != nil || len(r.Repair.Repaired) != 0 || len(r.Repair.Findings) != 1 || r.Repair.Findings[0] != finding {
		t.Fatalf("repair: %+v, %v", r.Repair, err)
	}
}

func TestDoctorRepairDryRunOnlyChecks(t *testing.T) {
	dir := repo(t)
	if _, err := Init(InitOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	ran := stubTool(t, map[string][]string{"check": {"docs/road-map.md" + notCanonical}}, map[string]error{"check": errors.New("1 problem(s)")})
	r, err := Doctor(DoctorOptions{Dir: dir, DryRun: true})
	if err != nil || len(r.Repair.Repaired) != 1 || r.Repair.Repaired[0] != "docs/road-map.md" || len(r.Repair.Findings) != 0 {
		t.Fatalf("dry-run repair: %+v, %v", r.Repair, err)
	}
	if strings.Join(*ran, ",") != "check" {
		t.Fatalf("a dry run ran %v", *ran)
	}
}

func TestDoctorRepairSkipsWithoutGo(t *testing.T) {
	dir := older(t)
	lookGo = func() error { return errors.New("not found") }
	t.Cleanup(func() { lookGo = func() error { return nil } })
	ran := stubTool(t, nil, nil)
	r, err := Doctor(DoctorOptions{Dir: dir})
	if err != nil || !strings.Contains(r.Repair.Skipped, "no Go toolchain") || len(*ran) != 0 {
		t.Fatalf("repair without Go: %+v, ran %v, %v", r.Repair, *ran, err)
	}
	if actions(r.Files)["README.md"] != "created" {
		t.Fatal("doctor without Go did not do the rest")
	}
}

func TestDoctorRepairSkipsUntrackedCorpus(t *testing.T) {
	dir := repo(t)
	if _, err := Init(InitOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	stubTool(t, nil, map[string]error{"fmt": errors.New(`no corpus document matches "all"`)})
	r, err := Doctor(DoctorOptions{Dir: dir})
	if err != nil || !strings.Contains(r.Repair.Skipped, "commit the documents") {
		t.Fatalf("repair of an untracked corpus: %+v, %v", r.Repair, err)
	}
}
