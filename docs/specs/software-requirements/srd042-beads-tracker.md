---
id: srd042-beads-tracker
title: "Beads tracker: the program's facts live in a repository"
---

<!-- field: design_notes -->
## Design notes

| GitLab concept | beads concept |
| --- | --- |
| issue | issue |
| scoped label | label, same axis grammar |
| milestone | release field |
| assignee | assignee |
| note | comment |
| merge request | local merge; closing commit cited on the issue |
| project path | repository field on the issue |
| group hierarchy | one tracker repository, queries scoped by field |

<!-- field: problem -->
## Problem

Every fact the program runs on — issues, states, labels, comments,
milestones — lives in a hosted GitLab instance. That buys a web UI and
merge requests, and costs three things the program feels daily: an
external service to provision and keep reachable (the Terraform roots,
service accounts, and CI wiring exist only to shape it), tracker text
stored and indexed outside the repository, and a hard dependency that
makes every tpm command a network call. The skills already merge
branches locally, and the tracker data is small, structured, and
line-oriented. This SRD states the requirements for beads as the
program's tracker: issues as data in a dedicated shared repository,
synchronized through git, read and written by tpm through the same
tracker boundary the GitLab client serves today.

<!-- req: R1 -->
## R1 - The tracker boundary holds

<!-- req: R1.1 -->
### R1.1

tpm's commands reach the tracker only through the tracker interface the
GitLab client implements today. The beads backend is a second
implementation of that interface; no command, skill, or gate knows which
backend serves it.

<!-- req: R1.2 -->
### R1.2

The state machine of srd041 is backend-neutral: every transition, actor,
and invariant holds unchanged under beads. A state is read from issue
fields, never inferred.

<!-- req: R1.3 -->
### R1.3

While both backends exist, a program names its backend in its
configuration document; 03.0 completes when the GitLab client, the
platform Terraform, and the rig's GitLab dependency are removed.

<!-- req: R2 -->
## R2 - Mapping

<!-- req: R2.1 -->
### R2.1

A GitLab issue maps to a beads issue: title, description, state, one
assignee, labels, and comments carry over one to one. The scoped labels
of srd001 become beads labels with the same axis grammar; the lint of
srd013 enforces exclusivity exactly as it does on GitLab Free.

<!-- req: R2.2 -->
### R2.2

A milestone maps to an issue field holding the release or sprint
identifier. Anything srd031 writes to a milestone today it writes to
that field.

<!-- req: R2.3 -->
### R2.3

A merge request maps to nothing: delivery is a local merge to main whose
closing commit cites the issue id, and the closing actor of srd041 R7
becomes the merge commit observed by the implement skill, which closes
the issue and records the commit hash as the closing comment.

<!-- req: R2.4 -->
### R2.4

A project path (group/subgroup/repo) maps to a repository field on the
issue. One tracker holds every repository's issues; a query scoped to a
repository replaces a query scoped to a project.

<!-- req: R3 -->
## R3 - The shared tracker repository

<!-- req: R3.1 -->
### R3.1

The program's beads database lives in one dedicated git repository, the
tracker repository, separate from every code repository. It is the
single source of truth for program facts; a working clone's local beads
state is a cache. Its .beads/config.yaml sets the program's issue
prefix, once, when tpm init seeds it: every issue's ID is that prefix and
eight hex digits, and tpm when the file sets none. An ID is never renamed.

<!-- req: R3.2 -->
### R3.2

Every tpm command that reads the tracker synchronizes from the tracker
repository before reading; every command that writes synchronizes to it
after writing. The protocol, its conflict rules, and its offline
behavior are eng04-beads-sync.

<!-- req: R3.3 -->
### R3.3

A report produced from a clone that could not synchronize says so,
naming the time of its last successful synchronization.

<!-- req: R4 -->
## R4 - What leaves

<!-- req: R4.1 -->
### R4.1

The platform Terraform (groups, projects, memberships, recurring-issue
seeding) is retired; recurring issues of srd014 are seeded by tpm
against the tracker repository.

<!-- req: R4.2 -->
### R4.2

The regression rig builds fixture tracker repositories instead of
GitLab subgroups; eng01 is rewritten when 03.0 is planned.

<!-- field: goals -->
## Goals

- G1: Every program fact lives in a repository the program owns, synchronized through git, with no hosted tracker in the loop.
- G2: tpm behaves identically over beads and GitLab while both exist, because both serve one tracker interface.

<!-- field: non_goals -->
## Non-goals

- The synchronization protocol's mechanics and conflict rules; those are eng04-beads-sync.
- A web UI over the tracker repository.
- Migrating historical GitLab data; a program adopts beads at a release boundary with open issues re-filed.

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 -->
### AC1

Every tpm command passes its tests against a fixture beads tracker with
no GitLab reachable, and the srd041 state machine replays identically on
both backends.

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
- [R1.2](#r12)
- [R1.3](#r13)

<!-- ac: AC2 -->
### AC2

An issue created on GitLab and its beads mapping carry the same title,
state, labels, assignee, and comments; a closed issue on beads cites the
closing commit; a repository-scoped query returns the same issues a
project-scoped query returns today.

<!-- field: traces -->
#### Traces

- [R2.1](#r21)
- [R2.2](#r22)
- [R2.3](#r23)
- [R2.4](#r24)

<!-- ac: AC3 -->
### AC3

A command run against a stale clone synchronizes before reading; a write
reaches the tracker repository before the command reports success; a
command that cannot synchronize says so and names the last good
synchronization.

<!-- field: traces -->
#### Traces

- [R3.1](#r31)
- [R3.2](#r32)
- [R3.3](#r33)

<!-- ac: AC4 -->
### AC4

Release 03.0 ends with no Terraform platform root, no GitLab client, and
a rig that provisions fixture tracker repositories.

<!-- field: traces -->
#### Traces

- [R4.1](#r41)
- [R4.2](#r42)
