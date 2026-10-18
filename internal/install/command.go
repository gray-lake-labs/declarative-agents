package install

import (
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/gray-lake-labs/scrum-master"
	"github.com/gray-lake-labs/scrum-master/internal/cli"
)

// Version is the tpm version stamped into installed files: the module
// version the binary was built at, or dev for a local build.
func Version() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "dev"
}

func gitRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repository; name one with --dir")
	}
	return strings.TrimSpace(string(out)), nil
}

func rel(dir, path string) string {
	if r, err := filepath.Rel(dir, path); err == nil {
		return filepath.ToSlash(r)
	}
	return path
}

func init() {
	var (
		agents             []string
		dir                string
		list, check, force bool
	)
	cli.Default.Register(&cli.Command{
		Group:   "install",
		Summary: "write the embedded skill commands for an agent",
		Flags: func(fs *flag.FlagSet) {
			agents = nil
			fs.Func("agent", "cursor, claude, copilot, or all (repeatable)", func(v string) error {
				agents = append(agents, v)
				return nil
			})
			fs.StringVar(&dir, "dir", "", "repository to install into (default: the git root)")
			fs.BoolVar(&list, "list", false, "name the embedded skills and write nothing")
			fs.BoolVar(&check, "check", false, "report installed files that differ from the binary; write nothing")
			fs.BoolVar(&force, "force", false, "replace files a person edited")
		},
		Run: func(ctx *cli.Context) error {
			skills, err := Skills(scrummaster.Plugin, scrummaster.SkillsDir)
			if err != nil {
				return err
			}
			if list {
				var b strings.Builder
				for _, s := range skills {
					fmt.Fprintf(&b, "%-20s %s\n", s.Name, s.Description)
				}
				return ctx.Emit(skills, b.String())
			}
			targets, err := ParseAgents(agents)
			if err != nil {
				return fmt.Errorf("%w: %v", cli.ErrUsage, err)
			}
			if dir == "" {
				if dir, err = gitRoot(); err != nil {
					return err
				}
			}
			version := Version()
			if check {
				findings, err := Check(dir, targets, skills, version)
				if err != nil {
					return err
				}
				var b strings.Builder
				for _, f := range findings {
					fmt.Fprintf(&b, "%-9s %s\n", f.Problem, rel(dir, f.Path))
				}
				if len(findings) == 0 {
					b.WriteString("installed skills match tpm " + version)
				}
				if err := ctx.Emit(findings, b.String()); err != nil {
					return err
				}
				if len(findings) > 0 {
					return fmt.Errorf("%d installed skill file(s) differ from tpm %s; run tpm install", len(findings), version)
				}
				return nil
			}
			results, err := Install(dir, targets, skills, version, force, ctx.DryRun)
			if err != nil {
				return err
			}
			var b strings.Builder
			for _, r := range results {
				fmt.Fprintf(&b, "%-9s %s", r.Action, rel(dir, r.Path))
				if r.Reason != "" {
					fmt.Fprintf(&b, " (%s)", r.Reason)
				}
				b.WriteString("\n")
			}
			return ctx.Emit(results, b.String())
		},
	})
}
