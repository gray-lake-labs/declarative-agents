---
id: srd045-epics
title: "Epics: the Product Backlog, velocity, and forecasts"
---

<!-- field: problem -->
## Problem

An objective document says what the program commits to for a horizon of
periods, but nothing holds the work behind those commitments: how much of
it there is, what it is estimated at, what waits on what, and how much of it
is left. Without that, plan chooses each period's objectives from memory,
nobody can say when a body of work will be done, and every forecast is a
chart someone draws by hand and that contradicts the tracker a week later.

Agile practice has a name for the missing piece. The Product Backlog holds
the work, ordered by its owner and estimated in relative points; velocity,
the points a team finishes per sprint, turns what is left into a forecast;
refinement changes the backlog every sprint as the team learns. This SRD
gives the process those three: the epic document as the backlog, velocity
as a fact read from the ledger, and forecasts regenerated from both.

<!-- field: goals -->
## Goals

- G1: Every body of work spanning several sprints is an epic document whose backlog items carry points and dependencies, ordered by its owner.
- G2: Each period's objectives are derived from what the epics hold and what the ledger says is already met, so remaining work is computed, never recalled.
- G3: Velocity, progress, and forecasts are read from the ledger and the epics; nobody types a date, a percentage, or a burn-down.

<!-- req: R1 -->
## R1 - Direction

<!-- req: R1.1 -->
### R1.1

An epic document (design.md epic) is the Product Backlog for a body of work spanning several sprints: backlog items in its owner's order, each with points, the items it depends on, and an optional anchor. It lives in the system repository beside the objective documents, and a changed epic is a reviewed merge through the audit gate (srd034 R2), which is how refinement is recorded.

<!-- req: R1.2 -->
### R1.2

An objective item may cite the epic items it draws down, by URL pinned to a release tag (srd007 R5). An epic item is drawn down when an objective that cites it is met. An objective that cites no epic, and an epic no objective cites yet, are both valid.

<!-- req: R1.3 -->
### R1.3

Each period, plan (srd031) derives the next objectives from the agreed epics: the items no met objective has drawn down, in the owner's order and the order their dependencies and anchors allow, the changes to the epics since the last release tag, and the program's velocity.

<!-- req: R1.4 -->
### R1.4

Velocity is the points of epic items drawn down per period, read from the ledger. Until a program has a period of it, plan asks the person for an assumed velocity (D5).

<!-- req: R1.5 -->
### R1.5

From the epics and the ledger, tpm renders a roadmap forecast, the remaining items spread over future periods at velocity in dependency order, and a burn-up per epic, points drawn down against total points so that scope growth shows. Both are committed beside the dashboard (srd018), regenerated each run, and never edited.

<!-- req: R1.6 -->
### R1.6

The corpus audit gates epic documents as it gates objective documents: every depends_on and every objective citation of an epic item resolves (D9), and the depends_on graph over all epics has no cycle.

<!-- req: R1.7 -->
### R1.7

Until a release plans this SRD, tpm and the corpus audit ignore docs/specs/epics/, plan chooses objective items as srd031 states, and the objective's existing due item (srd034 R1.4) is its only date.

<!-- field: non_goals -->
## Non-goals

- A baseline or a commitment date. The roadmap is a forecast regenerated from current knowledge each run, and nothing gates on it.
- Duration estimates. Items are estimated in relative points, and dates come only from velocity.
- Estimating, ranking, or assigning a person (D5). Points and velocity belong to the work and the program.
- A second unit of work (D4). An epic is a document; the tracking issue (srd001 R6) is the tracker's grouping of issues.

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 -->
### AC1

The release that plans this SRD writes its requirement items from R1.1 to R1.6; until then the interim rule of R1.7 holds.

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
- [R1.2](#r12)
- [R1.3](#r13)
- [R1.4](#r14)
- [R1.5](#r15)
- [R1.6](#r16)
- [R1.7](#r17)
