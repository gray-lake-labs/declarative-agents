---
id: srd013-program
title: "Script: issue (tpm issue ...)"
---

<!-- field: design_notes -->
## Design notes

- The group is named for its object. Every subcommand acts on an issue, and the group was named program before release 01.4, which said nothing about any of them.
- pop keeps its name. Pull, never push, is the process's rule, and popping an issue is its verb in the skills (implement), the vision, and the states (srd041); under the issue group the command reads as what it does.
- The old name breaks cleanly. Every caller is a skill embedded in the same tpm (srd037), so an alias would only keep stale skills running; instead tpm program runs nothing and names the fix.
- recur moves here from standup (srd010 R8.2, retired). It closes and reopens a recurring issue, wherever the issue lives and whoever runs it, which is the issue lifecycle, not a person's morning. The marker it writes into an issue body is tracker data, so its text does not change.

<!-- field: problem -->
## Problem

Every issue that enters a program has to carry the labels and sections the
other scripts read, and every merge has to tell its reviewer what to
look at. Done by hand, admission means something different in each
repository, and an issue missing its acceptance criteria reaches do-work
with nothing to build against.

The issue subcommands are the small set of issue operations every
repository shares: the labels admission reads, admission, filing, showing,
taking an issue into a worktree, and delivering it by a reviewed merge. Together with
the specify and implement skills (srd035, srd036) they are the inner
loop on GitLab.
Tracking Sprint Goals against evidence, verifying what was built, closing
drained tracking issues, and rendering the dashboard belong to track-goals (srd019);
recording a release plan belongs to plan.

<!-- field: goals -->
## Goals

- G1: Admission is a script, so needs-triage means the same thing in every repository.
- G2: Filing an issue and merging its branch each have one subcommand, used by every skill.
- G3: Every issue is taken the same way, claimed by the person who works it, on its own branch, in its own worktree.

<!-- req: R1 -->
## R1 - tpm issue lint

<!-- req: R1.1 -->
### R1.1

Checks one issue, or every issue changed since a date, against the label schema and policy.admission: exactly one plane, a component listed by a subsystem SRD on every issue except the sprint issue, once a subsystem SRD declares one (srd001 R21.2), exactly one kind on data issues, listed values only, Requirements and Acceptance Criteria sections where policy requires them, each criterion naming a test and an environment (design.md D8), every Cites line pinned (srd007 R5.3), and a management or control issue filed by someone policy allows.

<!-- req: R1.2 -->
### R1.2

Sets needs-triage on an issue that fails, with a comment listing each failure; clears needs-triage on an issue that passes. It changes no other label and never the assignee (srd041 R9.4).

<!-- req: R1.3 -->
### R1.3

Runs in CI on issue creation and edit where the tracker allows, and on a schedule otherwise.

<!-- req: R2 -->
## R2 - Common behavior

<!-- req: R2.1 -->
### R2.1

Subcommands are in the tpm group issue, honour --json and --dry-run, and never prompt (srd010 R1).

<!-- req: R2.2 -->
### R2.2

tpm program, the group's name before release 01.4, runs no subcommand: it exits non-zero naming tpm issue and tpm install, so a skill installed from an earlier tpm fails with its fix in the message.

<!-- req: R3 -->
## R3 - tpm issue file

<!-- req: R3.1 -->
### R3.1

Creates an issue in a given project with the given title, body, plane, kind, component, and milestone, unassigned, and runs lint on it before returning (R1). It never sets an assignee; a person who wants the issue pops it with implement (srd041 R9.1).

<!-- req: R3.2 -->
### R3.2

With a parent tracking issue given, appends the new issue to its Members section (srd001 R6); with a parent issue given, attaches the new issue as its sub-issue.

<!-- req: R3.3 -->
### R3.3

A data issue filed without a milestone and without a type gets the sprint: the active milestone of the project or its ancestor groups whose start date is on or before today and whose due date is on or after it. A milestone whose sprint has ended, its sprint-end issue in the project closed, no longer counts: the issue goes on the next milestone by start date, so a retrospective's actions land in the next sprint. With none, or more than one, it is filed without a milestone and the result says why; a milestone given explicitly always wins. tpm issue milestone puts an issue already filed in a milestone its project finds and lints it again. A kind given explicitly makes a data issue of that kind, whatever plane its paths alone would give.

