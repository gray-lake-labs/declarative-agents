# This Process and the Scrum Guide

This page is for a reader who knows the [Scrum Guide
(2020)](https://scrumguides.org/scrum-guide.html) and wants to check one
claim: this process is Scrum. Every accountability, event, artifact, and
commitment the Guide defines has a named counterpart here, or a deviation
stated with its reason. The tables below are the whole claim; the sections
after them explain the deviations. [process.md](process.md) shows the same
process in Scrum's three time horizons, and this page is its index.

Two facts frame every row. First, this process runs a program of several
Scrum teams, not one: as in Nexus, a program-level goal is broken into each
team's goals, and a level above the teams exists (the system, owned by the
technical program manager). Second, scripts and agents do the
administrative work around each event, so a person is asked only for the
judgments the tracker cannot hold (design.md D5). Neither changes what an
event is for; both change who does the bookkeeping.

## Accountabilities

Table 1: The Scrum Team's accountabilities.

| Scrum Guide | Here | Fit |
|---|---|---|
| Scrum Team: small, cross-functional, self-managing | A level's owner and the people who pull its issues; nobody assigns work, people pop issues for themselves (srd041 R9.1) | Aligned |
| Developers: create the Increment, own the Sprint Backlog, hold to the Definition of Done, adapt daily | Component owners: turn the planned release into issues with make-work, pull and build them, review every merge, release (roles.md developer) | Aligned |
| Product Owner: maximizes value, owns the Product Backlog and the Product Goal, orders the backlog | Subsystem owner: writes the requirements that satisfy the objectives pushed to the subsystem, orders the subsystem's road-map and epics, judges the Increment (roles.md product_owner) | Aligned, with one addition: the Product Owner writes requirements with an agent, not only stories |
| Scrum Master: accountable for the team's effectiveness, the events, and removing impediments | Owns the test subsystem: environments, suites, evidence, and audits; facilitates the Retrospective (roles.md scrum_master) | Deviation, explained below |
| (none: the Guide has no role above one team) | Technical program manager: owns the system level, its road-map and objectives, the label schema, and the policy (roles.md technical_program_manager) | Addition, explained below |

## Events

Table 2: The Scrum events.

| Scrum Guide | Here | Fit |
|---|---|---|
| The Sprint: fixed length, one month or less | A three-week period ending in a road-map release: a group milestone and one type::sprint issue holding the exit checklist (srd001 R7) | Aligned |
| Sprint Planning: why, what, and how for the Sprint | plan (srd031) chooses what the release proves, tasks (srd022) pushes one objective per downstream owner, and make-work turns the plan into issues | Aligned |
| Daily Scrum: Developers inspect progress toward the Sprint Goal and adapt the Sprint Backlog | daily-scrum (srd020): the facts are shown, and only blockers and interrupts are asked; until it is built, tpm standup (srd010) | Aligned, with status recitation removed |
| Sprint Review: inspect the Increment with stakeholders and adapt the Product Backlog | sprint-review (srd032): each objective's ledger and evidence shown, the Product Owner's verdict recorded, follow-up items pushed; until it is built, the sprint-end issue (srd038) | Aligned |
| Sprint Retrospective: plan ways to increase quality and effectiveness | sprint-retrospective (srd033): the period's metrics read, at most three plane::control issues that change the policy; until it is built, a policy change is a plane::control issue | Aligned |
| Backlog refinement: an ongoing activity, not an event | A change to an epic document or a requirement, merged through the audit gate; each period plan derives the next objectives from the epics' remaining items (srd045 R1.1, R1.3; srd031 R6.2) | Aligned |

## Artifacts and commitments

Table 3: Each artifact and the commitment that belongs to it.

| Scrum Guide | Here | Fit |
|---|---|---|
| Product Backlog | The epic documents (design.md epic, srd045) and the road-map's releases | Aligned |
| Product Backlog order | Road-map release order, then each epic's item order, set by the owner of the level: the Product Owner for a subsystem, the technical program manager for the program; plan's ranking (srd031 R3.1) proposes, the owner decides | Aligned |
| Product Goal | An objective document: what the program commits to for a horizon of periods (srd034 R1.3) | Aligned |
| Sprint Backlog | The issues carrying the period's milestone | Aligned |
| Sprint Goal | The objective issue each owner receives for the period, citing the requirements it must satisfy and the environment where they count (srd034 R3.2) | Aligned; one per team, as in Nexus |
| Increment | A release tag, and the pinned set of releases promoted together (srd008, srd015) | Aligned |
| Definition of Done | The release gate in control/policy.md (definition_of_done) and the reviewed merge today; tests passing in a named environment, recorded as evidence, once track-goals runs (srd008, srd019) | Aligned, and stricter |

## Complementary practices

The Scrum Guide leaves estimation, epics, and forecasting to the team. The
process uses the common practices, and keeps each one's usual meaning.

Table 4: Practices outside the Guide.

| Practice | Here |
|---|---|
| Epics | Epic documents in the system repository, each a Product Backlog for multi-sprint work (srd045). The tracker's grouping of issues is a tracking issue, type::tracking (srd001 R6) |
| Story points | On epic backlog items, relative and never of a person (design.md epic) |
| Velocity | Points of epic items drawn down per period, read from the ledger (srd045 R1.4) |
| Burn-up and release forecast | Regenerated from the epics and the ledger beside the dashboard, never a baseline (srd045 R1.5, srd018 R11) |
| User stories | Requirements with acceptance criteria that name a test and an environment; use cases carry the priority (design.md use_case) |

## Deviations and their reasons

### The Scrum Master owns the test subsystem

The Guide's Scrum Master is accountable for the team's effectiveness:
coaching, making the events happen, and removing impediments. Most of a
Scrum Master's day in practice is not that; it is administration: running
standups, updating the board, counting points, chasing blockers. Here that
administration is scripts. The events run as skills, the board is the
tracker, and blockers are a label and a Daily Scrum question that end in
issues (srd020).

What is left is the part of effectiveness that is a system to own rather
than a meeting to run: the environments, suites, and audits that tell the
team whether its work is done. The Scrum Master owns that, the evidence the
Review inspects, and the Retrospective, where the team changes its rules.
Coaching stays a person's work; there is no Scrum Master agent.

### The technical program manager

The Guide describes one team and has no role above it. A program of
several teams needs one: someone owns the system's requirements and
interfaces, declares each system release, and pushes each team the
objectives that become its Sprint Goals. Nexus calls the equivalent
integration work the Nexus Integration Team; SAFe splits it between the
Release Train Engineer and Solution Management. Here it is one role, and
it owns clarity and process, not the teams' delivery: it never assigns
work, and it orders the program's backlog the way a Product Owner orders a
team's.

### The Product Owner writes requirements

The Guide lets the Product Owner delegate backlog work but keeps them
accountable. Here the Product Owner writes requirements with an agent
rather than handing stories to Developers, because a requirement whose
acceptance criteria name a test and an environment is what makes done
checkable. The Product Owner still orders the backlog and judges the
Increment.

## The pillars

Scrum rests on transparency, inspection, and adaptation. Transparency: the
tracker is the one source of truth, every object is reachable from an issue,
and the dashboard and forecasts are regenerated from it rather than kept by
hand. Inspection: an objective is met only by evidence recorded on its
ledger, and the Review inspects that ledger. Adaptation: the
Retrospective's output is a change to the versioned policy, and each
period's objectives are derived again from the backlog as it now stands.
