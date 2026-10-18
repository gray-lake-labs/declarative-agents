package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestSpecToolSourceUsesThePinnedRelease(t *testing.T) {
	dir, release := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(release, "go.sum"), []byte("example.com/x v1.0.0 h1:abc=\n"), 0o644)
	if err := writeSpecToolSource(dir, release); err != nil {
		t.Fatal(err)
	}
	mod, _ := os.ReadFile(filepath.Join(dir, "go.mod"))
	if !strings.Contains(string(mod), "replace "+releaseModule+" => "+release) {
		t.Fatalf("go.mod does not resolve the release to its extracted source:\n%s", mod)
	}
	if sum, _ := os.ReadFile(filepath.Join(dir, "go.sum")); !strings.Contains(string(sum), "example.com/x") {
		t.Fatal("go.sum does not carry the release's checksums")
	}
	main, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	for _, want := range []string{`"` + releaseModule + `/pkg/spec/markdown"`, "ConvertYAML", "FormatLinked", "LinkIndexes", "Diagnose", "not-canonical"} {
		if !strings.Contains(string(main), want) {
			t.Errorf("main.go lacks %s", want)
		}
	}
}

func TestSpecToolIsBuiltOncePerTag(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	dir, err := releaseDir("v1")
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "spectool"), []byte("built"), 0o755)
	ran := 0
	got, err := ensureSpecTool("/cache", "v1", func(string, string, ...string) error { ran++; return nil })
	if err != nil || got != filepath.Join(dir, "spectool") || ran != 0 {
		t.Fatalf("got %s err %v ran %d", got, err, ran)
	}
}

// TestSpecToolConvertsAndFormats builds the tool against a real release and
// runs it on a copy of the corpus. It is slow and needs the clone cache, so
// it runs only when SCRUM_MASTER_SPEC_TOOL_TAG names the release.
func TestSpecToolConvertsAndFormats(t *testing.T) {
	tag := os.Getenv("SCRUM_MASTER_SPEC_TOOL_TAG")
	if tag == "" {
		t.Skip("set SCRUM_MASTER_SPEC_TOOL_TAG to a release in the clone cache")
	}
	cache, err := parentCheckout("..")
	if err != nil {
		t.Fatal(err)
	}
	tool, err := ensureSpecTool(cache, tag, runStreaming)
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	run := func(name string, args ...string) {
		t.Helper()
		if err := runStreaming(work, name, args...); err != nil {
			t.Fatalf("%s %v: %v", name, args, err)
		}
	}
	run("cp", "-R", mustAbs(t, filepath.Join("..", "docs")), ".")
	run("mkdir", "-p", filepath.Join("docs", "specs", "objectives"))
	writeFile(t, filepath.Join(work, "docs", "specs", "objectives", "obj001-reliable-agents.yaml"), objectiveYAML)
	run("git", "init", "-q")
	run("git", "add", ".")
	run(tool, "convert", "all")
	if _, err := os.Stat(filepath.Join(work, "docs", "specs", "objectives", "obj001-reliable-agents.md")); err != nil {
		t.Fatal("convert left no objective document in Markdown")
	}
	run("git", "add", "-A")
	run(tool, "fmt", "all")
	run(tool, "check", "all")
}

// objectiveYAML is an objective document in YAML with its data in the
// declared fields, which the release converts to an SRD.
const objectiveYAML = `id: obj001-reliable-agents
title: Reliable agents
status: agreed
context:
  owner: technical program manager
  horizon: 2026-W43..2027-W03
problem: Nobody can show that an agent works before it is deployed.
goals:
  - "G1: Every deployed agent passed an evaluation."
requirements:
  R1:
    title: Evaluation of one agent
    applicability:
      due: "2026-11-06"
    items:
      - R1.1: A grader scores one agent's trajectories.
non_goals:
  - Choosing which agents to deploy.
acceptance_criteria:
  - id: AC1
    criterion: One agent is scored.
    traces: [R1.1]
    tests: [grader_scores_agent]
    applicability: dev
`

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	a, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// TestSpecToolReadsObjectivesAsSRDs holds the tool to reading an objective
// document, at any depth under an objectives directory, with the SRD
// declaration under the scope objective (design.md objective).
func TestSpecToolReadsObjectivesAsSRDs(t *testing.T) {
	m := regexp.MustCompile(`objectivePath = regexp.MustCompile\(("[^"]+")\)`).FindStringSubmatch(specToolMain)
	if m == nil {
		t.Fatal("specToolMain declares no objectivePath")
	}
	pattern, err := strconv.Unquote(m[1])
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(pattern)
	for path, want := range map[string]bool{
		"docs/specs/objectives/obj001-reliable-agents.md":       true,
		"docs/specs/objectives/obj001-reliable-agents.yaml":     true,
		"testdata/objectives/malformed/obj002-missing-owner.md": true,
		"docs/specs/software-requirements/srd001-labels.md":     false,
		"docs/specs/objectives/README.md":                       false,
		"testdata/legacy/obj001-reliable-agents.yaml":           false,
	} {
		if re.MatchString(path) != want {
			t.Errorf("%s: objective = %v, want %v", path, !want, want)
		}
	}
	for _, want := range []string{`declarations["srd"]`, `"objective"`} {
		if !strings.Contains(specToolMain, want) {
			t.Errorf("specToolMain lacks %s", want)
		}
	}
}
