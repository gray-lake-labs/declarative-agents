---
id: srd019-track-goals
title: "Script: track-goals (tpm goals ...)"
---

<!-- field: problem -->
## Problem

An objective is met when the tests its requirements name pass in the
environment it names, on a recorded version. That evidence is spread across
repositories: the objective issue sits in a subsystem repository, the
requirement it satisfies in the system repository at a tagged version, the
test results in the test subsystem's pipelines. Nobody reconciles that by
hand every night, so objectives are declared met when their issues close,
and gaps surface at the review.

track-goals is the audit that does the reconciling. It follows pinned
references from the system repository down through every subsystem and
component, collects the evidence each objective's acceptance criteria name,
records it on the objective as ledger events, closes the objective when the
evidence passes, keeps every issue's waiting state true to its links, and
fails when evidence is missing or a reference does not resolve. Daily
scrum, sprint review, and the retrospective read its output.

<!-- field: goals -->
## Goals

- G1: Every open objective's ledger reflects the latest evidence for each of its requirements after one run.
- G2: An objective closes when, and only when, its evidence passes.
- G3: Missing evidence and unresolved references fail the audit instead of passing silently.
- G4: The waiting flag on every open issue matches its links after one run.

<!-- req: R1 -->
## R1 - Triggers and common behavior

<!-- req: R1.1 | status: retired | retired_in: 1.2 -->
### R1.1

Retired in 01.2 (design.md D12).

<!-- req: R1.2 | status: retired | retired_in: 1.2 -->
### R1.2

Retired in 01.2 (design.md D12).

<!-- req: R2 -->
## R2 - Follow references

<!-- req: R2.1 | status: retired | retired_in: 1.2 -->
### R2.1

Retired in 01.2 (design.md D12).

<!-- req: R2.2 | derived_from: [{"repo": "github.com/gray-lake-labs/declarative-agents", "release": "v0.20261018.1", "srd": "srd014-spec-graph", "item": "R3.16", "reason": "track-goals reports unresolvable references with the parent's resolution rule: local, read-only, at the cited release."}] | status: retired | retired_in: 1.2 -->
### R2.2

Retired in 01.2 (design.md D12).

<!-- req: R2.3 | status: retired | retired_in: 1.2 -->
### R2.3

Retired in 01.2 (design.md D12).

<!-- req: R3 -->
## R3 - Collect evidence

<!-- req: R3.1 | status: retired | retired_in: 1.2 -->
### R3.1

Retired in 01.2 (design.md D12).

<!-- req: R3.2 | status: retired | retired_in: 1.2 -->
### R3.2

Retired in 01.2 (design.md D12).

<!-- req: R3.3 | status: retired | retired_in: 1.2 -->
### R3.3

Retired in 01.2 (design.md D12).

<!-- req: R4 -->
## R4 - Close objectives and drained tracking issues

<!-- req: R4.1 | status: retired | retired_in: 1.2 -->
### R4.1

Retired in 01.2 (design.md D12).

<!-- req: R4.2 | status: retired | retired_in: 1.2 -->
### R4.2

Retired in 01.2 (design.md D12).

<!-- req: R4.3 | status: retired | retired_in: 1.2 -->
### R4.3

Retired in 01.2 (design.md D12).

<!-- req: R5 -->
## R5 - Waiting states

<!-- req: R5.1 | status: retired | retired_in: 1.2 -->
### R5.1

Retired in 01.2 (design.md D12).

<!-- req: R5.2 | status: retired | retired_in: 1.2 -->
### R5.2

Retired in 01.2 (design.md D12).

<!-- req: R5.3 | status: retired | retired_in: 1.2 -->
### R5.3

Retired in 01.2 (design.md D12).

<!-- req: R5.4 | status: retired | retired_in: 1.2 -->
### R5.4

Retired in 01.2 (design.md D12).

<!-- req: R5.5 | status: retired | retired_in: 1.2 -->
### R5.5

Retired in 01.2 (design.md D12).

<!-- req: R5.6 | status: retired | retired_in: 1.2 -->
### R5.6

Retired in 01.2 (design.md D12).

<!-- req: R6 -->
## R6 - tpm goals verify

<!-- req: R6.1 | status: retired | retired_in: 1.2 -->
### R6.1

Retired in 01.2 (design.md D12).

<!-- req: R6.2 | status: retired | retired_in: 1.2 -->
### R6.2

