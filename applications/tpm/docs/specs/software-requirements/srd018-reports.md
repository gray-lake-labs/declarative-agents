---
id: srd018-reports
title: "Script: reports (tpm reports ...)"
---
<!-- Copyright (c) 2026 Gray Lake Labs -->
<!-- SPDX-License-Identifier: MIT-0 -->

<!-- field: problem -->
## Problem

Weekly status is the one place people say what the tracker cannot know:
why work mattered, what blocks them, what comes next. Written from memory
on a Friday, reports contradict the tracker and leave out work that
shipped. Read one at a time, they hide what cuts across them: the
component everyone waits on, the commitment that slipped three weeks
running.

The reports subcommands keep each report in a fixed, checkable shape and
aggregate a week or a sprint of them against the tracker. The weekly
sprint update and the stakeholder update are recurring audit issues worked
by the inner loop (implement and the reviewed merge); these
subcommands are what those issues read and what their acceptance criteria
check, so a roll-up starts from computed candidates instead of a pile of
prose.

<!-- field: goals -->
## Goals

- G1: Every report has the same machine-checkable shape, and its index cannot drift from its body.
- G2: Aggregation finds what cuts across reports and where reports contradict the tracker.
- G3: Nothing aggregated ranks or compares people.

<!-- req: R1 -->
## R1 - tpm reports init

<!-- req: R1.1 -->
### R1.1

Scaffolds a reports repository: a reports directory per quarter, the report schema, a summaries directory, the configuration reports.md (config format reports-config), and a deck template, and installs the skills for the agents --agent names, as tpm init does. The team runs tpm reports status-issues each week on a schedule of its own, and tpm reports validate before each report merges.

<!-- req: R1.2 -->
### R1.2

Is idempotent and never overwrites an existing report or summary.

<!-- req: R1.3 -->
### R1.3

Writes into a local checkout of the reports repository (--dir), whose owner commits the scaffold through the inner loop.

<!-- req: R2 -->
## R2 - tpm reports validate

<!-- req: R2.1 -->
### R2.1

Checks that each report's body has a Work section and a Blockers section, one entry per item, each with its key-value fields, and that its front matter (author, sprint, components, blocked-on) is derived from the body (config format report).

<!-- req: R2.2 -->
### R2.2

Exits non-zero for any report whose front matter disagrees with its body.

<!-- req: R3 -->
## R3 - tpm reports aggregate

<!-- req: R3.1 -->
### R3.1

For a week or a sprint, reads every report in it and groups items by component.

<!-- req: R3.2 -->
### R3.2

Computes signals: blockers shared by two or more reports, blockers older than policy.escalation.blocker_age_days, commitments that slipped in policy.escalation.slip_releases or more consecutive reports, components with an expected report missing, or whose reports name no Sprint Goal of theirs (the system repository, which holds none, aside), items that contradict the ledger (a report calling a Sprint Goal done that has no met event, srd006 R10, unless a sprint-end merge closed it as reported complete before track-goals runs, srd038 R2), and merged work in the tracker that no report mentions. Merged work counts only deliveries, merges that close an issue they cite other than a recurring one (a sprint-end or status issue), landed no earlier than the creation of the first Sprint Goal the range's reports name: a merge from before the period began, such as an earlier run's on the same days, is not the period's work. Merge times compare as instants in UTC. A Sprint Goal a sprint-end merge closed as reported complete reads so in progress whatever its last report said, and a blocker a component reports on its own work is named as such, not as the component blocking itself. A cost overrun has no signal of its own: a report states it as a blocker naming the spend and the limit it passed.

<!-- req: R3.3 -->
### R3.3

For a sprint, adds the dashboard signals of tpm goals status (srd019) as further candidates.

<!-- req: R3.4 -->
### R3.4

Expresses every signal by component or by blocker, never as a comparison or ranking of people.

<!-- req: R3.5 -->
### R3.5

States progress as the ledger state of each Sprint Goal (srd006 R11) and its latest evidence, never as counts of closed issues or tasks.

<!-- req: R3.6 -->
### R3.6

With --push, files each signal that needs action as an issue in the issue body's work format in the system repository the configuration names (R6.3), unassigned, its body mentioning the person who pushed it (srd041 R9.2), and reports each issue it files with its admission, as tpm issue file does.

