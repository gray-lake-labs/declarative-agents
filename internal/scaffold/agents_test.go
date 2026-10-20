package scaffold_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master"
	"github.com/gray-lake-labs/scrum-master/internal/cli"
	_ "github.com/gray-lake-labs/scrum-master/internal/commands"
	"github.com/gray-lake-labs/scrum-master/internal/install"
	"github.com/gray-lake-labs/scrum-master/internal/scaffold"
)

func TestAgentsFile(t *testing.T) {
	dir := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if _, err := scaffold.Init(scaffold.InitOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, scaffold.AgentsFile))
	if err != nil {
		t.Fatal(err)
	}
	agents := string(b)
	if !strings.HasPrefix(agents, scaffold.BlockBegin) || !strings.HasSuffix(agents, scaffold.BlockEnd+"\n") {
		t.Fatalf("AGENTS.md is not one tpm block:\n%s", agents)
	}
	for _, g := range cli.Default.Groups() {
		if !strings.Contains(agents, "- tpm "+g+": ") {
			t.Errorf("AGENTS.md does not name tpm %s", g)
		}
	}
	skills, err := install.Skills(scrummaster.Plugin, scrummaster.SkillsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range skills {
		listed := strings.Contains(agents, "- "+s.Name+": ")
		if listed != (s.Name != "sprint-end") {
			t.Errorf("skill %s listed %v at the system level", s.Name, listed)
		}
	}
	if !strings.Contains(agents, "Every figure is Mermaid in a fenced code block") {
		t.Error("AGENTS.md lacks the Mermaid figure rule")
	}
}
