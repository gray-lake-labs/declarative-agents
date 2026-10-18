// Package program is the tpm program group: the issue operations every
// repository shares, admission and filing (srd013).
package program

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/cli"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
)

func init() {
	var labelArgs struct {
		group, control, ref string
		update              bool
	}
	cli.Default.Register(&cli.Command{
		Group: "program", Name: "labels", Summary: "create the program's labels from control/labels.md or the defaults",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&labelArgs.group, "group", "", "program group path (required)")
			fs.StringVar(&labelArgs.control, "control", "", "program repository holding control/labels.md; empty uses the defaults")
			fs.StringVar(&labelArgs.ref, "ref", "main", "ref of control/labels.md")
			fs.BoolVar(&labelArgs.update, "update", false, "also rewrite the color and description of labels that differ")
		},
		Run: func(ctx *cli.Context) error {
			if labelArgs.group == "" {
				return fmt.Errorf("%w: labels needs --group", cli.ErrUsage)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			want, err := LoadLabels(t, labelArgs.control, labelArgs.ref)
			if err != nil {
				return err
			}
			res, err := SyncLabels(t, labelArgs.group, want, labelArgs.update, ctx.DryRun)
			if err != nil {
				return err
			}
			counts := map[string]int{}
			for _, r := range res {
				counts[r.Action]++
			}
			human := fmt.Sprintf("%s: %d created, %d updated, %d unchanged, %d differ", labelArgs.group,
				counts["created"], counts["updated"], counts["unchanged"], counts["differs"])
			return ctx.Emit(res, human)
		},
	})

	var popArgs struct {
		project, dir, control, ref string
		iid                        int
	}
	cli.Default.Register(&cli.Command{
		Group: "program", Name: "pop", Summary: "claim an issue and open its own branch and worktree",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&popArgs.project, "project", "", "project path (required)")
			fs.IntVar(&popArgs.iid, "issue", 0, "issue IID (required)")
			fs.StringVar(&popArgs.dir, "dir", ".", "a directory in the local checkout of the project")
			fs.StringVar(&popArgs.control, "control", "", "program repository holding control/ files; empty uses the defaults")
			fs.StringVar(&popArgs.ref, "ref", "main", "ref of the control files")
		},
		Run: func(ctx *cli.Context) error {
			if popArgs.project == "" || popArgs.iid == 0 {
				return fmt.Errorf("%w: pop needs --project and --issue", cli.ErrUsage)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			p, err := LoadPolicy(t, popArgs.control, popArgs.ref)
			if err != nil {
				return err
			}
			r, err := Pop(t, ExecGit{}, popArgs.project, popArgs.iid, popArgs.dir, p, ctx.DryRun)
			if err != nil {
				return err
			}
			human := fmt.Sprintf("%s#%d on %s in %s", r.Project, r.IID, r.Branch, r.Worktree)
			if !ctx.DryRun {
				if r.Installed, err = InstallSkills(r.Checkout, r.Worktree); err != nil {
					r.InstallError = err.Error()
					human += "\nskills not installed in the worktree: " + r.InstallError
				} else if len(r.Installed) > 0 {
					human += "\nskills installed in the worktree for " + strings.Join(r.Installed, ", ")
				}
			}
			return ctx.Emit(r, human)
		},
	})

	var mrArgs struct {
		project, issueProject, branch, audit, dir, control, ref string
		iid                                                     int
		closes                                                  cli.Strings
	}
	cli.Default.Register(&cli.Command{
		Group: "program", Name: "open-merge-request", Summary: "open the merge request that closes an issue",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&mrArgs.project, "project", "", "project path (required)")
			fs.IntVar(&mrArgs.iid, "issue", 0, "issue IID (required)")
			fs.StringVar(&mrArgs.issueProject, "issue-project", "", "project holding the issue, when not --project")
			fs.StringVar(&mrArgs.branch, "branch", "", "source branch; empty uses the current branch of the working directory")
			fs.StringVar(&mrArgs.audit, "audit", "", "corpus gate command for a docs/ change; empty uses the policy's audit_command (mage audit)")
			fs.StringVar(&mrArgs.dir, "dir", ".", "the worktree holding the branch, where the audit runs")
			fs.StringVar(&mrArgs.control, "control", "", "program repository holding control/ files; empty uses the defaults")
			fs.StringVar(&mrArgs.ref, "ref", "main", "ref of the control files")
			fs.Var(&mrArgs.closes, "closes", "another issue the merge closes, project#iid; repeat for each (srd038 R2.1)")
		},
		Run: func(ctx *cli.Context) error {
			if mrArgs.project == "" || mrArgs.iid == 0 {
				return fmt.Errorf("%w: open-merge-request needs --project and --issue", cli.ErrUsage)
			}
			branch := mrArgs.branch
			if branch == "" {
				out, err := ExecGit{}.Run(mrArgs.dir, "rev-parse", "--abbrev-ref", "HEAD")
				if err != nil {
					return err
				}
				branch = strings.TrimSpace(out)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			p, err := LoadPolicy(t, mrArgs.control, mrArgs.ref)
			if err != nil {
				return err
			}
			gate := AuditGate{Command: p.AuditCommand, Dir: mrArgs.dir}
			if mrArgs.audit != "" {
				gate.Command = mrArgs.audit
			}
			issueProject := mrArgs.issueProject
			if issueProject == "" {
				issueProject = mrArgs.project
			}
			r, err := OpenMergeRequestClosing(t, mrArgs.project, issueProject, mrArgs.iid, branch, gate, mrArgs.closes, ctx.DryRun)
			if err != nil {
				return err
			}
			return ctx.Emit(r, fmt.Sprintf("%s!%d %s (closes #%d)", r.Project, r.MR, r.WebURL, r.IID))
		},
	})

	var lintArgs struct {
		project, control, ref, since string
		iid                          int
	}
	cli.Default.Register(&cli.Command{
		Group: "program", Name: "lint", Summary: "check issues against the label schema and admission policy",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&lintArgs.project, "project", "", "project path (required)")
			fs.IntVar(&lintArgs.iid, "issue", 0, "issue IID; or use --since")
			fs.StringVar(&lintArgs.since, "since", "", "lint every issue updated since YYYY-MM-DD")
			fs.StringVar(&lintArgs.control, "control", "", "program repository holding control/ files; empty uses the defaults")
			fs.StringVar(&lintArgs.ref, "ref", "main", "ref of the control files")
		},
		Run: func(ctx *cli.Context) error {
			if lintArgs.project == "" || (lintArgs.iid == 0) == (lintArgs.since == "") {
				return fmt.Errorf("%w: lint needs --project and exactly one of --issue or --since", cli.ErrUsage)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			p, err := LoadPolicy(t, lintArgs.control, lintArgs.ref)
			if err != nil {
				return err
			}
			var issues []gitlab.Issue
			if lintArgs.iid != 0 {
				i, err := t.GetIssue(lintArgs.project, lintArgs.iid)
				if err != nil {
					return err
				}
				issues = []gitlab.Issue{i}
			} else if issues, err = t.ListIssues(lintArgs.project, gitlab.IssueQuery{State: "opened", UpdatedAfter: lintArgs.since}); err != nil {
				return err
			}
			var results []LintResult
			var human strings.Builder
			for _, i := range issues {
				r, err := Apply(t, i, Lint(i, p), p, ctx.DryRun)
				if err != nil {
					return err
				}
				results = append(results, r)
				fmt.Fprintf(&human, "%s#%d: %s\n", r.Project, r.IID, verdict(r))
			}
			return ctx.Emit(results, human.String())
		},
	})

	var fileArgs struct {
		project, title, bodyFile, plane, kind, typ, component, milestone, due, control, system, ref string
		paths                                                                                       []string
	}
	cli.Default.Register(&cli.Command{
		Group: "program", Name: "file", Summary: "file an unassigned issue and lint it",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&fileArgs.project, "project", "", "project path (required)")
			fs.StringVar(&fileArgs.title, "title", "", "issue title (required)")
			fs.StringVar(&fileArgs.bodyFile, "body-file", "", "file holding the description (required)")
			fs.StringVar(&fileArgs.plane, "plane", "", "plane: management, control, or data; derived from --path when empty")
			fs.Func("path", "a path the request changes (repeatable); plane and kind follow from it", func(v string) error {
				fileArgs.paths = append(fileArgs.paths, v)
				return nil
			})
			fs.StringVar(&fileArgs.system, "system", "", "system repository whose subsystem SRDs give the component; also the default --control")
			fs.StringVar(&fileArgs.kind, "kind", "", "kind, on data issues")
			fs.StringVar(&fileArgs.typ, "type", "", "type: epic, sprint, or recurring")
			fs.StringVar(&fileArgs.component, "component", "", "component")
			fs.StringVar(&fileArgs.milestone, "milestone", "", "milestone title")
			fs.StringVar(&fileArgs.due, "due", "", "due date YYYY-MM-DD")
			fs.StringVar(&fileArgs.control, "control", "", "program repository holding control/ files")
			fs.StringVar(&fileArgs.ref, "ref", "main", "ref of the control files")
		},
		Run: func(ctx *cli.Context) error {
			if fileArgs.project == "" || fileArgs.title == "" || fileArgs.bodyFile == "" {
				return fmt.Errorf("%w: file needs --project, --title, and --body-file", cli.ErrUsage)
			}
			body, err := os.ReadFile(fileArgs.bodyFile)
			if err != nil {
				return err
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			control := fileArgs.control
			if control == "" {
				control = fileArgs.system
			}
			p, err := LoadPolicy(t, control, fileArgs.ref)
			if err != nil {
				return err
			}
			l, err := Derive(t, fileArgs.project, fileArgs.system, fileArgs.ref, fileArgs.paths,
				Labelling{Plane: fileArgs.plane, Kind: fileArgs.kind, Component: fileArgs.component})
			if err != nil {
				return err
			}
			milestone, reason := fileArgs.milestone, ""
			if milestone == "" && l.Plane == "data" && fileArgs.typ == "" {
				if milestone, reason, err = SprintMilestone(t, fileArgs.project, time.Now()); err != nil {
					return err
				}
			}
			in := gitlab.NewIssue{Title: fileArgs.title, Description: string(body), DueDate: fileArgs.due, Milestone: milestone,
				Labels: Labels(l.Plane, fileArgs.typ, l.Kind, l.Component)}
			r, err := File(t, fileArgs.project, in, p, ctx.DryRun)
			if err != nil {
				return err
			}
			r.Plane, r.Kind, r.Component, r.Derived = l.Plane, l.Kind, l.Component, l.Derived
			r.Milestone, r.MilestoneReason = milestone, reason
			return ctx.Emit(r, fmt.Sprintf("%s#%d %s: %s\nplane %s, kind %s, component %s, milestone %s",
				r.Project, r.IID, r.WebURL, verdict(r.Lint), r.Plane, orNone(r.Kind), orNone(r.Component), orNone(r.Milestone)))
		},
	})
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

func verdict(r LintResult) string {
	if r.Admitted {
		return "admitted"
	}
	return "needs-triage: " + strings.Join(r.Reasons, "; ")
}
