---
id: srd006-events
title: Comment events
---

<!-- field: problem -->
## Problem

Most of what passes between components is information, not work: an
artifact was released, a consumer finished adapting to it, a version was
superseded, someone has a question, a test produced evidence. Filing each
of those as an issue fills the backlog with things nobody owes. Sending them by mail or chat takes them
out of the tracker, where nobody can find them next to the work they are
about.

Events are comments on the issue they concern, with a marker that scripts
can read and a mention when a person must act. They inform; they never
change an issue's state.

Three events make an objective issue its own ledger. The road-map is the
plan and tags are releases; the objective issue records when its work was
planned, every piece of test evidence that bears on it, and when it was
met. Replaying those events reconstructs the objective's state without
reading anything else.

<!-- field: goals -->
## Goals

- G1: Information between components travels as comments on the issue it concerns.
- G2: Scripts can recognize, query, and deduplicate events without parsing prose.
- G3: A person who must act on an event is told through the tracker's own notification.
- G4: An objective's state (planned, evidence so far, met) is reconstructable from the events on its issue alone.

<!-- req: R1 -->
## R1 - Marker format

<!-- req: R1.1 -->
### R1.1

An event is a comment whose body contains exactly one marker of the form <!-- tpm program event=NAME key=value ... -->, where NAME is an event in this SRD's vocabulary and every value is free of spaces.

<!-- req: R1.2 -->
### R1.2

The text of the comment, outside the marker, is written for a person and states the event in one or two sentences with links.

<!-- req: R1.3 -->
### R1.3

A comment without a marker is ordinary discussion. Scripts ignore it.

<!-- req: R2 -->
## R2 - event=release

<!-- req: R2.1 -->
### R2.1

Posted by handoff on each open recurring issue subscribed to an artifact, and on each open consume issue for an earlier version of it, when the artifact is tagged.

<!-- req: R2.2 -->
### R2.2

Required keys are artifact, version, tag, and breaking (true or false).

<!-- req: R3 -->
## R3 - event=downstream-done

<!-- req: R3.1 -->
### R3.1

Posted by handoff on the issue named in a closed issue's Upstream line (srd007 R1), when the merge that closed it is merged.

<!-- req: R3.2 -->
### R3.2

Required keys are from (the closed issue) and mr (the closing merge request). It mentions the upstream issue's assignee.

<!-- req: R4 -->
## R4 - event=superseded

<!-- req: R4.1 -->
### R4.1

Posted by handoff on an open consume issue when a later version of the same artifact is tagged before the issue closes.

<!-- req: R4.2 -->
### R4.2

Required keys are artifact, old, and new. It mentions the consume issue's assignee, if any.

<!-- req: R5 -->
## R5 - event=question

<!-- req: R5.1 -->
### R5.1

Posted by a person on any issue to ask its assignee or author something that does not block work.

<!-- req: R5.2 -->
### R5.2

Required key is id, unique within the issue. It mentions the person asked.

<!-- req: R6 -->
## R6 - event=answer

<!-- req: R6.1 -->
### R6.1

Posted by a person in reply to a question event.

<!-- req: R6.2 -->
### R6.2

Required key is to, the id of the question answered. It mentions the asker.

<!-- req: R7 -->
## R7 - Rules across events

<!-- req: R7.1 -->
### R7.1

An event that requires a person to act mentions that person, so the tracker creates a to-do for them. An event that only informs does not mention anyone.

<!-- req: R7.2 -->
### R7.2

A script posts an event only if no comment on the same issue already carries a marker with the same name and the same required keys. Posting is idempotent.

<!-- req: R7.3 -->
### R7.3

No event changes a label, an assignee, a milestone, or the open or closed state of any issue. A script that changes state alongside an event does so under srd041 R1–R32, not because of the event.

<!-- req: R7.4 -->
### R7.4

The vocabulary is closed. A program adds an event name only through a plane::control change to its policy files (srd001 R2).

<!-- req: R8 -->
## R8 - event=planned

<!-- req: R8.1 -->
### R8.1

Posted on an objective issue when tasks pushes it, and again whenever plan places it in a different road-map release.

<!-- req: R8.2 -->
### R8.2

Required keys are release (the road-map release the objective counts toward), requirement (each upstream requirement it satisfies, as a pinned URL, srd007 R5), and environment (where its evidence counts). requirement may repeat.

<!-- req: R9 -->
## R9 - event=evidence

<!-- req: R9.1 -->
### R9.1

Posted on an objective issue by track-goals each time a test named in one of the objective's acceptance criteria has run in the named environment on a recorded version.

