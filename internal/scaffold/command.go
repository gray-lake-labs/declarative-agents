package scaffold

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/cli"
	"github.com/gray-lake-labs/scrum-master/internal/install"
)

func init() {
	var (
		o      InitOptions
		agents []string
	)
	cli.Default.Register(&cli.Command{
		Group:   "init",
		Summary: "lay down a program's system repository: corpus, control files, skills",
		Flags: func(fs *flag.FlagSet) {
			o, agents = InitOptions{}, nil
			fs.StringVar(&o.Dir, "dir", "", "the system repository's checkout (default: the git root)")
			fs.Func("agent", "install the skills for cursor, claude, copilot, or all (repeatable)", func(v string) error {
				agents = append(agents, v)
				return nil
			})
			fs.StringVar(&o.Tracker, "tracker", "", "a tracker repository to seed when it has no commits")
			fs.StringVar(&o.Group, "group", "", "the program group whose labels --tracker gets")
			fs.BoolVar(&o.Force, "force", false, "replace files a person edited")
		},
		Run: func(ctx *cli.Context) error {
			var err error
			if len(agents) > 0 {
				if o.Agents, err = install.ParseAgents(agents); err != nil {
					return fmt.Errorf("%w: %v", cli.ErrUsage, err)
				}
			}
			if o.Dir == "" {
				if o.Dir, err = gitRoot(); err != nil {
					return err
				}
			}
			o.Version, o.DryRun = install.Version(), ctx.DryRun
			if o.Tracker != "" {
				if o.User, err = person(); err != nil {
					return err
				}
			}
			r, err := Init(o)
			if err != nil {
				return err
			}
			return ctx.Emit(r, humanInit(r))
		},
	})
}

func humanInit(r InitResult) string {
	var b strings.Builder
	for _, f := range r.Files {
		fmt.Fprintf(&b, "%-12s %s", f.Action, f.Path)
		if f.Reason != "" && f.Reason != "unchanged" {
			fmt.Fprintf(&b, " (%s)", f.Reason)
		}
		b.WriteString("\n")
	}
	if len(r.Skills) > 0 {
		fmt.Fprintf(&b, "skills: %d files for the named agents\n", len(r.Skills))
	}
	if t := r.Tracker; t != nil {
		switch {
		case t.Seeded:
			fmt.Fprintf(&b, "tracker %s: first commit made, %d labels\n", t.Remote, len(t.Labels))
		default:
			fmt.Fprintf(&b, "tracker %s: %s\n", t.Remote, t.Reason)
		}
	}
	if r.DryRun {
		b.WriteString("dry run: nothing written\n")
	}
	return b.String()
}

func gitRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repository; name one with --dir")
	}
	return strings.TrimSpace(string(out)), nil
}

// person is who the tracker's writes are made as, as the beads tracker
// resolves it: TPM_USER, else git's user.name.
func person() (string, error) {
	if u := strings.TrimSpace(os.Getenv("TPM_USER")); u != "" {
		return u, nil
	}
	out, err := exec.Command("git", "config", "user.name").Output()
	if u := strings.TrimSpace(string(out)); err == nil && u != "" {
		return u, nil
	}
	return "", fmt.Errorf("no person for the tracker's writes: set TPM_USER or git config user.name")
}
