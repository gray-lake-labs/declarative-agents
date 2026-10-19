---
id: srd045-backlog
title: "The backlog hierarchy: themes, initiatives, epics, features, and stories"
---

<!-- field: problem -->
## Problem

An agile reader expects a backlog shaped the way their tools shape it: a
theme names a strategic area, an initiative a multi-team outcome, an epic
a large body of work, an optional feature a capability inside it, and a
user story the unit of value a team finishes in one sprint, with tasks
beneath. Each item has one parent, progress rolls up from the stories,
story points and velocity live on stories, and the Product Owner orders
the backlog by rank.

Before release 04.0 the process had none of that under those names. A
program's intent was an initiative, the per-period commitment
was a Sprint Goal, and stories did not exist as such: their
acceptance criteria were scattered across SRD items and issues. A reader
had to translate every word, and the translation hid where value was
delivered.

This SRD defines the hierarchy as document types the corpus gate checks
(design.md theme, initiative, epic, feature, story), how progress and
velocity roll up through Parent links, how the Product Backlog is ordered,
and how Sprint Planning takes stories from it.

<!-- field: goals -->
## Goals

- G1: Every planning item is a theme, initiative, epic, feature, or story document, with at most one parent of the kind above it.
- G2: Progress, velocity, and forecasts are computed from story points and the ledger, never typed.
- G3: The Product Backlog is the ranked list of agreed stories, and Sprint Planning takes stories from it.

<!-- req: R1 -->
## R1 - Direction

<!-- req: R1.1 | status: retired | retired_in: 4 -->
### R1.1

Retired in 04.0; replaced by the requirement items R2 to R7 write when release 04.0 is planned (design.md D12).

<!-- req: R1.2 | status: retired | retired_in: 4 -->
### R1.2

Retired in 04.0; replaced by the requirement items R2 to R7 write when release 04.0 is planned (design.md D12).

<!-- req: R1.3 | status: retired | retired_in: 4 -->
### R1.3

Retired in 04.0; replaced by the requirement items R2 to R7 write when release 04.0 is planned (design.md D12).

<!-- req: R1.4 | status: retired | retired_in: 4 -->
### R1.4

Retired in 04.0; replaced by the requirement items R2 to R7 write when release 04.0 is planned (design.md D12).

<!-- req: R1.5 | status: retired | retired_in: 4 -->
### R1.5

Retired in 04.0; replaced by the requirement items R2 to R7 write when release 04.0 is planned (design.md D12).

<!-- req: R1.6 | status: retired | retired_in: 4 -->
### R1.6

Retired in 04.0; replaced by the requirement items R2 to R7 write when release 04.0 is planned (design.md D12).

<!-- req: R1.7 | status: retired | retired_in: 4 -->
### R1.7

Retired in 04.0; replaced by the requirement items R2 to R7 write when release 04.0 is planned (design.md D12).

<!-- req: R2 -->
## R2 - The documents

<!-- req: R2.1 -->
### R2.1

A program's planning items are documents of the five hierarchy types (design.md theme, initiative, epic, feature, story), each an SRD in the pinned parent release's Markdown format (D2) with a status of to_be_agreed, agreed, or retired. Only agreed documents are read by plan, tasks, and roll-up.

<!-- req: R2.2 -->
### R2.2

Themes and initiatives live in the system repository. An epic lives there when it spans teams and in a subsystem repository when one team owns it. Features and stories live in the owning team's repository.

<!-- req: R2.3 -->
### R2.3

No hierarchy document is a unit of work (D4). A story is executed by issues that name it in an Implements line (srd007 R6.5), and those issues and their tasks are the tracker's work.

<!-- req: R3 -->
## R3 - Parent and roll-up

<!-- req: R3.1 -->
### R3.1

A document names its parent as srd007 R7 states: an initiative's parent is a theme, an epic's an initiative, a feature's an epic, and a story's a feature or an epic. Only a feature must have a parent; an orphan story or epic is valid.

<!-- req: R3.2 -->
### R3.2

A story is done when every one of its acceptance criteria has passing latest evidence on a Sprint Goal (srd006 R10). Until track-goals runs, a story is done when the sprint-end issue of the sprint that listed it reports it complete (srd038 R2).

<!-- req: R3.3 -->
### R3.3

The progress of a feature, epic, or initiative is the story points of its done descendant stories over the story points of all its agreed descendant stories, found through Parent links; a theme reports the progress of its initiatives. Progress is computed, never typed.

<!-- req: R3.4 -->
### R3.4

Roll-up reads the documents of the repository it runs in and the documents their Parent links name by pinned URL. A story in another repository whose parent is in this one is not discovered from the parent; until a program registry names every repository, roll-up across repositories counts only what it reads, and says so.

<!-- req: R4 -->
## R4 - Stories

<!-- req: R4.1 -->
### R4.1

A story states who wants what and why in its context items as_a, i_want, and so_that, written from the user's view, and carries points, a relative estimate of its work that is never an estimate of a person (D5).

<!-- req: R4.2 -->
### R4.2

