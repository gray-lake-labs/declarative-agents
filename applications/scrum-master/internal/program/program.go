// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
// Package program is the tpm issue group: the issue operations every
// repository shares, admission and filing (srd013). Its former name,
// program, runs nothing and names the group (srd013 R2.2).
package program

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/cli"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/tracker"
)

func init() {
	cli.Default.Retire("program", "issue")
	var labelArgs struct {
		group, control, ref string
		update, migrate     bool
	}
	cli.Default.Register(&cli.Command{
		Group: "issue", Name: "labels", Summary: "create the program's labels from control/labels.md or the defaults",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&labelArgs.group, "group", "", "program group path (required)")
			fs.StringVar(&labelArgs.control, "control", "", "program repository holding control/labels.md; empty uses the defaults")
			fs.StringVar(&labelArgs.ref, "ref", "main", "ref of control/labels.md")
			fs.BoolVar(&labelArgs.update, "update", false, "also rewrite the color and description of labels that differ")
			fs.BoolVar(&labelArgs.migrate, "migrate", false, "also relabel issues carrying a renamed type value (type::epic to type::tracking) in --control and its subsystems' repositories")
		},
		Run: func(ctx *cli.Context) error {
			if labelArgs.group == "" {
				return fmt.Errorf("%w: labels needs --group", cli.ErrUsage)
			}
			if labelArgs.migrate && labelArgs.control == "" {
				return fmt.Errorf("%w: labels --migrate needs --control, the system repository", cli.ErrUsage)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			want, err := LoadLabels(t, labelArgs.control, labelArgs.ref)
			if err != nil {
				return err
			}
			if labelArgs.control != "" {
				components, err := ComponentLabels(t, labelArgs.control, labelArgs.ref)
				if err != nil {
					return err
				}
				want = append(want, components...)
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
			if !labelArgs.migrate {
				return ctx.Emit(res, human)
			}
			m, err := migrate(t, labelArgs.control, labelArgs.ref, ctx.DryRun)
			if err != nil {
				return err
			}
			m.Labels = res
			human += fmt.Sprintf("\n%d issues relabelled", len(m.Relabelled))
			for _, r := range m.Relabelled {
				human += fmt.Sprintf("\n  %s#%d %s -> %s", r.Project, r.IID, r.From, r.To)
			}
			if len(m.StaleTypes) > 0 {
				human += fmt.Sprintf("\n%s %s still lists %s; change it with a plane::control issue", labelArgs.control, LabelsFile, strings.Join(m.StaleTypes, ", "))
			}
			return ctx.Emit(m, human)
		},
	})

	var listArgs struct{ project, dir, story, epic string }
	cli.Default.Register(&cli.Command{
		Group: "issue", Name: "list", Summary: "the open issues implementing a story, or every story under an epic, in the order implement works them",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&listArgs.project, "project", "", "project path (required)")
			fs.StringVar(&listArgs.story, "implements", "", "the story the issues implement, by id")
			fs.StringVar(&listArgs.epic, "epic", "", "every story under this epic, by id, read from --dir's hierarchy")
			fs.StringVar(&listArgs.dir, "dir", ".", "the checkout holding the stories, for --epic")
		},
		Run: func(ctx *cli.Context) error {
			if listArgs.project == "" || (listArgs.story == "") == (listArgs.epic == "") {
				return fmt.Errorf("%w: list needs --project and one of --implements or --epic", cli.ErrUsage)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			r, err := ListImplementing(t, listArgs.project, listArgs.dir, listArgs.story, listArgs.epic)
			if err != nil {
				return err
			}
			var b strings.Builder
			for _, i := range r.Issues {
				fmt.Fprintf(&b, "%s  %-12s %s  %s\n", i.Ref, i.Story, orNone(strings.Join(i.Assignees, ",")), i.Title)
			}
			for _, i := range r.Blocked {
				fmt.Fprintf(&b, "%s  %-12s blocked by %s  %s\n", i.Ref, i.Story, strings.Join(i.BlockedBy, ", "), i.Title)
			}
			if len(r.Issues)+len(r.Blocked) == 0 {
				fmt.Fprintf(&b, "no open issue implements %s\n", strings.Join(r.Stories, ", "))
			}
			return ctx.Emit(r, b.String())
		},
	})

	var popArgs struct {
		project, issueProject, branch, dir, control, ref, worktree string
		iid                                                        int
	}
	cli.Default.Register(&cli.Command{
		Group: "issue", Name: "pop", Summary: "claim an issue and open its own branch and worktree",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&popArgs.project, "project", "", "project path (required)")
			fs.IntVar(&popArgs.iid, "issue", 0, "issue IID (required)")
			fs.StringVar(&popArgs.issueProject, "issue-project", "", "project holding the issue, when not --project")
			fs.StringVar(&popArgs.branch, "branch", "", "the branch to open; empty names it from the issue")
			fs.StringVar(&popArgs.dir, "dir", ".", "a directory in the local checkout of the project")
			fs.StringVar(&popArgs.worktree, "worktree", "", "claim the issue into this worktree another pop opened, on its branch, so a story's issues share one branch (srd036)")
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
			issueProject := popArgs.issueProject
			if issueProject == "" {
				issueProject = popArgs.project
			}
			if popArgs.worktree != "" && popArgs.branch != "" {
				return fmt.Errorf("%w: --worktree takes the worktree's branch; give --branch or --worktree, not both", cli.ErrUsage)
			}
			var r PopResult
			if popArgs.worktree != "" {
				r, err = PopInto(t, ExecGit{}, popArgs.project, issueProject, popArgs.iid, popArgs.worktree, p, ctx.DryRun)
			} else {
				r, err = PopIn(t, ExecGit{}, popArgs.project, issueProject, popArgs.iid, popArgs.branch, popArgs.dir, p, ctx.DryRun)
			}
			if err != nil {
				return err
			}
			human := fmt.Sprintf("%s#%d on %s in %s", issueProject, r.IID, r.Branch, r.Worktree)
			if issueProject != r.Project {
				human = fmt.Sprintf("%s#%d on %s of %s in %s", issueProject, r.IID, r.Branch, r.Project, r.Worktree)
			}
			if len(r.NotAdmitted) > 0 {
				human += "\nnot admitted, which does not block a non-data issue: " + strings.Join(r.NotAdmitted, "; ")
			}
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

	var claimArgs struct {
		project, issueProject, control, ref string
		iid                                 int
	}
	cli.Default.Register(&cli.Command{
		Group: "issue", Name: "claim", Summary: "the claim word: assign the issue to the caller with pop's refusals, naming the branch to open (GH-204)",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&claimArgs.project, "project", "", "project path (required)")
			fs.IntVar(&claimArgs.iid, "issue", 0, "issue IID (required)")
			fs.StringVar(&claimArgs.issueProject, "issue-project", "", "project holding the issue, when not --project")
			fs.StringVar(&claimArgs.control, "control", "", "program repository holding control/ files; empty uses the defaults")
			fs.StringVar(&claimArgs.ref, "ref", "main", "ref of the control files")
		},
		Run: func(ctx *cli.Context) error {
			if claimArgs.project == "" || claimArgs.iid == 0 {
				return fmt.Errorf("%w: claim needs --project and --issue", cli.ErrUsage)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			p, err := LoadPolicy(t, claimArgs.control, claimArgs.ref)
			if err != nil {
				return err
			}
			issueProject := claimArgs.issueProject
			if issueProject == "" {
				issueProject = claimArgs.project
			}
			r, _, err := Claim(t, claimArgs.project, issueProject, claimArgs.iid, p, ctx.DryRun)
			if err != nil {
				return err
			}
			return ctx.Emit(r, fmt.Sprintf("%s#%d claimed; open its branch %s with tpm issue open-branch", issueProject, r.IID, r.Branch))
		},
	})

	var obArgs struct {
		project, issueProject, branch string
		iid                           int
	}
	cli.Default.Register(&cli.Command{
		Group: "issue", Name: "open-branch", Summary: "the branch word: create or reuse the issue's branch and, on a hosted tracker, its review (GH-204)",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&obArgs.project, "project", "", "project path (required)")
			fs.IntVar(&obArgs.iid, "issue", 0, "issue IID (required)")
			fs.StringVar(&obArgs.issueProject, "issue-project", "", "project holding the issue, when not --project")
			fs.StringVar(&obArgs.branch, "branch", "", "the branch to open (required; tpm issue claim names it)")
		},
		Run: func(ctx *cli.Context) error {
			if obArgs.project == "" || obArgs.iid == 0 || obArgs.branch == "" {
				return fmt.Errorf("%w: open-branch needs --project, --issue, and --branch", cli.ErrUsage)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			issueProject := obArgs.issueProject
			if issueProject == "" {
				issueProject = obArgs.project
			}
			i, err := t.GetIssue(issueProject, obArgs.iid)
			if err != nil {
				return err
			}
			created, review, err := OpenIssueBranch(t, obArgs.project, issueProject, obArgs.iid, obArgs.branch, i.Title, ctx.DryRun)
			if err != nil {
				return err
			}
			r := struct {
				Project string `json:"project"`
				IID     int    `json:"iid"`
				Branch  string `json:"branch"`
				Created bool   `json:"branch_created"`
				Review  string `json:"review,omitempty"`
				DryRun  bool   `json:"dry_run"`
			}{obArgs.project, obArgs.iid, obArgs.branch, created, review, ctx.DryRun}
			human := fmt.Sprintf("branch %s opened on %s", r.Branch, r.Project)
			if !created {
				human = fmt.Sprintf("branch %s already on %s", r.Branch, r.Project)
			}
			return ctx.Emit(r, human)
		},
	})

	var awArgs struct {
		branch, dir string
	}
	cli.Default.Register(&cli.Command{
		Group: "issue", Name: "add-worktree", Summary: "the worktree word: the sibling worktree for the issue's branch, created or reused (GH-204)",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&awArgs.branch, "branch", "", "the branch the worktree checks out (required)")
			fs.StringVar(&awArgs.dir, "dir", ".", "a directory in the local checkout of the project")
		},
		Run: func(ctx *cli.Context) error {
			if awArgs.branch == "" {
				return fmt.Errorf("%w: add-worktree needs --branch", cli.ErrUsage)
			}
			checkout, worktree, added, err := AddIssueWorktree(ExecGit{}, awArgs.dir, awArgs.branch, ctx.DryRun)
			if err != nil {
				return err
			}
			r := struct {
				Branch   string `json:"branch"`
				Checkout string `json:"checkout"`
				Worktree string `json:"worktree"`
				Added    bool   `json:"worktree_added"`
				DryRun   bool   `json:"dry_run"`
			}{awArgs.branch, checkout, worktree, added, ctx.DryRun}
			human := fmt.Sprintf("worktree %s on %s", worktree, awArgs.branch)
			if !added {
				human = fmt.Sprintf("worktree %s already there", worktree)
			}
			return ctx.Emit(r, human)
		},
	})

	var isArgs struct {
		checkout, worktree string
	}
	cli.Default.Register(&cli.Command{
		Group: "issue", Name: "install-skills", Summary: "the install word: the checkout's agents' skills into the worktree (GH-204, srd036 R2.3)",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&isArgs.checkout, "checkout", "", "the checkout whose installed agents decide what installs (required)")
			fs.StringVar(&isArgs.worktree, "worktree", "", "the worktree the skills install into (required)")
		},
		Run: func(ctx *cli.Context) error {
			if isArgs.checkout == "" || isArgs.worktree == "" {
				return fmt.Errorf("%w: install-skills needs --checkout and --worktree", cli.ErrUsage)
			}
			if ctx.DryRun {
				return fmt.Errorf("install-skills has no dry run; tpm issue pop --dry-run shows the whole pop")
			}
			installed, err := InstallSkills(isArgs.checkout, isArgs.worktree)
			if err != nil {
				return err
			}
			r := struct {
				Worktree  string   `json:"worktree"`
				Installed []string `json:"installed"`
			}{isArgs.worktree, installed}
			human := "skills installed for " + strings.Join(installed, ", ")
			if len(installed) == 0 {
				human = "nothing to install; the checkout has no agents or the worktree already has their skills"
			}
			return ctx.Emit(r, human)
		},
	})

	var closeArgs struct {
		project, dir string
		iid          int
	}
	cli.Default.Register(&cli.Command{
		Group: "issue", Name: "close", Summary: "close an issue on the local merge that delivered it (beads tracker)",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&closeArgs.project, "project", "", "project path (required)")
			fs.IntVar(&closeArgs.iid, "issue", 0, "issue IID (required)")
			fs.StringVar(&closeArgs.dir, "dir", ".", "a directory in the local checkout holding the merge")
		},
		Run: func(ctx *cli.Context) error {
			if closeArgs.project == "" || closeArgs.iid == 0 {
				return fmt.Errorf("%w: close needs --project and --issue", cli.ErrUsage)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			r, err := Close(t, ExecGit{}, closeArgs.project, closeArgs.iid, closeArgs.dir, "", ctx.DryRun)
			if err != nil {
				return err
			}
			human := fmt.Sprintf("%s#%d closed on %s", r.Project, r.IID, r.Commit)
			if !r.Closed && !r.DryRun {
				human = fmt.Sprintf("%s#%d was already closed on %s", r.Project, r.IID, r.Commit)
			}
			return ctx.Emit(r, human)
		},
	})

	var commentArgs struct {
		project, body, bodyFile string
		iid                     int
	}
	cli.Default.Register(&cli.Command{
		Group: "issue", Name: "comment", Summary: "post a comment on an issue, such as a completed issue's Actual LOC line",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&commentArgs.project, "project", "", "project holding the issue (required)")
			fs.IntVar(&commentArgs.iid, "issue", 0, "issue IID (required)")
			fs.StringVar(&commentArgs.body, "body", "", "the comment's text")
			fs.StringVar(&commentArgs.bodyFile, "body-file", "", "a file holding the comment's text, in place of --body")
		},
		Run: func(ctx *cli.Context) error {
			if commentArgs.project == "" || commentArgs.iid == 0 {
				return fmt.Errorf("%w: comment needs --project and --issue", cli.ErrUsage)
			}
			body := commentArgs.body
			if commentArgs.bodyFile != "" {
				if body != "" {
					return fmt.Errorf("%w: give --body or --body-file, not both", cli.ErrUsage)
				}
				b, err := os.ReadFile(commentArgs.bodyFile)
				if err != nil {
					return err
				}
				body = string(b)
			}
			if strings.TrimSpace(body) == "" {
				return fmt.Errorf("%w: comment needs a non-empty --body or --body-file", cli.ErrUsage)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			r, err := Comment(t, commentArgs.project, commentArgs.iid, body, ctx.DryRun)
			if err != nil {
				return err
			}
			human := fmt.Sprintf("commented on %s#%d", r.Project, r.IID)
			if r.DryRun {
				human = fmt.Sprintf("dry run: would comment on %s#%d", r.Project, r.IID)
			}
			return ctx.Emit(r, human)
		},
	})

	var approveArgs struct {
		project, issueProject, branch, dir string
		iid                                int
	}
	cli.Default.Register(&cli.Command{
		Group: "issue", Name: "approve", Summary: "approve an issue's branch for merge, as a reviewer other than its assignee",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&approveArgs.project, "project", "", "project holding the branch (required)")
			fs.IntVar(&approveArgs.iid, "issue", 0, "issue IID (required)")
			fs.StringVar(&approveArgs.issueProject, "issue-project", "", "project holding the issue, when not --project")
			fs.StringVar(&approveArgs.branch, "branch", "", "branch to approve; empty uses the issue's pop branch")
			fs.StringVar(&approveArgs.dir, "dir", ".", "a checkout of the project")
		},
		Run: func(ctx *cli.Context) error {
			if approveArgs.project == "" || approveArgs.iid == 0 {
				return fmt.Errorf("%w: approve needs --project and --issue", cli.ErrUsage)
			}
			if ctx.DryRun {
				return fmt.Errorf("%w: approve is one comment and has no dry run", cli.ErrUsage)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			issueProject := approveArgs.issueProject
			if issueProject == "" {
				issueProject = approveArgs.project
			}
			r, err := Approve(t, ExecGit{}, approveArgs.project, issueProject, approveArgs.iid, approveArgs.branch, approveArgs.dir)
			if err != nil {
				return err
			}
			human := fmt.Sprintf("%s approved at %s by %s", r.Branch, r.Commit, r.By)
			if !r.Posted {
				human = fmt.Sprintf("%s was already approved at %s by %s", r.Branch, r.Commit, r.By)
			}
			return ctx.Emit(r, human)
		},
	})

	var mergeArgs struct {
		project, issueProject, branch, audit, dir, control, ref string
		iid                                                     int
		closes                                                  cli.Strings
	}
	cli.Default.Register(&cli.Command{
		Group: "issue", Name: "merge", Summary: "deliver an issue by a local merge that closes it (beads tracker)",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&mergeArgs.project, "project", "", "project path (required)")
			fs.IntVar(&mergeArgs.iid, "issue", 0, "issue IID (required)")
			fs.StringVar(&mergeArgs.issueProject, "issue-project", "", "project holding the issue, when not --project")
			fs.StringVar(&mergeArgs.branch, "branch", "", "branch to merge; empty uses the current branch of the working directory")
			fs.StringVar(&mergeArgs.audit, "audit", "", "corpus gate command for a docs/ change; empty uses the policy's audit_command, tpm backlog check in a repository without magefiles")
			fs.StringVar(&mergeArgs.dir, "dir", ".", "the worktree holding the branch, where the audit runs")
			fs.StringVar(&mergeArgs.control, "control", "", "program repository holding control/ files; empty reads them from --project, else the defaults")
			fs.StringVar(&mergeArgs.ref, "ref", "main", "ref of the control files")
			fs.Var(&mergeArgs.closes, "closes", "another issue the merge closes, project#iid; repeat for each (srd038 R2.1)")
		},
		Run: func(ctx *cli.Context) error {
			if mergeArgs.project == "" || mergeArgs.iid == 0 {
				return fmt.Errorf("%w: merge needs --project and --issue", cli.ErrUsage)
			}
			branch := mergeArgs.branch
			if branch == "" {
				out, err := ExecGit{}.Run(mergeArgs.dir, "rev-parse", "--abbrev-ref", "HEAD")
				if err != nil {
					return err
				}
				branch = strings.TrimSpace(out)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			control := mergeArgs.control
			if control == "" {
				control = mergeArgs.project
			}
			p, err := LoadPolicy(t, control, mergeArgs.ref)
			if err != nil {
				return err
			}
			gate := MergeGate{Command: AuditFor(p, mergeArgs.dir), Dir: mergeArgs.dir, RequireApproval: p.RequireApproval}
			if mergeArgs.audit != "" {
				gate.Command = mergeArgs.audit
			}
			issueProject := mergeArgs.issueProject
			if issueProject == "" {
				issueProject = mergeArgs.project
			}
			r, err := Merge(t, ExecGit{}, mergeArgs.project, issueProject, mergeArgs.iid, branch, mergeArgs.dir, gate, mergeArgs.closes, ctx.DryRun)
			if err != nil {
				return err
			}
			audit := "no docs/ change, so no audit ran"
			if r.Audit {
				audit = "audit " + r.AuditCommand + " passed"
			}
			human := fmt.Sprintf("%s merged into %s on %s; %s; closed %s", r.Branch, r.Project, r.Commit, audit, strings.Join(r.Closed, ", "))
			if len(r.Reported) > 0 {
				human += "; reported complete " + strings.Join(r.Reported, ", ")
			}
			if r.DryRun {
				approved := "with no approval"
				if r.ApprovedBy != "" {
					approved = "approved by " + r.ApprovedBy
				}
				human = fmt.Sprintf("dry run: %s would merge into %s, %s, closing %s#%d", r.Branch, r.Project, approved, issueProject, r.IID)
				if r.Commit != "" && !r.Merged {
					// Already merged: nothing merges, approval is moot, and
					// the run would only close what is open (GH-20).
					human = fmt.Sprintf("dry run: %s is already merged on %s; a run would close %s#%d", r.Branch, short12(r.Commit), issueProject, r.IID)
				}
				if len(mergeArgs.closes) > 0 {
					human += " and " + strings.Join(mergeArgs.closes, ", ")
				}
			} else if !r.Merged {
				human = fmt.Sprintf("%s was already merged on %s; closed %s", r.Branch, r.Commit, strings.Join(r.Closed, ", "))
			}
			return ctx.Emit(r, human)
		},
	})

	var mbArgs struct {
		project, issueProject, branch, audit, dir, control, ref string
		iid                                                     int
		closes                                                  cli.Strings
	}
	cli.Default.Register(&cli.Command{
		Group: "issue", Name: "merge-branch", Summary: "the merge word: gates, the --no-ff merge, and the push, closing nothing (GH-201)",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&mbArgs.project, "project", "", "project path (required)")
			fs.IntVar(&mbArgs.iid, "issue", 0, "issue IID (required)")
			fs.StringVar(&mbArgs.issueProject, "issue-project", "", "project holding the issue, when not --project")
			fs.StringVar(&mbArgs.branch, "branch", "", "branch to merge; empty uses the current branch of the working directory")
			fs.StringVar(&mbArgs.audit, "audit", "", "corpus gate command for a docs/ change; empty uses the policy's audit_command")
			fs.StringVar(&mbArgs.dir, "dir", ".", "the worktree holding the branch, where the audit runs")
			fs.StringVar(&mbArgs.control, "control", "", "program repository holding control/ files; empty reads them from --project, else the defaults")
			fs.StringVar(&mbArgs.ref, "ref", "main", "ref of the control files")
			fs.Var(&mbArgs.closes, "closes", "another issue the merge cites and close-delivered closes, project#iid; repeat for each")
		},
		Run: func(ctx *cli.Context) error {
			if mbArgs.project == "" || mbArgs.iid == 0 {
				return fmt.Errorf("%w: merge-branch needs --project and --issue", cli.ErrUsage)
			}
			branch := mbArgs.branch
			if branch == "" {
				out, err := ExecGit{}.Run(mbArgs.dir, "rev-parse", "--abbrev-ref", "HEAD")
				if err != nil {
					return err
				}
				branch = strings.TrimSpace(out)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			control := mbArgs.control
			if control == "" {
				control = mbArgs.project
			}
			p, err := LoadPolicy(t, control, mbArgs.ref)
			if err != nil {
				return err
			}
			gate := MergeGate{Command: AuditFor(p, mbArgs.dir), Dir: mbArgs.dir, RequireApproval: p.RequireApproval}
			if mbArgs.audit != "" {
				gate.Command = mbArgs.audit
			}
			issueProject := mbArgs.issueProject
			if issueProject == "" {
				issueProject = mbArgs.project
			}
			r, err := MergeBranch(t, ExecGit{}, mbArgs.project, issueProject, mbArgs.iid, branch, mbArgs.dir, gate, mbArgs.closes, ctx.DryRun)
			if err != nil {
				return err
			}
			human := fmt.Sprintf("%s merged into %s on %s; nothing closed: run tpm issue close-delivered --commit %s", r.Branch, r.Project, r.Commit, r.Commit)
			if r.DryRun && r.Commit != "" && !r.Merged {
				human = fmt.Sprintf("dry run: %s is already merged on %s; nothing would merge", r.Branch, r.Commit)
			} else if r.DryRun {
				human = fmt.Sprintf("dry run: %s would merge into %s", r.Branch, r.Project)
			} else if !r.Merged {
				human = fmt.Sprintf("%s was already merged on %s; nothing closed", r.Branch, r.Commit)
			}
			// The merge word's output always carries closes, what
			// close-delivered takes next (GH-201, GH-20).
			out := struct {
				MergeResult
				Closes string `json:"closes"`
			}{r, r.ToClose}
			return ctx.Emit(out, human)
		},
	})

	var cdArgs struct {
		project, issueProject, commit, closes, dir string
		iid                                        int
	}
	cli.Default.Register(&cli.Command{
		Group: "issue", Name: "close-delivered", Summary: "the close word: close the issue and each --closes on the merge commit, and report complete each closed Sprint Goal (GH-201)",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&cdArgs.project, "project", "", "project path (required)")
			fs.IntVar(&cdArgs.iid, "issue", 0, "issue IID (required)")
			fs.StringVar(&cdArgs.issueProject, "issue-project", "", "project holding the issue, when not --project")
			fs.StringVar(&cdArgs.commit, "commit", "", "the merge commit merge-branch reported (required)")
			fs.StringVar(&cdArgs.closes, "closes", "", "the other issues the merge closes, project#iid comma-separated")
			fs.StringVar(&cdArgs.dir, "dir", ".", "a directory in the local checkout")
		},
		Run: func(ctx *cli.Context) error {
			if cdArgs.project == "" || cdArgs.iid == 0 || cdArgs.commit == "" {
				return fmt.Errorf("%w: close-delivered needs --project, --issue, and --commit", cli.ErrUsage)
			}
			if ctx.DryRun {
				return fmt.Errorf("close-delivered has no dry run; tpm issue merge --dry-run shows the whole delivery")
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			issueProject := cdArgs.issueProject
			if issueProject == "" {
				issueProject = cdArgs.project
			}
			var closes []string
			for _, ref := range strings.Split(cdArgs.closes, ",") {
				if ref = strings.TrimSpace(ref); ref != "" {
					closes = append(closes, ref)
				}
			}
			closed, reported, already, err := CloseDelivered(t, ExecGit{}, cdArgs.dir, cdArgs.commit, issueProject, cdArgs.iid, closes)
			r := MergeResult{Project: cdArgs.project, IID: cdArgs.iid, Commit: cdArgs.commit, Closed: closed, Reported: reported, AlreadyClosed: already}
			if issueProject != cdArgs.project {
				r.IssueProject = issueProject
			}
			if err != nil {
				return err
			}
			human := fmt.Sprintf("closed %s on %s", strings.Join(closed, ", "), cdArgs.commit)
			if len(reported) > 0 {
				human += "; reported complete " + strings.Join(reported, ", ")
			}
			return ctx.Emit(r, human)
		},
	})

	var lintArgs struct {
		project, control, ref, since string
		iid                          int
	}
	cli.Default.Register(&cli.Command{
		Group: "issue", Name: "lint", Summary: "check issues against the label schema and admission policy",
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
			var issues []tracker.Issue
			if lintArgs.iid != 0 {
				i, err := t.GetIssue(lintArgs.project, lintArgs.iid)
				if err != nil {
					return err
				}
				issues = []tracker.Issue{i}
			} else if issues, err = t.ListIssues(lintArgs.project, tracker.IssueQuery{State: "opened", UpdatedAfter: lintArgs.since}); err != nil {
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
		Group: "issue", Name: "file", Summary: "file an unassigned issue and lint it",
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
			fs.StringVar(&fileArgs.typ, "type", "", "type: tracking, sprint, or recurring")
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
				if milestone, reason, err = SprintMilestone(t, fileArgs.project, time.Now(), l.Kind == "release"); err != nil {
					return err
				}
			}
			in := tracker.NewIssue{Title: fileArgs.title, Description: string(body), DueDate: fileArgs.due, Milestone: milestone,
				Labels: append(Labels(l.Plane, fileArgs.typ, l.Kind, l.Component), Op(OpFile))}
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
