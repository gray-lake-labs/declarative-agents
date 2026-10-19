package sprintend

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/cli"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/release"
	"github.com/gray-lake-labs/scrum-master/internal/reports"
	"github.com/gray-lake-labs/scrum-master/internal/sprintgoal"
)

func init() {
	var a struct{ project, reportsDir, from, to, dir, artifact, control, ref string }
	cli.Default.Register(&cli.Command{
		Group: "sprint-end", Name: "facts", Summary: "the period's aggregate, each Sprint Goal's ledger and reported status, and the recommended version",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&a.project, "project", "", "the repository the sprint-end issue is in (required)")
			fs.StringVar(&a.reportsDir, "reports", "", "a checkout of the reports repository (required)")
			fs.StringVar(&a.from, "from", "", "the period's first week, YYYY-Www (required)")
			fs.StringVar(&a.to, "to", "", "the period's last week, YYYY-Www (required)")
			fs.StringVar(&a.dir, "dir", ".", "the repository's checkout, for its release tags")
			fs.StringVar(&a.artifact, "artifact", "", "the artifact the repository releases (required)")
			fs.StringVar(&a.control, "control", "", "program repository holding control/policy.md; empty uses the defaults")
			fs.StringVar(&a.ref, "ref", "main", "ref of the control files")
		},
		Run: func(ctx *cli.Context) error {
			if a.project == "" || a.reportsDir == "" || a.from == "" || a.to == "" || a.artifact == "" {
				return fmt.Errorf("%w: facts needs --project, --reports, --from, --to, and --artifact", cli.ErrUsage)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			p := program.Defaults()
			if a.control != "" {
				if p, err = program.LoadPolicy(t, a.control, a.ref); err != nil {
					return err
				}
			}
			cfg, err := reports.LoadConfig(a.reportsDir)
			if err != nil {
				return err
			}
			agg, err := reports.Aggregate(t, reports.AggregateInput{Dir: a.reportsDir, Config: cfg, Policy: p, From: a.from, To: a.to})
			if err != nil {
				return err
			}
			tags, err := release.RepoTags(a.dir)
			if err != nil {
				return err
			}
			f, err := MakeFacts(Input{Tracker: t, Project: a.project, Aggregate: agg, Releases: p.Releases, Artifact: a.artifact, Tags: tags})
			if err != nil {
				return err
			}
			var h strings.Builder
			fmt.Fprintf(&h, "%s, %s to %s: %d reports, %d signals; recommended version %s\n", f.Project, a.from, a.to, agg.Reports, len(agg.Signals), f.Recommended)
			for _, o := range f.SprintGoals {
				fmt.Fprintf(&h, "  %s %s (%s): %s, reported %q, complete=%v, met=%v\n", o.Issue, strings.Join(o.Serves, ","), o.Release, o.State, o.Reported, o.ReportedComplete, o.Met)
			}
			return ctx.Emit(f, h.String())
		},
	})

	var st struct{ dir, release, done string }
	cli.Default.Register(&cli.Command{
		Group: "sprint-end", Name: "stories", Summary: "record which of the Sprint Backlog's stories the owner reports done in the sprint-end summary",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&st.dir, "dir", ".", "the sprint-end issue's worktree")
			fs.StringVar(&st.release, "release", "", "the release whose summary records them (required)")
			fs.StringVar(&st.done, "done", "", "the done stories' ids, comma-separated; empty records none")
		},
		Run: func(ctx *cli.Context) error {
			if st.release == "" {
				return fmt.Errorf("%w: stories needs --release", cli.ErrUsage)
			}
			var ids []string
			for _, id := range strings.Split(st.done, ",") {
				if id = strings.TrimSpace(id); id != "" {
					ids = append(ids, id)
				}
			}
			path := filepath.Join(st.dir, sprintgoal.SummaryDir, st.release+".md")
			changed, err := sprintgoal.RecordStoriesDone(path, ids, ctx.DryRun)
			if err != nil {
				return err
			}
			res := struct {
				Path    string   `json:"path"`
				Done    []string `json:"done"`
				Changed bool     `json:"changed"`
			}{path, ids, changed}
			return ctx.Emit(res, fmt.Sprintf("%s: %d stories done, changed=%v", path, len(ids), changed))
		},
	})
}
