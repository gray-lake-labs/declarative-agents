package goals

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/cli"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/program"
)

// WebBase is the instance's web root, derived from GITLAB_BASE_URL.
func WebBase() string { return strings.TrimSuffix(gitlab.BaseURL(), "/api/v4") }

func init() {
	var a struct{ system, release, ref, planFile, control string }
	common := func(fs *flag.FlagSet) {
		fs.StringVar(&a.system, "system", "", "system repository path (required)")
		fs.StringVar(&a.release, "release", "", "road-map release version (required)")
		fs.StringVar(&a.ref, "ref", "main", "ref of the road map, docs/road-map.md")
	}
	load := func(ctx *cli.Context) (gitlab.Tracker, Sources, error) {
		if a.system == "" || a.release == "" {
			return nil, Sources{}, fmt.Errorf("%w: needs --system and --release", cli.ErrUsage)
		}
		t, err := ctx.Tracker()
		if err != nil {
			return nil, Sources{}, err
		}
		s, err := Load(t, a.system, a.release, a.ref)
		return t, s, err
	}
	cli.Default.Register(&cli.Command{
		Group: "goals", Name: "plan", Summary: "objectives of a release with recommended owner and environment",
		Flags: common,
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
				fmt.Fprintf(&h, "%s %q items=%s -> repository=%q environment=%s due=%s existing=%s\n", o.Key, o.Title, strings.Join(o.Items, ","), o.Repository, o.Environment, o.Due, o.Existing)
			}
			return ctx.Emit(p, h.String())
		},
	})
	cli.Default.Register(&cli.Command{
		Group: "goals", Name: "push", Summary: "file the confirmed objective issues and post planned events",
		Flags: func(fs *flag.FlagSet) {
			common(fs)
			fs.StringVar(&a.planFile, "plan", "", "confirmed plan JSON from tpm goals plan (required)")
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
				return fmt.Errorf("plan is for %s at %s, the road-map has %s at %s; run tpm goals plan again", p.Release, p.Tag, s.Release.Version, s.Release.Tag)
			}
			control := a.control
			if control == "" {
				control = a.system
			}
			policy, err := program.LoadPolicy(t, control, a.ref)
			if err != nil {
				return err
			}
			res, err := Push(t, s, p, WebBase(), policy, ctx.DryRun)
			if err != nil {
				return err
			}
			var h strings.Builder
			for _, r := range res {
				fmt.Fprintf(&h, "%s: %s %s %s\n", r.Key, r.Action, r.WebURL, strings.Join(r.Reasons, "; "))
			}
			return ctx.Emit(res, h.String())
		},
	})
}
