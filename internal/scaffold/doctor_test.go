package scaffold

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
	"github.com/gray-lake-labs/scrum-master/internal/tracker/fake"
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

func TestDoctorAppendsToOrientationWithoutBlock(t *testing.T) {
	dir := older(t)
	agents := filepath.Join(dir, AgentsFile)
	os.WriteFile(agents, []byte("Run the linter before committing.\n"), 0o644)
	dry, err := Doctor(DoctorOptions{Dir: dir, DryRun: true})
	if err != nil || actions(dry.Files)[AgentsFile] != "appended" {
		t.Fatalf("dry run: %v, %v", actions(dry.Files), err)
	}
	if b, _ := os.ReadFile(agents); string(b) != "Run the linter before committing.\n" {
		t.Fatalf("the dry run wrote AGENTS.md:\n%s", b)
	}
	r, err := Doctor(DoctorOptions{Dir: dir})
	if err != nil || actions(r.Files)[AgentsFile] != "appended" {
		t.Fatalf("doctor: %v, %v", actions(r.Files), err)
	}
	b, _ := os.ReadFile(agents)
	if !strings.HasPrefix(string(b), "Run the linter before committing.\n\n"+BlockBegin) || !strings.Contains(string(b), "# Agent instructions") {
		t.Fatalf("appended AGENTS.md:\n%s", b)
	}
	if again, _ := Doctor(DoctorOptions{Dir: dir}); actions(again.Files)[AgentsFile] != "kept" {
		t.Fatalf("second run: %s", actions(again.Files)[AgentsFile])
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

// olderLabels is control/labels.md as a program set up before the op axis
// keeps it: no op section, no kind::question, kind::build recolored, and a
// kind::chore of the program's own.
func olderLabels() string {
	var labels []tracker.Label
	for _, l := range program.DefaultLabels() {
		switch {
		case strings.HasPrefix(l.Name, "op::"), l.Name == "kind::question":
			continue
		case l.Name == "kind::build":
			l.Color = "#000000"
		}
		labels = append(labels, l)
	}
	labels = append(labels, tracker.Label{Name: "kind::chore", Color: "#111111", Description: "Housekeeping"})
	doc := LabelsDocument(labels)
	start, end := strings.Index(doc, "\n<!-- field: op -->"), strings.Index(doc, "\n<!-- field: state -->")
	return doc[:start] + doc[end:]
}

// TestDoctorExtendsLabels adds the op axis and the missing kind::question
// to an older control/labels.md, keeps the program's recolor and its own
// value byte for byte, reports the additions on a dry run without writing,
// and changes nothing on a second run (srd043 R4.2, R4.6).
func TestDoctorExtendsLabels(t *testing.T) {
	dir := repo(t)
	if _, err := Init(InitOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, program.LabelsFile)
	before := olderLabels()
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	labelsResult := func(r DoctorResult) Result {
		for _, f := range r.Files {
			if f.Path == program.LabelsFile {
				return f
			}
		}
		t.Fatalf("doctor reported no %s", program.LabelsFile)
		return Result{}
	}
	want := []string{"kind::question", "op::file", "op::plan", "op::goals", "op::onboard", "op::report", "op::status", "op::recur"}

	dry, err := Doctor(DoctorOptions{Dir: dir, DryRun: true})
	if f := labelsResult(dry); err != nil || f.Action != "extended" || !slices.Equal(f.Labels, want) || !strings.Contains(f.Reason, "tpm issue labels") {
		t.Fatalf("dry run: %+v, %v", f, err)
	}
	if b, _ := os.ReadFile(path); string(b) != before {
		t.Fatal("the dry run wrote control/labels.md")
	}

	r, err := Doctor(DoctorOptions{Dir: dir})
	if f := labelsResult(r); err != nil || f.Action != "extended" || !slices.Equal(f.Labels, want) {
		t.Fatalf("doctor: %+v, %v", f, err)
	}
	b, _ := os.ReadFile(path)
	got := string(b)
	for _, keep := range []string{"| build | #000000 | Builds or changes software |", "| chore | #111111 | Housekeeping |", "| question | #a8d695 | Needs an answer, not a change |"} {
		if !strings.Contains(got, keep) {
			t.Errorf("control/labels.md lacks %q:\n%s", keep, got)
		}
	}
	if strings.Index(got, "<!-- field: op -->") < strings.Index(got, "<!-- field: component -->") || strings.Index(got, "<!-- field: op -->") > strings.Index(got, "<!-- field: state -->") {
		t.Errorf("the op axis is not between component and state:\n%s", got)
	}
	f := fake.New()
	f.AddFile("g/system", "main", program.LabelsFile, b)
	labels, err := program.LoadLabels(f, "g/system", "main")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, l := range labels {
		names = append(names, l.Name)
	}
	for _, w := range append(want, "kind::chore") {
		if !slices.Contains(names, w) {
			t.Errorf("LoadLabels lacks %s: %v", w, names)
		}
	}

	again, err := Doctor(DoctorOptions{Dir: dir})
	if f := labelsResult(again); err != nil || f.Action != "kept" {
		t.Fatalf("second run: %+v, %v", f, err)
	}
	if b2, _ := os.ReadFile(path); string(b2) != got {
		t.Fatal("the second run changed control/labels.md")
	}

	if err := os.WriteFile(path, []byte("---\nid: labels\n---\n\n<!-- field: axes -->\n## Axes\n\n- [unclosed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	broken, err := Doctor(DoctorOptions{Dir: dir})
	if f := labelsResult(broken); err != nil || f.Action != "kept" || !strings.HasPrefix(f.Reason, "does not parse") {
		t.Fatalf("unparseable labels: %+v, %v", f, err)
	}
}

// TestExtendLabelsLeavesACompleteFile changes nothing in a file that lists
// every default.
func TestExtendLabelsLeavesACompleteFile(t *testing.T) {
	doc := []byte(LabelsDocument(program.DefaultLabels()))
	out, added, err := ExtendLabels(doc, program.DefaultLabels())
	if err != nil || len(added) != 0 || string(out) != string(doc) {
		t.Fatalf("added %v, err %v, changed %v", added, err, string(out) != string(doc))
	}
}

// TestDoctorUpgradesTheSystem adds what tpm init now writes to a system
// repository an older tpm laid down: the system SRD's context and its
// component row, and the corpus release's peer read, keeping the owner's
// text; a dry run writes nothing and a second run changes nothing.
func TestDoctorUpgradesTheSystem(t *testing.T) {
	dir := repo(t)
	if _, err := Init(InitOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "remote", "add", "origin", "https://git.example.invalid/g/system.git")
	srdPath, policyPath := filepath.Join(dir, systemSRDPath), filepath.Join(dir, program.PolicyFile)
	oldSRD := "---\nid: srd001-system\ntitle: \"System: what the system does\"\n---\n\n<!-- field: problem -->\n## Problem\n\nThe owner's problem.\n\n" +
		"<!-- field: goals -->\n## Goals\n\n- G1: the owner's goal.\n"
	oldPolicy := PolicyDocument(program.Defaults())
	os.WriteFile(srdPath, []byte(oldSRD), 0o644)
	os.WriteFile(policyPath, []byte(oldPolicy), 0o644)
	result := func(r DoctorResult, path string) Result {
		for _, f := range r.Files {
			if f.Path == path {
				return f
			}
		}
		t.Fatalf("doctor reported no %s", path)
		return Result{}
	}

	dry, err := Doctor(DoctorOptions{Dir: dir, DryRun: true})
	if err != nil || result(dry, systemSRDPath).Action != "extended" || result(dry, program.PolicyFile).Action != "extended" {
		t.Fatalf("dry run: %+v, %+v, %v", result(dry, systemSRDPath), result(dry, program.PolicyFile), err)
	}
	if b, _ := os.ReadFile(srdPath); string(b) != oldSRD {
		t.Fatal("the dry run wrote the system SRD")
	}

	if _, err := Doctor(DoctorOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	srd, _ := os.ReadFile(srdPath)
	got := string(srd)
	if !strings.HasPrefix(got, "---\nid: srd001-system\ntitle: \"System: what the system does\"\n---\n\n"+systemContext+"\n<!-- field: problem -->") ||
		!strings.Contains(got, "The owner's problem.") || !strings.HasSuffix(got, "| system | https://git.example.invalid/g/system.git | / |\n") {
		t.Fatalf("system SRD:\n%s", got)
	}
	f := fake.New()
	f.AddFile("g/system", "main", systemSRDPath, srd)
	policy, _ := os.ReadFile(policyPath)
	f.AddFile("g/system", "main", program.PolicyFile, policy)
	subs, _ := program.LoadSubsystems(f, "g/system", "main")
	if c, _, ok := subs.ComponentFor("g/system"); !ok || c.Name != "system" {
		t.Fatalf("the system is no component: %+v", subs)
	}
	p, err := program.LoadPolicy(f, "g/system", "main")
	if g := p.Gate("corpus"); err != nil || len(g) != 2 || g[0].Command != ProgramAudit || g[1].Kind != "peer-read" || len(p.Gate("default")) != 1 || p.Gate("default")[0].Kind != "tests" {
		t.Fatalf("policy gates: corpus %+v default %+v, %v", p.Gate("corpus"), p.Gate("default"), err)
	}

	again, err := Doctor(DoctorOptions{Dir: dir})
	if err != nil || result(again, systemSRDPath).Action != "kept" || result(again, program.PolicyFile).Action != "kept" {
		t.Fatalf("second run: %+v, %+v, %v", result(again, systemSRDPath), result(again, program.PolicyFile), err)
	}
}

// TestCorpusPeerReadKeepsTheProgramsGate leaves a policy that names its own
// corpus gate alone, and gives one with no definition of done init's whole
// one.
func TestCorpusPeerReadKeepsTheProgramsGate(t *testing.T) {
	own := InitPolicy()
	own.DefinitionOfDone["corpus"] = []program.Check{{Kind: "custom", Environment: "local", Command: "make check"}}
	doc := []byte(PolicyDocument(own))
	if out, changed, err := AddCorpusPeerRead(doc); err != nil || changed || string(out) != string(doc) {
		t.Fatalf("own corpus gate: changed %v, %v", changed, err)
	}
	bare := PolicyDocument(program.Defaults())
	bare = bare[:strings.Index(bare, "<!-- field: definition_of_done -->")]
	out, changed, err := AddCorpusPeerRead([]byte(bare))
	if err != nil || !changed {
		t.Fatalf("no definition of done: changed %v, %v", changed, err)
	}
	f := fake.New()
	f.AddFile("g/system", "main", program.PolicyFile, out)
	p, err := program.LoadPolicy(f, "g/system", "main")
	if err != nil || len(p.Gate("corpus")) != 2 || p.Gate("default")[0].Command != "go test ./..." {
		t.Fatalf("policy from %q: %+v, %v", out, p.DefinitionOfDone, err)
	}
}

// TestDoctorReportsUntrackedDocuments names documents git does not track,
// which the spec tool does not read, instead of passing them silently.
func TestDoctorReportsUntrackedDocuments(t *testing.T) {
	dir := repo(t)
	if _, err := Init(InitOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "add", "-A")
	run(t, dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "init")
	os.WriteFile(filepath.Join(dir, "docs", "specs", "epics", "ep099-new.md"), []byte("# new\n"), 0o644)
	r, err := Doctor(DoctorOptions{Dir: dir, DryRun: true})
	if err != nil || !slices.Equal(r.Repair.Untracked, []string{"docs/specs/epics/ep099-new.md"}) {
		t.Fatalf("untracked = %v, %v", r.Repair.Untracked, err)
	}
	if out := humanDoctor(r); !strings.Contains(out, "git does not track them") {
		t.Fatalf("doctor's text does not say so:\n%s", out)
	}
}