A story's acceptance criteria are its own, and each names its test and, as applicability, the environment where its evidence counts (D8). Its cites context item lists the SRD requirement items its criteria prove, each pinned (D9).

<!-- req: R4.3 -->
### R4.3

make-work turns a story selected onto a Sprint Goal into the issues that execute it, each with an Implements line naming the story.

<!-- req: R5 -->
## R5 - Product Backlog and Sprint Planning

<!-- req: R5.1 -->
### R5.1

A team's Product Backlog is its agreed stories that are not done, ordered by their rank context item, lowest first, then by id. The Product Owner sets rank; changing it is a merged docs change, which is how refinement is recorded.

<!-- req: R5.2 -->
### R5.2

At Sprint Planning, plan in the team's repository (srd031) lists the Product Backlog with the team's velocity and recommends the stories, in rank order, whose points fit it; the Product Owner decides. tpm plan backlog writes the chosen stories into the Sprint Goal's Sprint Backlog section and posts a planned event naming them (srd034 R3.5, srd006 R8).

<!-- req: R5.3 -->
### R5.3

A team's velocity is the story points of its stories done per sprint, read from the ledger, or from the sprint-end reports until track-goals runs. Until the team has finished a sprint, plan asks the Product Owner for an assumed velocity.

<!-- req: R6 -->
## R6 - Gate

<!-- req: R6.1 -->
### R6.1

The corpus gate checks every hierarchy document against its type: the required fields and context items, a parent of the right kind that resolves, link targets that resolve (srd007 R8.2), no parent or blocked_by cycle, a story's points, and a criterion without a test or an environment. The parent's corpus audit does not read these directories, so a magefiles gate inside mage audit checks them, as it checks initiatives (srd034 R2.1).

<!-- req: R7 -->
## R7 - Forecasts (direction)

<!-- req: R7.1 -->
### R7.1

Once track-goals runs (02.0), reports render a roadmap forecast, the remaining stories of each epic spread over future sprints at velocity in Blocked-by order, and a burn-up per epic and initiative, story points done against total points; both are committed beside the dashboard, regenerated each run, never edited, and labelled forecasts, not baselines (srd018 R11).

<!-- req: R7.2 -->
### R7.2

Until then, reports render no forecast and no burn-up.

<!-- field: non_goals -->
## Non-goals

- A baseline or a commitment date; a forecast is regenerated from current knowledge.
- Duration estimates; stories are estimated in relative points and dates come only from velocity.
- Estimating, ranking, or assigning a person (D5); rank orders stories, points belong to the work.
- Tracker-native epics, features, or stories; the hierarchy is documents, and the tracker holds the issues that execute stories.

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 | status: retired | retired_in: 4 -->
### AC1

Retired in 04.0 with the direction items it traced.

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
- [R1.2](#r12)
- [R1.3](#r13)
- [R1.4](#r14)
- [R1.5](#r15)
- [R1.6](#r16)
- [R1.7](#r17)

<!-- ac: AC2 -->
### AC2

Each planning item is one of the five document types in the repository R2.2 names, only agreed documents are read, and no hierarchy document is popped or merged as work; its stories are executed by issues that name them.

<!-- field: traces -->
#### Traces

- [R2.1](#r21)
- [R2.2](#r22)
- [R2.3](#r23)

<!-- ac: AC3 -->
### AC3

A story is done only when every criterion has passing evidence, or before track-goals when the sprint-end report says so; an epic with stories of 3, 5, and 8 points of which the 5 is done reports 5 of 16; a story in another repository is counted only when roll-up reads it, and the report says what it could not read.

<!-- field: traces -->
#### Traces

- [R3.1](#r31)
- [R3.2](#r32)
- [R3.3](#r33)
- [R3.4](#r34)

<!-- ac: AC4 -->
### AC4

Every story has a statement, points, and criteria naming a test and an environment, and its cites are pinned; make-work's issues for a selected story carry Implements naming it.

<!-- field: traces -->
#### Traces

- [R4.1](#r41)
- [R4.2](#r42)
- [R4.3](#r43)

<!-- ac: AC5 -->
### AC5

plan lists a team's not-done agreed stories by rank, recommends those that fit velocity, asks an assumed velocity for a team with no finished sprint, and writes the chosen stories to the Sprint Goal with a planned event naming them; velocity equals the points of the stories done in the last sprint.

<!-- field: traces -->
#### Traces

- [R5.1](#r51)
- [R5.2](#r52)
- [R5.3](#r53)

<!-- ac: AC6 -->
### AC6

mage audit fails on a hierarchy document with a missing field, a parent of the wrong kind or that does not resolve, an unresolved link, a parent or blocked_by cycle, a story without points, or a criterion without a test or an environment.

<!-- field: traces -->
#### Traces

- [R6.1](#r61)

<!-- ac: AC7 -->
### AC7

Release 02.0's planning writes the forecast items from R7.1; until then the interim rule of R7.2 holds.

<!-- field: traces -->
#### Traces

- [R7.1](#r71)
- [R7.2](#r72)
