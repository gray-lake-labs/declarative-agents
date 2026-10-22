package program

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/tracker"
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

// MergeGate is what merge checks before merging. The corpus gate runs on a
// branch that changes docs/: the command (mage audit by default, policy
// admission.audit_command) run in Dir, the worktree holding the branch.
// RequireApproval (policy admission.require_approval) refuses a branch
// without another person's approval of its tip.
type MergeGate struct {
	Command         string
	Dir             string
	Run             AuditRunner
	RequireApproval bool
}

// checkCloses parses each --closes reference, project#iid, and refuses one
// that is malformed or missing. One already closed, as a Sprint Goal both a
// team's and the program's sprint-end merges name, is returned apart: the
// merge reports it and closes the rest.
func checkCloses(t tracker.Tracker, closes []string) (open [][2]any, closed []string, err error) {
	for _, ref := range closes {
		p, n, ok := strings.Cut(ref, "#")
		other, err := strconv.Atoi(n)
		if !ok || p == "" || err != nil || other <= 0 {
			return nil, nil, fmt.Errorf("--closes %q is not project#iid", ref)
		}
		c, err := t.GetIssue(p, other)
		if err != nil {
			return nil, nil, fmt.Errorf("--closes %s: %w", ref, err)
		}
		if c.State == "closed" {
			closed = append(closed, ref)
			continue
		}
		open = append(open, [2]any{p, other})
	}
	return open, closed, nil
}

// checkBefore runs the gate, naming what was withheld when it fails.
func (g MergeGate) checkBefore(withheld string) error {
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
