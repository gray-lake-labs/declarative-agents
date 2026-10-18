---
id: srd013-program
title: "Script: program (tpm program ...)"
---

<!-- field: problem -->
## Problem

Every issue that enters a program has to carry the labels and sections the
other scripts read, and every merge request has to tell its reviewer what to
look at. Done by hand, admission means something different in each
repository, and an issue missing its acceptance criteria reaches do-work
with nothing to build against.

The program subcommands are the small set of issue operations every
repository shares: the labels admission reads, admission, filing, showing,
taking an issue into a worktree, and opening a merge request. Together with
the issue-push and issue-pop skills (srd035, srd036) they are the inner
loop on GitLab.
Tracking objectives against evidence, verifying what was built, closing
drained epics, and rendering the dashboard belong to track-goals (srd019);
recording a release plan belongs to plan-release.

<!-- field: goals -->
## Goals

- G1: Admission is a script, so needs-triage means the same thing in every repository.
- G2: Filing an issue and opening its merge request each have one subcommand, used by every skill.
- G3: Every issue is taken the same way, claimed by the person who works it, on its own branch, in its own worktree.

<!-- req: R1 -->
## R1 - tpm program lint

<!-- req: R1.1 -->
### R1.1

Checks one issue, or every issue changed since a date, against the label schema and policy.admission: exactly one plane, a component listed by a subsystem SRD on every issue except the sprint issue, exactly one kind on data issues, listed values only, Requirements and Acceptance Criteria sections where policy requires them, each criterion naming a test and an environment (design.md D8), every Cites line pinned (srd007 R5.3), and a management or control issue filed by someone policy allows.

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

Subcommands are in the tpm group program, honour --json and --dry-run, and never prompt (srd010 R1).

<!-- req: R3 -->
## R3 - tpm program file

<!-- req: R3.1 -->
### R3.1

Creates an issue in a given project with the given title, body, plane, kind, component, and milestone, unassigned, and runs lint on it before returning (R1). It never sets an assignee; a person who wants the issue pops it with issue-pop (srd041 R9.1).

<!-- req: R3.2 -->
### R3.2

With a parent epic given, appends the new issue to the epic's Members section; with a parent issue given, attaches the new issue as its sub-issue.

<!-- req: R3.3 -->
### R3.3

A data issue filed without a milestone and without a type gets the sprint: the active milestone of the project or its ancestor groups whose start date is on or before today and whose due date is on or after it. With none, or more than one, it is filed without a milestone and the result says why; a milestone given explicitly always wins.

<!-- req: R4 -->
## R4 - tpm program show

<!-- req: R4.1 -->
### R4.1

Returns one issue with its labels, its state per srd041 R1–R32, its sub-issues and their states, its Upstream, Requires, and Cites lines with track-goals' explanation of each (srd019), its epic if any, and its event comments, including its ledger events for an objective.

<!-- req: R5 -->
## R5 - tpm program merge

<!-- req: R5.1 -->
### R5.1

Merges a worktree branch into the default branch of the repository's main checkout with a merge commit whose message cites the issue and every issue named with --closes, pushes it, and closes each issue on that merge (srd042 R2.3).

<!-- req: R5.2 -->
### R5.2

Refuses when the branch has no commits ahead of the default branch, when the issue is closed, or when an issue named with --closes is malformed or closed.

<!-- req: R5.3 -->
### R5.3

For a branch that changes docs/, runs the corpus gate, mage audit or the policy's audit_command, in the worktree first, and merges nothing when it fails (srd034 R2.2).

<!-- req: R5.4 -->
### R5.4

With --issue-project, merges in the given project and closes the issue in the other by its full reference; the weekly status report merges in the reports repository and closes the status issue in the owner's repository (srd018 R6.1).

<!-- req: R6 -->
## R6 - tpm program labels

<!-- req: R6.1 -->
### R6.1

Creates or updates the program's group labels from control/labels.md in the program repository (config format labels), or from the built-in defaults when the file is absent: every plane, type, kind, and state value, with its color and description.

<!-- req: R6.2 -->
### R6.2

Is idempotent: a second run changes nothing. It never deletes or renames a label, so issues keep the labels they carry.

<!-- req: R7 -->
## R7 - tpm program pop

<!-- req: R7.1 -->
### R7.1

Claims one issue for the caller: assigns the caller and no one else (srd041 R9.1). Refuses an issue assigned to someone else, a closed issue, and a plane::data issue that admission has not admitted (srd041 R2).

<!-- req: R7.2 -->
### R7.2

Creates the branch <iid>-<slug> from the default branch, the slug taken from the title, and adds a local git worktree for it beside the repository's checkout; reuses both when they already exist.

<!-- req: R7.3 -->
### R7.3

Returns the issue, its branch, and its worktree path, so the skill and the person work there and nowhere else.

<!-- req: R7.4 -->
### R7.4

Installs into the worktree the skills of every agent installed in the checkout (srd037), and reports which; a failed install is reported without undoing the pop, and --dry-run installs nothing.

<!-- field: non_goals -->
## Non-goals

- Splitting an issue into sub-issues and working them; that arrives with epics and do-work in a later release.
- Choosing what goes in a release; that is plan-release.
- Tracking objectives, verifying merged work, closing epics, or rendering the dashboard; that is track-goals (srd019).
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

Every program subcommand honours --json and --dry-run and never prompts.

<!-- field: traces -->
#### Traces

- [R2.1](#r21)

<!-- ac: AC3 -->
### AC3

file creates an unassigned, linted issue (or an epic member, or a sub-issue of a parent issue) and never sets an assignee; show returns the issue with state, sub-issues, explained links, epic, and events; a data issue filed without a milestone carries the one active sprint covering today, or none with the reason.

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

A merge request that changes docs/ names mage audit in its reviewer focus, whatever its issue's plane or kind.

<!-- field: traces -->
#### Traces

- [R5.3](#r53)

<!-- ac: AC6 -->
### AC6

labels creates every listed label once, from control/labels.md or the defaults; a second run changes nothing; no label is deleted or renamed.

<!-- field: traces -->
#### Traces

- [R6.1](#r61)
- [R6.2](#r62)

<!-- ac: AC7 -->
### AC7

pop assigns only the caller, refuses an issue assigned to someone else, a closed issue, and a non-admitted data issue, creates or reuses the branch <iid>-<slug> and its worktree, and returns both, with the checkout's installed skills installed in the worktree.

<!-- field: traces -->
#### Traces

- [R7.1](#r71)
- [R7.2](#r72)
- [R7.3](#r73)
- [R7.4](#r74)