Retired in 01.2 (design.md D12).

<!-- req: R7 -->
## R7 - tpm goals status

<!-- req: R7.1 | status: retired | retired_in: 1.2 -->
### R7.1

Retired in 01.2 (design.md D12).

<!-- req: R7.2 | status: retired | retired_in: 1.2 -->
### R7.2

Retired in 01.2 (design.md D12).

<!-- req: R7.3 | status: retired | retired_in: 1.2 -->
### R7.3

Retired in 01.2 (design.md D12).

<!-- req: R7.4 | status: retired | retired_in: 1.2 -->
### R7.4

Retired in 01.2 (design.md D12).

<!-- req: R8 -->
## R8 - Audit result

<!-- req: R8.1 | status: retired | retired_in: 1.2 -->
### R8.1

Retired in 01.2 (design.md D12).

<!-- req: R9 -->
## R9 - Direction

<!-- req: R9.1 -->
### R9.1

track-goals follows each objective's pinned references, records test results as evidence events, and closes an objective with a met event when every requirement's latest evidence in its environment passed.

<!-- req: R9.2 -->
### R9.2

It keeps waiting states true from Upstream, Requires, and Cites dependencies, and reports goals status (met, proving, slipped) for planning.

<!-- req: R9.3 -->
### R9.3

Until it runs, an objective closes as reported complete by its repository's sprint-end merge (srd038 R2), and planning takes slipped objectives from the aggregate's slipped-commitment signal (srd018 R3.2).

<!-- req: R9.4 -->
### R9.4

Once a release plans srd045, track-goals also reports each objective's progress: the points of the epic items it cites whose evidence passed, or whose citing issues closed where no evidence is named yet, over all the points it cites. Progress reads 100 percent only when the ledger replays to met, and an objective that cites no epic reports no percentage.

<!-- field: non_goals -->
## Non-goals

- Running tests; the test subsystem's pipelines run them (srd008 R4).
- Promoting pinned sets between environments; that is CI/CD in the test subsystem.
- Judging whether an Increment is acceptable; that is the Product Owner at sprint-review.
- The channel for evidence from customer R&D environments; recorded as an open decision in SPECIFICATIONS.

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 | status: retired | retired_in: 1.2 -->
### AC1

Retired in 01.2 (design.md D12).

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
- [R1.2](#r12)

<!-- ac: AC2 | status: retired | retired_in: 1.2 -->
### AC2

Retired in 01.2 (design.md D12).

<!-- field: traces -->
#### Traces

- [R2.1](#r21)
- [R2.2](#r22)
- [R2.3](#r23)

<!-- ac: AC3 | status: retired | retired_in: 1.2 -->
### AC3

Retired in 01.2 (design.md D12).

<!-- field: traces -->
#### Traces

- [R3.1](#r31)
- [R3.2](#r32)
- [R3.3](#r33)

<!-- ac: AC4 | status: retired | retired_in: 1.2 -->
### AC4

Retired in 01.2 (design.md D12).

<!-- field: traces -->
#### Traces

- [R4.1](#r41)
- [R4.2](#r42)
- [R4.3](#r43)

<!-- ac: AC5 | status: retired | retired_in: 1.2 -->
### AC5

Retired in 01.2 (design.md D12).

<!-- field: traces -->
#### Traces

- [R5.1](#r51)
- [R5.2](#r52)
- [R5.3](#r53)
- [R5.4](#r54)
- [R5.5](#r55)
- [R5.6](#r56)

<!-- ac: AC6 | status: retired | retired_in: 1.2 -->
### AC6

Retired in 01.2 (design.md D12).

<!-- field: traces -->
#### Traces

- [R6.1](#r61)
- [R6.2](#r62)
- [R7.1](#r71)
- [R7.2](#r72)
- [R7.3](#r73)
- [R7.4](#r74)

<!-- ac: AC7 | status: retired | retired_in: 1.2 -->
### AC7

Retired in 01.2 (design.md D12).

<!-- field: traces -->
#### Traces

- [R8.1](#r81)

<!-- ac: AC8 -->
### AC8

Release 02.0's planning writes track-goals' requirements from R9; until then the interim rule of R9.3 holds.

<!-- field: traces -->
#### Traces

- [R9.1](#r91)
- [R9.2](#r92)
- [R9.3](#r93)
- [R9.4](#r94)
