---
id: srd022-tasks
title: "Skill: tasks"
---

<!-- field: problem -->
## Problem

Once a release is planned, the owners below have to know what they owe it.
A system release needs each subsystem to satisfy some system requirements;
a subsystem release may need work from its downstream component
repositories; every release needs the test subsystem to provide the suites
and environments that will prove it. Passed down in meetings, these
obligations lose the exact requirement and version they came from, and the
owner below writes requirements against a moving target.

tasks reads the release just planned and files one Sprint Goal issue
for each downstream team, citing the initiative items it serves by URL
pinned to the release tag and naming the environment where its evidence
counts. Each Sprint Goal is the ledger track-goals writes evidence to, and
the team's Sprint Planning lists on it the stories it selected (srd034,
srd045). Before release 04.0 a Sprint Goal was called an objective issue.
Invoking tasks is the decision to file them: it takes every owner
and environment tpm can compute, asks only for what it cannot, and pushes
without a further confirmation.

<!-- field: goals -->
## Goals

- G1: Every initiative item of a planned release reaches the downstream team that must serve it, on that team's Sprint Goal.
- G2: Every Sprint Goal cites the initiative items it serves at an exact tag and names the environment where it counts.
- G3: The test subsystem receives a Sprint Goal for the suites and environments each release needs.

<!-- req: R1 -->
## R1 - Trigger

<!-- req: R1.1 -->
### R1.1

Started after plan (srd031) by the technical program manager for a system release, or by a Product Owner whose subsystem has downstream repositories.

<!-- req: R2 -->
## R2 - Reads

<!-- req: R2.1 -->
### R2.1

The release just planned in docs/road-map.md and its release tag, and the requirements it lists.

<!-- req: R2.2 -->
### R2.2

The subsystem SRDs (design.md subsystem_srd) for the downstream owners and their repositories, and the environment SRDs in the test-subsystem repository.

<!-- req: R2.3 -->
### R2.3

Open Sprint Goal issues (and objective issues, read as Sprint Goals) already in the downstream repositories for this release, so a Sprint Goal is updated, not duplicated.

<!-- req: R3 -->
## R3 - Asks only what is not computed

<!-- req: R3.1 -->
### R3.1

Takes the owner tpm recommends for each initiative item: the Product Owner whose subsystem SRD declares a repository the item's delivered_by names (srd034 R1.4). Asks which downstream team serves an item only when none is recommended, offering the repositories the subsystem SRDs declare.

<!-- req: R3.2 -->
### R3.2

Takes the environment the initiative's acceptance criteria name, else dev, and asks nothing about it.

<!-- req: R3.3 -->
### R3.3

Asks nothing the road-map, the SRDs, or the tracker already answers, and asks for no confirmation before pushing.

<!-- req: R4 -->
## R4 - Pushes and posts

<!-- req: R4.1 -->
### R4.1

One Sprint Goal issue per downstream team and release, filed with specify in the team's repository, in do-work's input format, with one Cites line per initiative item it serves pinned to the release tag (srd007 R5), one Environment line, and an empty Sprint Backlog section (srd034 R3.5).

<!-- req: R4.2 -->
### R4.2

One Sprint Goal issue to the Scrum Master in the test-subsystem repository naming the suites and environments the release needs to produce its evidence.

<!-- req: R4.3 -->
### R4.3

A planned event (srd006 R8) on each Sprint Goal it creates or updates.

<!-- req: R5 -->
## R5 - Refusals

<!-- req: R5.1 -->
### R5.1

Refuses to file a Sprint Goal with an unpinned citation or without an environment.

<!-- req: R5.2 -->
### R5.2

Refuses to write the downstream owner's requirements; the owner writes those with do-work and places them with plan.

<!-- req: R5.3 -->
### R5.3

Refuses to copy names or confidential content from inputs.

<!-- field: non_goals -->
## Non-goals

- Choosing what goes in the release; that is plan (srd031).
- Deciding that a Sprint Goal is met; that is track-goals (srd019).
- Selecting stories; that is the team's Sprint Planning (srd045 R5.2).

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 -->
### AC1

Facts come from the planned release, subsystem and environment SRDs, and existing Sprint Goals; the recommended owner and the environment are taken without asking; only an initiative item with no recommended owner is asked about; no confirmation is asked before the push.

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
- [R2.1](#r21)
- [R2.2](#r22)
- [R2.3](#r23)
- [R3.1](#r31)
- [R3.2](#r32)
- [R3.3](#r33)

<!-- ac: AC2 -->
### AC2

Each downstream team receives one Sprint Goal per release with Cites lines pinned to the release tag, an Environment line, and an empty Sprint Backlog section; the test subsystem receives one Sprint Goal for the release's suites and environments; each carries a planned event; a second run creates no duplicate.

<!-- field: traces -->
#### Traces

- [R4.1](#r41)
- [R4.2](#r42)
- [R4.3](#r43)

<!-- ac: AC3 -->
### AC3

An unpinned citation or a missing environment is refused; no downstream requirement is written by the skill; no name or confidential content is copied.

<!-- field: traces -->
#### Traces

- [R5.1](#r51)
- [R5.2](#r52)
- [R5.3](#r53)
