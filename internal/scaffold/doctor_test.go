package scaffold

import (
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