<!-- req: R9.2 -->
### R9.2

Required keys are requirement (the pinned URL of the requirement the test proves), test (the test's identifier), environment, version (the pinned set of releases under test), and result (pass or fail). A run that produced no result posts nothing; missing evidence is reported, never recorded as a pass.

<!-- req: R9.3 -->
### R9.3

Evidence events are append-only. A later run posts a new event; it never edits or deletes an earlier one.

<!-- req: R10 -->
## R10 - event=met

<!-- req: R10.1 -->
### R10.1

Posted on an objective issue by track-goals when, for every requirement named by its planned event, the latest evidence event in the named environment has result pass.

<!-- req: R10.2 -->
### R10.2

Required keys are release and version (the pinned set on which the objective was met). It mentions the objective's owner.

<!-- req: R10.3 -->
### R10.3

A later evidence event with result fail on a requirement of a met objective posts a new planned event for the same release, which reopens the ledger; met is never edited away.

<!-- req: R11 -->
## R11 - Ledger replay

<!-- req: R11.1 -->
### R11.1

The state of an objective is computed from its events in order: planned when the latest planned event has no later met event; proving while some but not all of its requirements have a passing latest evidence event; met when the latest met event is later than the latest planned event.

<!-- req: R11.2 -->
### R11.2

No other source (labels, assignee, closed state, checklists) is read to decide whether an objective is met.

<!-- field: non_goals -->
## Non-goals

- Chat or mail integration; the tracker's notifications are the delivery.
- Threading or reactions; events are flat comments.
- Events on merge requests; events attach to issues.

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 -->
### AC1

Every event comment carries exactly one well-formed marker with a known name and human-readable text; comments without a marker are never read by a script.

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
- [R1.2](#r12)
- [R1.3](#r13)

<!-- ac: AC2 -->
### AC2

Tagging an artifact posts one release event with artifact, version, tag, and breaking on each subscribing recurring issue and each open consume issue for an earlier version.

<!-- field: traces -->
#### Traces

- [R2.1](#r21)
- [R2.2](#r22)

<!-- ac: AC3 -->
### AC3

Merging a request that closes an issue with an Upstream line posts one downstream-done event with from and mr on the upstream issue, mentioning its assignee.

<!-- field: traces -->
#### Traces

- [R3.1](#r31)
- [R3.2](#r32)

<!-- ac: AC4 -->
### AC4

A later tag of an artifact posts superseded with artifact, old, and new on each still-open consume issue for the earlier version.

<!-- field: traces -->
#### Traces

- [R4.1](#r41)
- [R4.2](#r42)

<!-- ac: AC5 -->
### AC5

A question event carries a unique id and mentions the person asked; an answer carries the question's id and mentions the asker.

<!-- field: traces -->
#### Traces

- [R5.1](#r51)
- [R5.2](#r52)
- [R6.1](#r61)
- [R6.2](#r62)

<!-- ac: AC6 -->
### AC6

Running the same handoff twice posts no duplicate event; no event changes any label, assignee, milestone, or open state; an unknown event name is rejected unless the program's policy adds it; action-requiring events create a to-do and informational ones do not.

<!-- field: traces -->
#### Traces

- [R7.1](#r71)
- [R7.2](#r72)
- [R7.3](#r73)
- [R7.4](#r74)

<!-- ac: AC7 -->
### AC7

Pushing an objective posts one planned event with release, each requirement as a pinned URL, and environment; moving it to another release posts a new planned event.

<!-- field: traces -->
#### Traces

- [R8.1](#r81)
- [R8.2](#r82)

<!-- ac: AC8 -->
### AC8

Each test run named by an objective's criteria in the named environment posts one evidence event with requirement, test, environment, version, and result; a run with no result posts nothing; earlier evidence events are never edited.

<!-- field: traces -->
#### Traces

- [R9.1](#r91)
- [R9.2](#r92)
- [R9.3](#r93)

<!-- ac: AC9 -->
### AC9

met is posted, mentioning the owner, only when every planned requirement's latest evidence in the named environment passed; a later failing evidence event posts a new planned event and leaves the earlier met in place.

<!-- field: traces -->
#### Traces

- [R10.1](#r101)
- [R10.2](#r102)
- [R10.3](#r103)

<!-- ac: AC10 -->
### AC10

Replaying any sequence of planned, evidence, and met events yields planned, proving, or met as R11.1 defines, with the same answer as track-goals, and without reading labels, assignee, or closed state.

<!-- field: traces -->
#### Traces

- [R11.1](#r111)
- [R11.2](#r112)
