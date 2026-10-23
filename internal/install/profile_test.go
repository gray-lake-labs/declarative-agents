package install

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/gray-lake-labs/scrum-master"
)

// Every skill renders from its profile: the machine's states between the
// initial and the terminal ones are the numbered steps, in the machine's
// order (GH-135, GH-198; srd037 R1.4, srd036 R6.2).
func TestSkillsRenderFromTheirProfiles(t *testing.T) {
	skills, err := ProfileSkills(scrummaster.Plugin, scrummaster.AgentsDir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, s := range skills {
		names = append(names, s.Name)
	}
	want := []string{"draft", "implement", "merge", "plan", "pop", "specify", "sprint-end", "tasks"}
	if strings.Join(names, " ") != strings.Join(want, " ") {
		t.Fatalf("profiles rendered %v, want %v", names, want)
	}
	for _, s := range skills {
		if n := len([]rune(s.Description)); n < 1 || n > 1024 || strings.Contains(s.Description, "\n") {
			t.Errorf("%s: description must be one line of 1 to 1024 characters, got %d", s.Name, n)
		}
		mb, err := scrummaster.Plugin.ReadFile(scrummaster.AgentsDir + "/" + s.Name + "/machine.yaml")
		if err != nil {
			t.Fatal(err)
		}
		var m machineDoc
		if err := yaml.Unmarshal(mb, &m); err != nil {
			t.Fatal(err)
		}
		terminal := map[string]bool{}
		for _, name := range m.TerminalStates {
			terminal[name] = true
		}
		var steps []string
		for _, st := range m.States {
			if st.Name != m.InitialState && !terminal[st.Name] {
				steps = append(steps, st.Name)
			}
		}
		for i, name := range steps {
			if want := fmt.Sprintf("%d. **%s** — ", i+1, name); !strings.Contains(s.Body, want) {
				t.Errorf("%s: rendered body misses step %q", s.Name, want)
			}
		}
		if got := regexp.MustCompile(`(?m)^\d+\. \*\*`).FindAllString(s.Body, -1); len(got) != len(steps) {
			t.Errorf("%s: rendered %d steps, machine has %d", s.Name, len(got), len(steps))
		}
		for _, name := range m.TerminalStates {
			if !strings.Contains(s.Body, "**"+name+"**") {
				t.Errorf("%s: rendered body misses terminal state %s", s.Name, name)
			}
		}
		if !strings.Contains(s.Body, "design.md D1") {
			t.Errorf("%s: rendered body misses its design.md citation", s.Name)
		}
	}
}

// The implement skill keeps the workflow's commands in its rendered steps.
func TestImplementRenderKeepsItsCommands(t *testing.T) {
	skills, err := ProfileSkills(scrummaster.Plugin, scrummaster.AgentsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range skills {
		if s.Name != "implement" {
			continue
		}
		for _, cmd := range []string{"tpm issue pop", "git push", "tpm issue merge", "design.md D10"} {
			if !strings.Contains(s.Body, cmd) {
				t.Errorf("rendered body misses %q", cmd)
			}
		}
		if lines := strings.Count(Render("claude", s, "dev"), "\n"); lines > 70 {
			t.Errorf("rendered SKILL.md has %d lines; keep a profile-rendered skill readable at a glance", lines)
		}
	}
}

func TestMergeSkillsRefusesASkillWithTwoSources(t *testing.T) {
	a := []Skill{{Name: "implement", Description: "embedded"}}
	b := []Skill{{Name: "implement", Description: "rendered"}}
	if _, err := MergeSkills(a, b); err == nil {
		t.Fatal("a name with both an embedded SKILL.md and a profile must be refused")
	}
	merged, err := MergeSkills([]Skill{{Name: "plan", Description: "d"}}, b)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 2 || merged[0].Name != "implement" || merged[1].Name != "plan" {
		t.Fatalf("merged = %v, want implement then plan", merged)
	}
}

func TestRenderProfileSkillRefusesAnEmptyMachine(t *testing.T) {
	if _, err := renderProfileSkill("x", "plugin/agents/x", machineDoc{Purpose: "", States: []machineState{{Name: "A"}}}); err == nil {
		t.Error("a machine without a purpose must be refused")
	}
	if _, err := renderProfileSkill("x", "plugin/agents/x", machineDoc{Purpose: "p"}); err == nil {
		t.Error("a machine without states must be refused")
	}
	m := machineDoc{Purpose: "p", InitialState: "Idle", States: []machineState{{Name: "Idle"}, {Name: "End"}}, TerminalStates: []string{"End"}}
	if _, err := renderProfileSkill("x", "plugin/agents/x", m); err == nil {
		t.Error("a machine with no steps between initial and terminal states must be refused")
	}
}
