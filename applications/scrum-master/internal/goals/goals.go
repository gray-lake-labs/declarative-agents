// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package goals

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/cli"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/program"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/tracker"
)

// WebBase is the web root repositories are cited under, SCRUM_MASTER_WEB_BASE: a
// pinned citation is <web base>/<project>/-/blob/<tag>/<path> (srd007).
func WebBase() (string, error) {
	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("SCRUM_MASTER_WEB_BASE")), "/")
	if base == "" {
		return "", errors.New("no web base for citations: set SCRUM_MASTER_WEB_BASE to the web root repositories are cited under")
	}
	return base, nil
}

func init() {
	var a struct{ system, release, ref, planFile, control, dir string }
	common := func(fs *flag.FlagSet) {
		fs.StringVar(&a.system, "system", "", "system repository path (required)")
		fs.StringVar(&a.release, "release", "", "road-map release version (required)")
		fs.StringVar(&a.ref, "ref", "main", "ref of the road map, docs/road-map.md")
		fs.StringVar(&a.dir, "dir", ".", "the checkout the tracker is routed from, as the other commands a step runs from a checkout take it")
	}
	load := func(ctx *cli.Context) (tracker.Tracker, Sources, error) {
		if a.system == "" || a.release == "" {
			return nil, Sources{}, fmt.Errorf("%w: needs --system and --release", cli.ErrUsage)
		}
		// goals reads the corpus through the tracker, which scrum-master routes from
		// the working directory.
		if a.dir != "." {
			if err := os.Chdir(a.dir); err != nil {
				return nil, Sources{}, err
			}
		}
		t, err := ctx.Tracker()
		if err != nil {
			return nil, Sources{}, err
		}
		s, err := Load(t, a.system, a.release, a.ref)
		return t, s, err
	}
	cli.Default.Register(&cli.Command{
		Group: "goals", Name: "plan", Summary: "initiative outcomes of a release with recommended team and environment",
		Flags: func(fs *flag.FlagSet) {
			common(fs)
			// plan reads no control files; it takes --control so the one
			// flag set serves plan and push, as AGENTS.md asks (GH-20).
			fs.StringVar(&a.control, "control", "", "program repository holding control/ files; plan reads none, and accepts it so one flag set serves plan and push")
		},
		Run: func(ctx *cli.Context) error {
			t, s, err := load(ctx)
			if err != nil {
				return err
			}
			p, err := MakePlan(t, s)
			if err != nil {
				return err
			}
			var h strings.Builder
			for _, o := range p.Objectives {
				line := fmt.Sprintf("%s %q items=%s -> repository=%q environment=%s due=%s existing=%s", o.Key, o.Title, strings.Join(o.Items, ","), o.Repository, o.Environment, o.Due, o.Existing)
				if o.Carried != "" {
					line += " carried by " + o.Carried
				}
				fmt.Fprintln(&h, line)
			}
			return ctx.Emit(p, h.String())
		},
	})
	cli.Default.Register(&cli.Command{
		Group: "goals", Name: "push", Summary: "file one Sprint Goal per team and post planned events",
		Flags: func(fs *flag.FlagSet) {
			common(fs)
			fs.StringVar(&a.planFile, "plan", "", "confirmed plan JSON from scrum-master goals plan (required)")
			fs.StringVar(&a.control, "control", "", "program repository holding control/ files; defaults to --system")
		},
		Run: func(ctx *cli.Context) error {
			if a.planFile == "" {
				return fmt.Errorf("%w: push needs --plan", cli.ErrUsage)
			}
			raw, err := os.ReadFile(a.planFile)
			if err != nil {
				return err
			}
			var p Plan
			if err := json.Unmarshal(raw, &p); err != nil {
				return fmt.Errorf("%s: %w", a.planFile, err)
			}
			t, s, err := load(ctx)
			if err != nil {
				return err
			}
			if p.Release != s.Release.Version || p.Tag != s.Release.Tag {
				return fmt.Errorf("plan is for %s at %s, the road-map has %s at %s; run scrum-master goals plan again", p.Release, p.Tag, s.Release.Version, s.Release.Tag)
			}
			control := a.control
			if control == "" {
				control = a.system
			}
			policy, err := program.LoadPolicy(t, control, a.ref)
			if err != nil {
				return err
			}
			web, err := WebBase()
			if err != nil {
				return err
			}
			res, err := Push(t, s, p, web, policy, ctx.DryRun)
			if err != nil {
				return err
			}
			var h strings.Builder
			for _, r := range res {
				fmt.Fprintln(&h, pushedLine(r))
				for _, w := range r.Warnings {
					fmt.Fprintf(&h, "  warning: %s\n", w)
				}
			}
			return ctx.Emit(res, h.String())
		},
	})
}

// pushedLine is goals push's human line for one outcome: its key, what push
// did, the Sprint Goal as project#iid with its web address when the tracker
// has one, and any reasons.
func pushedLine(r Pushed) string {
	parts := []string{r.Key + ":", r.Action}
	if r.IID != 0 {
		parts = append(parts, fmt.Sprintf("%s#%d", r.Project, r.IID))
	}
	if strings.Contains(r.WebURL, "://") {
		parts = append(parts, r.WebURL)
	}
	if len(r.Reasons) > 0 {
		parts = append(parts, strings.Join(r.Reasons, "; "))
	}
	return strings.Join(parts, " ")
}
