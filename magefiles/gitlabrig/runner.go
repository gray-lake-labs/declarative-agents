package gitlabrig

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Runner executes host commands. Tests use a recording runner.
type Runner interface {
	// Run executes name with args in dir with extra environment entries and
	// returns its standard output.
	Run(dir string, env []string, name string, args ...string) (string, error)
	LookPath(name string) error
}

// ExecRunner runs real commands, streaming stderr so an operator sees
// terraform's progress.
type ExecRunner struct{}

func (ExecRunner) Run(dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return out.String(), nil
}

func (ExecRunner) LookPath(name string) error {
	_, err := exec.LookPath(name)
	return err
}
