package plan

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/cli"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/goals"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/release"
	"github.com/gray-lake-labs/scrum-master/internal/reports"
)

func init() {
	var a struct{ dir, system, aggregate, artifact, control, ref string }
	common := func(fs *flag.FlagSet) {
		fs.StringVar(&a.dir, "dir", ".", "the system repository's checkout")
		fs.StringVar(&a.system, "system", "", "system repository path, for slipped objectives and the last milestone")
		fs.StringVar(&a.aggregate, "aggregate", "", "the period's tpm reports aggregate --json output, for slipped objectives")
		fs.StringVar(&a.artifact, "artifact", "corpus", "the artifact the system repository releases its plan as")
		fs.StringVar(&a.control, "control", "", "program repository holding control/policy.md; defaults to --system")
		fs.StringVar(&a.ref, "ref", "main", "ref of the control files")
	}
	candidates := func(ctx *cli.Context) (Candidates, gitlab.Tracker, error) {
		in := Input{Dir: a.dir, System: a.system, Artifact: a.artifact, Now: time.Now(), Releases: program.Defaults().Releases}
		if a.system != "" {
			t, err := ctx.Tracker()
			if err != nil {
				return Candidates{}, nil, err
			}
			in.Tracker = t
			control := a.control
			if control == "" {
				control = a.system
			}
			p, err := program.LoadPolicy(t, control, a.ref)
			if err != nil {
				return Candidates{}, nil, err
			}
			in.Releases = p.Releases
		}
		if a.aggregate != "" {
			b, err := os.ReadFile(a.aggregate)
			if err != nil {
				return Candidates{}, nil, err
			}
			var agg reports.Aggregation
			if err := json.Unmarshal(b, &agg); err != nil {
				return Candidates{}, nil, fmt.Errorf("%s: %w", a.aggregate, err)
			}
			in.Aggregate = &agg
		}
		tags, err := release.RepoTags(a.dir)
		if err != nil {
			return Candidates{}, nil, err
		}
		in.Tags = tags
		c, err := MakeCandidates(in)
		return c, in.Tracker, err
	}

	cli.Default.Register(&cli.Command{
		Group: "plan", Name: "candidates", Summary: "rank the objective items the next release could prove, with the recommended version, end date, and tag",
		Flags: common,
		Run: func(ctx *cli.Context) error {
			c, _, err := candidates(ctx)
			if err != nil {
				return err
			}
			var h strings.Builder
			fmt.Fprintf(&h, "next release %s, ending %s, tagged %s\n", c.Version, c.EndDate, c.Tag)
			for _, it := range c.Items {
				flag := ""
				if it.Slipped {
					flag = " (slipped)"
				}
				fmt.Fprintf(&h, "  %s%s due %s: %s\n", it.ID, flag, it.Due, it.Text)
			}
			for _, it := range c.WithoutEvidence {
				fmt.Fprintf(&h, "  held back %s: missing %s\n", it.ID, strings.Join(it.Missing, " and "))
			}
			return ctx.Emit(c, h.String())
		},
	})

	var pu struct{ version, end, group string }
	cli.Default.Register(&cli.Command{
		Group: "plan", Name: "push", Summary: "date the milestone and sprint issue, post planned events, and file the period's sprint-end issues",
		Flags: func(fs *flag.FlagSet) {
			common(fs)
			fs.StringVar(&pu.version, "version", "", "the road-map release to push (required)")
			fs.StringVar(&pu.end, "end", "", "the period's end date, YYYY-MM-DD (required)")
			fs.StringVar(&pu.group, "group", "", "the program group the milestone lives on; defaults to the system repository's group")
		},
		Run: func(ctx *cli.Context) error {
			if a.system == "" || pu.version == "" || pu.end == "" {
				return fmt.Errorf("%w: push needs --system, --version, and --end", cli.ErrUsage)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			control := a.control
			if control == "" {
				control = a.system
			}
			p, err := program.LoadPolicy(t, control, a.ref)
			if err != nil {
				return err
			}
			group := pu.group
			if group == "" {
				group = path.Dir(program.ProjectPath(a.system))
			}
			web, err := goals.WebBase()
			if err != nil {
				return err
			}
			res, err := Push(t, PushInput{Dir: a.dir, System: a.system, Group: group, Version: pu.version, End: pu.end,
				WebBase: web, Policy: p, Ref: a.ref, DryRun: ctx.DryRun})
			if err != nil {
				return err
			}
			var h strings.Builder
			fmt.Fprintf(&h, "milestone %s: %s\nsprint issue %s#%d: %s\n", res.Release.Milestone, res.Milestone, res.Sprint.Project, res.Sprint.IID, res.Sprint.Action)
			for _, s := range res.SprintEnds {
				fmt.Fprintf(&h, "sprint-end issue %s#%d: %s\n", s.Project, s.IID, s.Action)
			}
			for _, e := range res.Planned {
				fmt.Fprintf(&h, "planned event on %s: posted=%v\n", e.Issue, e.Posted)
			}
			for _, s := range res.Skipped {
				fmt.Fprintf(&h, "skipped %s\n", s)
			}
			return ctx.Emit(res, h.String())
		},
	})

	var w struct{ version, milestone, tag, items, issue string }
	cli.Default.Register(&cli.Command{
		Group: "plan", Name: "write", Summary: "write the chosen release into docs/road-map.md in the checkout",
		Flags: func(fs *flag.FlagSet) {
			common(fs)
			fs.StringVar(&w.version, "version", "", "the release's version, such as 01.2 (required)")
			fs.StringVar(&w.milestone, "milestone", "", "the release's milestone; defaults to the version")
			fs.StringVar(&w.tag, "tag", "", "the tag that publishes the plan (required)")
			fs.StringVar(&w.items, "items", "", "comma-separated objective items, objNNN-short-name#Rn.m (required)")
			fs.StringVar(&w.issue, "issue", "", "planning issue as project#iid, to comment on items held back")
		},
		Run: func(ctx *cli.Context) error {
			if w.version == "" || w.tag == "" || w.items == "" {
				return fmt.Errorf("%w: write needs --version, --tag, and --items", cli.ErrUsage)
			}
			c, t, err := candidates(ctx)
			if err != nil {
				return err
			}
			r := Release{Version: w.version, Milestone: w.milestone, Tag: w.tag, Items: strings.Split(w.items, ",")}
			if r.Milestone == "" {
				r.Milestone = r.Version
			}
			var project string
			var iid int
			if w.issue != "" {
				if project, iid, err = release.ParseIssueRef(w.issue); err != nil {
					return fmt.Errorf("%w: %v", cli.ErrUsage, err)
				}
				if t == nil {
					if t, err = ctx.Tracker(); err != nil {
						return err
					}
				}
			}
			if ctx.DryRun {
				t = nil
			}
			res, err := WriteRelease(a.dir, r, c, t, project, iid)
			if err != nil {
				return err
			}
			h := fmt.Sprintf("%s release %s in %s: %s\n", res.Action, r.Version, res.Path, strings.Join(res.Release.Items, ", "))
			for _, x := range res.Returned {
				h += fmt.Sprintf("returned %s: missing %s\n", x.ID, strings.Join(x.Missing, " and "))
			}
			return ctx.Emit(res, h)
		},
	})
}