<!-- req: R4 -->
## R4 - tpm reports deck

<!-- req: R4.1 -->
### R4.1

Renders an approved summary into a slide deck from the template, one slide per outline section, up to the slide limit set in the reports repository's configuration.

<!-- req: R5 -->
## R5 - Common behavior

<!-- req: R5.1 -->
### R5.1

Subcommands are in the tpm group reports, honour --json and --dry-run, and never prompt (srd010 R1).

<!-- req: R6 -->
## R6 - Report audits

<!-- req: R6.1 -->
### R6.1

The sprint update is a recurring issue (srd001 R8) per member per week. Its Requirements name tpm standup activity, fetch, and inbox (srd010) as the facts to draft from; its Acceptance Criteria are a report in the fixed shape that passes tpm reports validate, written by the member about their own work only. It closes by its merge, and implement re-creates it for the next week.

<!-- req: R6.2 -->
### R6.2

The stakeholder update is a recurring issue per lead per period. Its Requirements name tpm reports aggregate with the tracker check and the previous roll-up's open risks; its Acceptance Criteria are a summary, an updated risk list, and a deck rendered by tpm reports deck, with no comparison or ranking of people. The lead approves the outline on the summary's issue before the summary is written.

<!-- req: R6.3 -->
### R6.3

A signal from aggregate that needs action is pushed as an issue in the issue body's work format, not carried in the prose of a report.

<!-- req: R7 -->
## R7 - tpm reports status-issues

<!-- req: R7.1 -->
### R7.1

For each person the configuration lists, ensures one status issue (the sprint update of R6.1) exists in their project for the coming ISO week. When the person's latest status issue is closed, it recurs that issue (srd010 R8.2); when there is none, it files a fresh one through tpm issue file.

<!-- req: R7.2 -->
### R7.2

Is idempotent: a second run in the same week files nothing. The scheduled CI job runs it each week, so the next instance follows the merge that closed the last one.

<!-- req: R8 -->
## R8 - tpm reports week

<!-- req: R8.1 -->
### R8.1

For a project and a person, returns the project's open Sprint Goal issues (Sprint Goals included) with their due dates and the person's activity since the ISO week began, or since the start of the week --date names, the week taken in UTC: branches merged and issues closed. A merge counts from the later of the week's start and the creation of the project's earliest open or recently closed Sprint Goal, so an earlier run's merges in the same week do not; a project holding no Sprint Goal of its own, the system repository, starts at its own earliest open or recently closed issue, so its planning and registration merges count, and only with no issue of its own takes the teams' earliest instead, the teams being those --control's subsystem SRDs name. The issues closed are those assigned to the person and those a counted merge closes. A merge is the person's when its Merged-by trailer, which tpm issue merge writes, names them, whatever git identity made the commit. These are the facts the status issue's guided script drafts from.

<!-- req: R9 -->
## R9 - Weekly status issue

<!-- req: R9.1 -->
### R9.1

The weekly status issue is the sprint update of R6.1, filed in each owner's repository for the people the reports repository's configuration lists as its team. A program that lists only sub-project leads receives one status issue per lead per week.

<!-- req: R9.2 -->
### R9.2

Its due date is the day after the reminder weekday in the reports repository's configuration, Wednesday by default, because the tracker sends its due-date notification the day before an issue is due. The body mentions the person it is for, which makes that person a participant who receives the notification (srd006 R7.1) without assigning them (srd041 R9.1).

<!-- req: R9.3 -->
### R9.3

Its Requirements section is the guided script the owner works with implement: for each open Sprint Goal issue in the repository, record its status, its problems, and complete or not complete. The output is a report in the srd018 shape that passes tpm reports validate.

<!-- req: R9.4 -->
### R9.4

It closes by its merge; the reports repository's scheduled tpm reports status-issues run recurs it with the next week's due date (R7, srd001 R8.3).

<!-- req: R10 -->
## R10 - Summary on a system release

<!-- req: R10.1 -->
### R10.1

A release tag in the system repository triggers a CI pipeline that runs tpm reports aggregate over the reports since the previous system release tag and tpm reports deck over the result.

<!-- req: R10.2 -->
### R10.2