<!-- req: R4 -->
## R4 - tpm issue show

<!-- req: R4.1 -->
### R4.1

Returns one issue with its labels, its state per srd041 R1–R32, its sub-issues and their states, its link lines (Upstream, Requires, Cites, Blocked-by, Blocks, Depends-on, Relates-to, Duplicates, Implements; srd007) with track-goals' explanation of each (srd019), its tracking issue if any, and its event comments, including its ledger events for a Sprint Goal.

<!-- req: R5 -->
## R5 - tpm issue merge

<!-- req: R5.1 -->
### R5.1

Merges a worktree branch into the default branch of the repository's main checkout with a merge commit whose message cites the issue and every issue named with --closes, pushes it, and closes each issue on that merge (srd042 R2.3).

<!-- req: R5.2 -->
### R5.2

Refuses when the branch has no commits ahead of the default branch, when the issue is closed, or when an issue named with --closes is malformed or closed.

<!-- req: R5.3 -->
### R5.3

For a branch that changes docs/, runs the corpus gate, the policy's audit_command, in the worktree first, and merges nothing when it fails (srd034 R2.2). Without --control the policy is read from the project's own control files; the built-in mage audit becomes tpm backlog check in a repository with no magefiles, which could not run it. The merge's output names the audit it ran, or says no docs/ change meant none.

<!-- req: R5.4 -->
### R5.4

With --issue-project, merges in the given project and closes the issue in the other by its full reference; the weekly status report merges in the reports repository and closes the status issue in the owner's repository (srd018 R6.1).

<!-- req: R5.5 -->
### R5.5

Refuses unless the issue carries an approval of the exact commit it would merge, from a person other than the issue's assignee, or, for an issue nobody took, other than the person merging (srd008 R1.4). An approval names the commit its reviewer read, so a commit made after it needs a new one.

<!-- req: R5.6 -->
### R5.6

tpm issue approve posts the approval as the person running it: a comment on the issue naming the tip of the branch at its origin. It refuses the issue's assignee, and posts nothing twice for the same commit.

<!-- req: R6 -->
## R6 - tpm issue labels

<!-- req: R6.1 -->
### R6.1

Creates or updates the program's group labels from control/labels.md in the program repository (config format labels), or from the built-in defaults when the file is absent: every plane, type, kind, and state value, with its color and description.

<!-- req: R6.2 -->
### R6.2

Is idempotent: a second run changes nothing. It never deletes or renames a label, so issues keep the labels they carry.

<!-- req: R6.3 -->
### R6.3

With --migrate and --control naming the system repository, also relabels issues for a value srd001 renamed: every issue, open or closed, labelled type::epic in the system repository and in every repository its subsystem SRDs declare gains type::tracking and loses type::epic (srd001 R6.6). It is opt-in, writes nothing with --dry-run, and is idempotent. It never deletes or renames the group label itself and never edits the program's control files; it reports when control/labels.md still lists epic, for a plane::control issue to change.

<!-- req: R7 -->
## R7 - tpm issue pop

<!-- req: R7.1 -->
### R7.1

Claims one issue for the caller: assigns the caller and no one else (srd041 R9.1). Refuses an issue assigned to someone else, a closed issue, and a plane::data issue that admission has not admitted (srd041 R2). With --issue-project, the issue is claimed there and the branch, which --branch may name, opens in --project: a weekly status issue's report lands in the reports repository (srd018). A plane::management or plane::control issue that admission has not admitted is popped, and pop reports the reasons it fails.

<!-- req: R7.2 -->
### R7.2

Creates the branch <iid>-<slug> from the default branch, the slug taken from the title, and adds a local git worktree for it beside the repository's checkout; reuses both when they already exist.

<!-- req: R7.3 -->
### R7.3

Returns the issue, its branch, and its worktree path, so the skill and the person work there and nowhere else.

<!-- req: R7.4 -->
### R7.4

Installs into the worktree the skills of every agent installed in the checkout (srd037), and reports which; a failed install is reported without undoing the pop, and --dry-run installs nothing.

<!-- req: R8 -->
## R8 - tpm issue recur

<!-- req: R8.1 -->
### R8.1

