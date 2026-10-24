// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package scaffold_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/commands"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/install"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/scaffold"
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
		t.Fatalf("AGENTS.md is not one managed block:\n%s", agents)
	}
	// Every workflow runs as a skill; AGENTS.md names no command (GH-36).
	if strings.Contains(agents, "tpm") {
		t.Errorf("AGENTS.md names scrum-master:\n%s", agents)
	}
	skills, err := install.EmbeddedSkills()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range skills {
		listed := strings.Contains(agents, "- "+s.Name+": ")
		// Every skill but none serves the system: sprint-end too, since the
		// system repository has its own sprint-end issue.
		if !listed {
			t.Errorf("skill %s listed %v at the system level", s.Name, listed)
		}
	}
	if !strings.Contains(agents, "Every figure is Mermaid in a fenced code block") {
		t.Error("AGENTS.md lacks the Mermaid figure rule")
	}
}
