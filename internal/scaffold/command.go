package scaffold

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"slices"
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
			fs.StringVar(&o.Prefix, "prefix", "", "the issue ID prefix --tracker gets (default: the last segment of --group)")
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

func init() {
	var (
		o      OnboardOptions
		agents []string
	)
	cli.Default.Register(&cli.Command{
		Group:   "onboard",
		Summary: "add a repository to a program: its corpus here, its registration in the system repository",
		Flags: func(fs *flag.FlagSet) {
			o, agents = OnboardOptions{}, nil
			fs.StringVar(&o.Dir, "dir", "", "the joining repository's checkout (default: the git root)")
			fs.Func("agent", "install the skills for cursor, claude, copilot, or all (repeatable)", func(v string) error {
				agents = append(agents, v)
				return nil
			})
			fs.StringVar(&o.System, "system", "", "the system repository the registration is filed in (required)")
			fs.StringVar(&o.Ref, "ref", "main", "ref of the system repository's SRDs and control files")
			fs.StringVar(&o.Subsystem, "subsystem", "", "the subsystem SRD that lists the repository, e.g. srd010-ingest (required)")
			fs.StringVar(&o.Repository, "repository", "", "the repository's URL (default: its origin)")
			fs.StringVar(&o.Path, "path", "/", "the component's path in the repository")
			fs.BoolVar(&o.Force, "force", false, "replace files a person edited")
		},
		Run: func(ctx *cli.Context) error {
			var err error
			if o.System == "" || o.Subsystem == "" {
				return fmt.Errorf("%w: onboard needs --system and --subsystem", cli.ErrUsage)
			}
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
			if o.Repository == "" {
				out, err := exec.Command("git", "-C", o.Dir, "remote", "get-url", "origin").Output()
				if o.Repository = strings.TrimSpace(string(out)); err != nil || o.Repository == "" {
					return fmt.Errorf("%s has no origin; name the repository's URL with --repository", o.Dir)
				}
			}
			o.Version, o.DryRun = install.Version(), ctx.DryRun
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			r, err := Onboard(t, o)
			if err != nil {
				return err
			}
			human := humanFiles(r.Files)
			switch reg := r.Registration; {
			case reg.Issue != nil && reg.Filed:
				human += fmt.Sprintf("registration filed: %s#%d\n", reg.Issue.Project, reg.Issue.IID)
			case reg.Issue != nil:
				human += "dry run: would file the registration\n"
			default:
				human += "no registration filed: " + reg.Reason + "\n"
			}
			return ctx.Emit(r, human)
		},
	})
}

func init() {
	var o DoctorOptions
	cli.Default.Register(&cli.Command{
		Group:   "doctor",
		Summary: "bring a program repository up to date after an upgrade of tpm: missing files, tpm's text, document format, installed skills",
		Flags: func(fs *flag.FlagSet) {
			o = DoctorOptions{}
			fs.StringVar(&o.Dir, "dir", "", "the program repository's checkout (default: the git root)")
		},
		Run: func(ctx *cli.Context) error {
			var err error
			if o.Dir == "" {
				if o.Dir, err = gitRoot(); err != nil {
					return err
				}
			}
			o.Version, o.DryRun = install.Version(), ctx.DryRun
			r, err := Doctor(o)
			if err != nil {
				return err
			}
			return ctx.Emit(r, humanDoctor(r))
		},
	})
}

// humanDoctor names what doctor changed, or would change on a dry run, and
// the skills tpm install would update; a file left as it was is counted,
// not listed.
func humanDoctor(r DoctorResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s repository %s\n", r.Level, r.Dir)
	kept := 0
	for _, f := range r.Files {
		if f.Action == "kept" && !strings.HasPrefix(f.Reason, "does not parse") {
			kept++
			continue
		}
		fmt.Fprintf(&b, "%-12s %s", f.Action, f.Path)
		if f.Reason != "" {
			fmt.Fprintf(&b, " (%s)", f.Reason)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "%d files current\n", kept)
	verb := "repaired"
	if r.DryRun {
		verb = "would repair"
	}
	for _, p := range r.Repair.Repaired {
		fmt.Fprintf(&b, "%-12s %s\n", verb, p)
	}
	for _, f := range r.Repair.Findings {
		fmt.Fprintf(&b, "finding      %s\n", f)
	}
	if len(r.Repair.Untracked) > 0 {
		fmt.Fprintf(&b, "not checked, git does not track them (git add them first): %s\n", strings.Join(r.Repair.Untracked, ", "))
	}
	if tr := r.Trackers; tr != nil {
		fmt.Fprintf(&b, "tracking systems (%s): default %s", tr.File, tr.Default.Name)
		for _, p := range slices.Sorted(maps.Keys(tr.Overrides)) {
			fmt.Fprintf(&b, ", %s %s", p, tr.Overrides[p])
		}
		b.WriteString("\n")
		for _, problem := range tr.Problems {
			fmt.Fprintf(&b, "  %s\n", problem)
		}
	}
	if r.Repair.Skipped != "" {
		fmt.Fprintf(&b, "repair skipped: %s\n", r.Repair.Skipped)
	}
	// The runtime's tag is the parent release tpm pins, not a tpm version,
	// though both read v0.YYYYMMDD.N (#188).
	runtime := fmt.Sprintf("runtime (declarative-agents %s, the release this tpm pins)", r.Runtime.Release)
	if r.Runtime.Problem != "" {
		fmt.Fprintf(&b, "%s: %s\n", runtime, r.Runtime.Problem)
	} else {
		fmt.Fprintf(&b, "%s: %s (%s)\n", runtime, r.Runtime.Agent, r.Runtime.From)
	}
	for _, p := range r.Credentials {
		var vars []string
		for _, v := range p.Vars {
			state := "set"
			if !v.Set {
				state = "unset"
			}
			vars = append(vars, v.Name+" ("+state+")")
		}
		fmt.Fprintf(&b, "profile %s reads: %s\n", p.Profile, strings.Join(vars, ", "))
	}
	agents := map[string]bool{}
	for _, f := range r.Skills {
		fmt.Fprintf(&b, "skill %-9s %s %s\n", f.Problem, f.Agent, f.Path)
		agents[f.Agent] = true
	}
	for _, a := range install.Agents {
		if agents[a] {
			fmt.Fprintf(&b, "run tpm install --agent %s to update its skills\n", a)
		}
	}
	if r.DryRun {
		b.WriteString("dry run: nothing written\n")
	}
	return b.String()
}

func humanFiles(files []Result) string {
	var b strings.Builder
	for _, f := range files {
		fmt.Fprintf(&b, "%-12s %s", f.Action, f.Path)
		if f.Reason != "" && f.Reason != "unchanged" {
			fmt.Fprintf(&b, " (%s)", f.Reason)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func humanInit(r InitResult) string {
	var b strings.Builder
	b.WriteString(humanFiles(r.Files))
	if len(r.Skills) > 0 {
		fmt.Fprintf(&b, "skills: %d files for the named agents\n", len(r.Skills))
	}
	if t := r.Tracker; t != nil {
		switch {
		case t.Seeded:
			fmt.Fprintf(&b, "tracker %s: first commit made, issue prefix %s, %d labels\n", t.Remote, t.Prefix, len(t.Labels))
		case t.Remote == "":
			fmt.Fprintf(&b, "tracker: %s\n", t.Reason)
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