The pipeline pushes a branch to the reports repository with the computed aggregate and deck and files one issue for its review, linked to the open stakeholder update issue (R6.2), and spends no model tokens. When the lead works that issue, the model drafts the summary prose from the reports and the aggregate and iterates on it with the lead (design.md D10). The lead's review is the approval; nothing is published unreviewed.

<!-- req: R10.3 -->
### R10.3

Run twice on the same tag, the pipeline pushes no second branch and files no second issue.

<!-- req: R11 -->
## R11 - Direction: forecasts

<!-- req: R11.1 -->
### R11.1

Once track-goals runs (06.0), the system release's aggregate also renders a roadmap forecast, the remaining stories of each epic spread over future sprints at velocity in Blocked-by order, and a burn-up per epic and initiative, story points done against total points, from the hierarchy documents and the ledger alone (srd045 R7.1). Both are committed beside the dashboard, regenerated each run, never edited, and labelled as forecasts, not baselines.

<!-- req: R11.2 -->
### R11.2

Until then, reports render no forecast and no burn-up (srd045 R7.2).

<!-- field: non_goals -->
## Non-goals

- Writing the prose of reports or summaries; the report audits (R6) are worked through the inner loop.
- Per-person productivity metrics of any kind.

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 -->
### AC1

init creates the reports repository layout, configuration, and CI in a local checkout once and never overwrites existing reports or summaries.

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
- [R1.2](#r12)
- [R1.3](#r13)

<!-- ac: AC2 -->
### AC2

validate passes a well-formed report and fails, non-zero, any report whose front matter disagrees with its body.

<!-- field: traces -->
#### Traces

- [R2.1](#r21)
- [R2.2](#r22)

<!-- ac: AC3 -->
### AC3

aggregate groups a week's or sprint's items by component and reports every listed signal, adds dashboard signals for a sprint, phrases no signal as a comparison of people, reports a team whose reports name no Sprint Goal of its own, counts no merge closing only recurring issues as unreported work, and with --push files each actionable signal as an unassigned issue.

<!-- field: traces -->
#### Traces

- [R3.1](#r31)
- [R3.2](#r32)
- [R3.3](#r33)
- [R3.4](#r34)
- [R3.5](#r35)
- [R3.6](#r36)

<!-- ac: AC4 -->
### AC4

deck renders an approved summary within the slide limit; every subcommand honours --json and --dry-run and never prompts.

<!-- field: traces -->
#### Traces

- [R4.1](#r41)
- [R5.1](#r51)

<!-- ac: AC5 -->
### AC5

The sprint update and the stakeholder update exist only as recurring issues whose acceptance criteria name validate, aggregate, and deck; each closes by its merge and is re-created for the next period; an actionable signal becomes a pushed issue.

<!-- field: traces -->
#### Traces

- [R6.1](#r61)
- [R6.2](#r62)
- [R6.3](#r63)

<!-- ac: AC6 -->
### AC6

status-issues leaves exactly one status issue per listed person for the coming week, recurring the last closed one or filing a fresh one, and files nothing on a second run; week returns the open Sprint Goal issues with due dates and the person's activity since the week began, the system repository's from its own first issue, with the issues its counted merges closed.

<!-- field: traces -->
#### Traces

- [R7.1](#r71)
- [R7.2](#r72)
- [R8.1](#r81)

<!-- ac: AC7 -->
### AC7

Each listed person receives one weekly status issue due the day after the reminder weekday, which mentions them without assigning them; worked with implement it produces a report with status, problems, and complete or not complete per open Sprint Goal that passes tpm reports validate, and it is re-created with the next due date.

<!-- field: traces -->
#### Traces

- [R9.1](#r91)
- [R9.2](#r92)
- [R9.3](#r93)
- [R9.4](#r94)

<!-- ac: AC8 -->
### AC8

A system release tag runs aggregate and deck in CI and pushes one branch with the computed aggregate and deck to the reports repository, with one issue for its review, linked to the stakeholder update issue; the summary prose is drafted by the model only during the lead's review; a second run on the same tag opens none.

<!-- field: traces -->
#### Traces

- [R10.1](#r101)
- [R10.2](#r102)
- [R10.3](#r103)

<!-- ac: AC9 -->
### AC9

Release 06.0's planning writes the forecast requirement items from R11.1; until then the interim rule of R11.2 holds.

<!-- field: traces -->
#### Traces

- [R11.1](#r111)
- [R11.2](#r112)
