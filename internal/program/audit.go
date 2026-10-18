package program

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
)

// AuditRunner runs the corpus gate command in a directory and returns its
// combined output. Tests replace it.
type AuditRunner func(dir, command string) (string, error)

// ShellAudit runs the command with sh -c.
func ShellAudit(dir, command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.String(), err
}

// AuditGate is the corpus gate merge runs on a branch that changes docs/:
// the command (mage audit by default, policy admission.audit_command) run
// in Dir, the worktree holding the branch.
type AuditGate struct {
	Command string
	Dir     string
	Run     AuditRunner
}

// checkCloses parses each --closes reference, project#iid, and refuses one
// that is malformed, missing, or already closed.
func checkCloses(t gitlab.Tracker, closes []string) ([][2]any, error) {
	var out [][2]any
	for _, ref := range closes {
		p, n, ok := strings.Cut(ref, "#")
		other, err := strconv.Atoi(n)
		if !ok || p == "" || err != nil || other <= 0 {
			return nil, fmt.Errorf("--closes %q is not project#iid", ref)
		}
		c, err := t.GetIssue(p, other)
		if err != nil {
			return nil, fmt.Errorf("--closes %s: %w", ref, err)
		}
		if c.State == "closed" {
			return nil, fmt.Errorf("--closes %s is already closed", ref)
		}
		out = append(out, [2]any{p, other})
	}
	return out, nil
}

// checkBefore runs the gate, naming what was withheld when it fails.
func (g AuditGate) checkBefore(withheld string) error {
	if g.Command == "" {
		return fmt.Errorf("the branch changes docs/ and no audit command is configured")
	}
	run := g.Run
	if run == nil {
		run = ShellAudit
	}
	out, err := run(g.Dir, g.Command)
	if err == nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
	}
	return fmt.Errorf("%s failed in %s; %s:\n%s", g.Command, g.Dir, withheld, strings.Join(lines, "\n"))
}
