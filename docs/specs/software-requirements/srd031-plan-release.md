---
id: srd031-plan-release
title: "Skill: plan"
---

<!-- field: problem -->
## Problem

Before each three-week period, someone has to say which requirements the
next release will prove. At system level that is the technical program
manager choosing the system requirements the next system release covers;
at subsystem level it is the Product Owner choosing which of the
subsystem's requirements, written to satisfy upstream objectives, the next
subsystem release covers. Decided in a meeting, the choice lives in slides,
and the road-map says something else.

plan reads the road-map, the requirements or, in the system
repository, the agreed objective documents, and what track-goals says is
already met (before track-goals runs, what the period's reports say
slipped), asks the owner which requirements go into the next
release, and records the answer in the road-map and on the objectives it
moves. It works the same at both levels, because every repository carries
the same corpus.

<!-- field: goals -->
## Goals

- G1: Each road-map release names the requirements it will prove before its period starts.
- G2: Only requirements whose acceptance criteria name a test and an environment enter a release.
- G3: The same skill serves the system and every subsystem.

<!-- req: R1 -->
## R1 - Trigger

<!-- req: R1.1 -->
### R1.1

Started before each three-week period by the technical program manager in the system repository, and by each Product Owner in a subsystem repository after the system release is declared.

<!-- req: R2 -->
## R2 - Reads

<!-- req: R2.1 -->
### R2.1

The repository's road-map (docs/road-map.md): its releases, their requirements or, in the system repository, their objective items (deliverables, design.md schema.roadmap), and their status.

<!-- req: R2.2 -->
### R2.2

The repository's SRDs, in the system repository the agreed objective documents (srd034 R1), and at subsystem level the open objective issues filed in the repository, with the upstream requirements they cite by pinned URL (srd007 R5).

<!-- req: R2.3 -->
### R2.3

tpm goals status (srd019) for what is met, proving, or slipped, and the latest release tags. Before track-goals runs in the program, the period aggregate's slipped-commitment signal (srd018 R3.2) names what slipped.

<!-- req: R2.4 -->
### R2.4

Waiting dependencies as track-goals explains them (srd019), so a requirement that cannot be proved this period is visible before it is chosen.

<!-- req: R3 -->
## R3 - Asks

<!-- req: R3.1 -->
### R3.1

Asks the owner which requirements, or in the system repository which objective items, enter the next release, recommending slipped ones first, then those that satisfy the objectives with the earliest upstream release, then the rest in road-map order. tpm plan candidates computes the ranking.

<!-- req: R3.2 -->
### R3.2

Asks for the release's version and end date, recommending the next minor version and the end of the three-week period.

<!-- req: R4 -->
## R4 - Pushes and posts

<!-- req: R4.1 -->
### R4.1

The release entry in docs/road-map.md, listing its requirements by id, or in the system repository its objective items in deliverables and its milestone and tag in release.publication, written by tpm plan write in the worktree of a plane::management issue filed with specify, and merged through the inner loop, so the plan is a reviewed change.

<!-- req: R4.2 -->
### R4.2

The tracker milestone named for the release, with its ISO weeks in its description and its end date, created if missing and updated if present (srd001 R7.1).

<!-- req: R4.3 -->
### R4.3

A planned event (srd006 R8) on each objective issue whose release changed.

<!-- req: R4.4 -->
### R4.4

Returns each requirement or objective item whose acceptance criteria lack a test or an environment to its owner with a comment naming what is missing, instead of placing it.

<!-- req: R4.5 -->
### R4.5

In the system repository, once the release entry merges, a documents-only release of the system repository, gated by mage audit and tagged by tpm release tag with the tag release.publication names (srd015 R2, srd034 R2.3), so tasks pins its Cites lines to the agreed text.

<!-- req: R4.6 -->
### R4.6

tpm plan push dates the milestone, sets the sprint issue (srd001 R7) with one checklist item per objective, posts the planned events, and files one sprint-end issue (srd038 R1.1) in each releasing repository due on the milestone end date; a second run updates and creates nothing.

<!-- req: R5 -->
## R5 - Refusals

<!-- req: R5.1 -->
### R5.1

Refuses to place a requirement without a named test and environment, and refuses to assign any issue to a person.

<!-- req: R5.2 -->
### R5.2

Refuses to create work issues; Developers turn the plan into issues with make-work.

<!-- field: non_goals -->
## Non-goals

- Estimating capacity per person.
- Pushing objectives downstream; that is tasks (srd022).

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 -->
### AC1

Facts come from the road-map, the SRDs, open objectives with pinned citations, track-goals status, and dependency explanations; the requirements and the version are asked with recommendations; the skill runs at system and subsystem level alike.

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
- [R2.1](#r21)
- [R2.2](#r22)
- [R2.3](#r23)
- [R2.4](#r24)
- [R3.1](#r31)
- [R3.2](#r32)

<!-- ac: AC2 -->
### AC2

The release entry is merged through the inner loop, the system repository's plan is tagged, the milestone exists with the end date, the sprint issue and the period's sprint-end issues exist once, each moved objective gets a planned event, and a requirement or objective item without test or environment is returned with a comment.

<!-- field: traces -->
#### Traces

- [R4.1](#r41)
- [R4.2](#r42)
- [R4.3](#r43)
- [R4.4](#r44)
- [R4.5](#r45)
- [R4.6](#r46)

<!-- ac: AC3 -->
### AC3

No requirement without test and environment is placed, no issue is assigned, and no work issue is created.

<!-- field: traces -->
#### Traces

- [R5.1](#r51)
- [R5.2](#r52)
