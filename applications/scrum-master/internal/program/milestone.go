// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package program

import (
	"flag"
	"fmt"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/cli"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/tracker"
)

// SetMilestone puts a filed issue in the milestone named title and lints it
// again, so a data issue refused for lacking a sprint milestone is admitted
// (srd001 R7, srd013 R1). The milestone must be one the project finds, in
// its own group or an ancestor.
func SetMilestone(t tracker.Tracker, project string, iid int, title string, p Policy, dryRun bool) (LintResult, error) {
	if _, err := t.FindMilestone(project, title); err != nil {
		return LintResult{}, fmt.Errorf("%s finds no milestone %s in its group or an ancestor: %w", project, title, err)
	}
	i, err := t.GetIssue(project, iid)
	if err != nil {
		return LintResult{}, err
	}
	if i.Milestone != title {
		i.Milestone = title
		if !dryRun {
			if i, err = t.UpdateIssue(project, iid, tracker.IssueUpdate{Milestone: &title}); err != nil {
				return LintResult{}, err
			}
		}
	}
	return Apply(t, i, Lint(i, p), p, dryRun)
}

func init() {
	var a struct {
		project, milestone, control, ref string
		iid                              int
	}
	cli.Default.Register(&cli.Command{
		Group: "issue", Name: "milestone", Summary: "put a filed issue in a milestone and lint it again",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&a.project, "project", "", "project path (required)")
			fs.IntVar(&a.iid, "issue", 0, "issue IID (required)")
			fs.StringVar(&a.milestone, "milestone", "", "milestone title (required)")
			fs.StringVar(&a.control, "control", "", "program repository holding control/ files; empty uses the defaults")
			fs.StringVar(&a.ref, "ref", "main", "ref of the control files")
		},
		Run: func(ctx *cli.Context) error {
			if a.project == "" || a.iid == 0 || a.milestone == "" {
				return fmt.Errorf("%w: milestone needs --project, --issue, and --milestone", cli.ErrUsage)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			p, err := LoadPolicy(t, a.control, a.ref)
			if err != nil {
				return err
			}
			r, err := SetMilestone(t, a.project, a.iid, a.milestone, p, ctx.DryRun)
			if err != nil {
				return err
			}
			return ctx.Emit(r, fmt.Sprintf("%s#%d in %s: %s", a.project, a.iid, a.milestone, verdict(r)))
		},
	})
}