tpm issue recur closes a recurring issue with a run summary, appends the run to its Previous Runs section, updates its Requires version, and opens the next instance (srd001 R8.3). With --summary (or --summary-file), the run's result is the run summary it closes the issue with and the Previous Runs line in place of the outcome, with how the instance closed after it, so a run that merges no file still carries its result forward. An issue a merge already closed is not closed again; recur only appends the run and opens the next instance, and it opens none when the next instance already exists.

<!-- req: R8.2 -->
### R8.2

The marker recur writes into the next instance's body, naming the issue it recurs, keeps the text it had as tpm standup recur, so an instance opened before release 01.4 is still found and never opened twice.

<!-- field: non_goals -->
## Non-goals

- Splitting an issue into sub-issues and working them; that arrives with tracking issues and do-work in a later release.
- Choosing what goes in a release; that is plan.
- Tracking Sprint Goals, verifying merged work, closing tracking issues, or rendering the dashboard; that is track-goals (srd019).
- Offering or assigning work to a person.

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 -->
### AC1

lint sets needs-triage with a listed reason on each failing issue, including a criterion without test or environment and an unpinned Cites line, clears it on passing ones, changes nothing else, and runs on creation or on a schedule.

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
- [R1.2](#r12)
- [R1.3](#r13)

<!-- ac: AC2 -->
### AC2

Every issue subcommand honours --json and --dry-run and never prompts; tpm program runs nothing, exits non-zero, and names tpm issue and tpm install.

<!-- field: traces -->
#### Traces

- [R2.1](#r21)
- [R2.2](#r22)

<!-- ac: AC3 -->
### AC3

file creates an unassigned, linted issue (or a tracking-issue member, or a sub-issue of a parent issue) and never sets an assignee; show returns the issue with state, sub-issues, explained links, tracking issue, and events; a data issue filed without a milestone carries the one active sprint covering today, or none with the reason.

<!-- field: traces -->
#### Traces

- [R3.1](#r31)
- [R3.2](#r32)
- [R3.3](#r33)
- [R4.1](#r41)

<!-- ac: AC4 -->
### AC4

merge merges the branch with a merge commit citing the issue and named sub-issues, pushes it, and closes each on it, runs the corpus gate first for a docs/ change, and refuses an empty branch or a closed issue; with --issue-project it merges in one project and closes the issue in another.

<!-- field: traces -->
#### Traces

- [R5.1](#r51)
- [R5.2](#r52)
- [R5.4](#r54)

<!-- ac: AC5 -->
### AC5

A merge of a branch that changes docs/ runs mage audit first and merges nothing when it fails, whatever its issue's plane or kind.

<!-- field: traces -->
#### Traces

- [R5.3](#r53)

<!-- ac: AC6 -->
### AC6

labels creates every listed label once, from control/labels.md or the defaults; a second run changes nothing; no label is deleted or renamed; with --migrate, issues labelled type::epic in the system repository and its subsystems' repositories are relabelled type::tracking once, nothing is written on a dry run, and a control/labels.md that still lists epic is reported, not edited.

<!-- field: traces -->
#### Traces

- [R6.1](#r61)
- [R6.2](#r62)
- [R6.3](#r63)

<!-- ac: AC7 -->
### AC7

pop assigns only the caller, refuses an issue assigned to someone else, a closed issue, and a non-admitted data issue, creates or reuses the branch <iid>-<slug> and its worktree, and returns both, with the checkout's installed skills installed in the worktree.

<!-- field: traces -->
#### Traces

- [R7.1](#r71)
- [R7.2](#r72)
- [R7.3](#r73)
- [R7.4](#r74)

<!-- ac: AC8 -->
### AC8

merge refuses with no approval, with only the assignee's, and with an approval of an earlier commit, and merges with another person's approval of the tip; approve refuses the assignee and posts once per commit.

<!-- field: traces -->
#### Traces

- [R5.5](#r55)
- [R5.6](#r56)

<!-- ac: AC9 -->
### AC9

recur closes a recurring issue with its run logged and Requires updated and opens the next instance once, closes again nothing a merge closed, and finds an instance whose marker an earlier tpm standup recur wrote.

<!-- field: traces -->
#### Traces

- [R8.1](#r81)
- [R8.2](#r82)
