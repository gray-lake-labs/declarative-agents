---
id: roles
title: Roles and levels
---

<!-- field: purpose -->
## Purpose

Defines the four roles of the process by the level each owns, what each writes, and when its level is done. Every repository carries the same corpus (design.md D7); roles differ only by which SRDs they own. Skill SRDs name these roles as their actors.

<!-- field: ownership_rules -->
## Ownership rules

| level | owner | writes | done_when |
| --- | --- | --- | --- |
| system | technical_program_manager | System SRDs and interface YAMLs in the system repository, and the subsystem SRDs that declare each subsystem's owner and components. | System tests pass on the pinned set in the final environment before customer handoff (srd008). |
| subsystem | product_owner | Requirements that satisfy the objectives pushed to the subsystem, and the subsystem's road-map. | Its requirements' tests pass in the environment each objective names (srd008). |
| component | developer | Issues with make-work, then code and documents through push, pop, and do-work. | Component tests pass in the local environment and the branch is approved and merged (srd008). |
| test_subsystem | scrum_master | Environment SRDs, pipeline SRDs, and test-suite SRDs in the test-subsystem repository. | Known-good sets pass and known-bad sets fail in every environment that runs them (srd008). |

<!-- field: roles | form: "keyed table" -->
## Roles

| id | was | is | runs |
| --- | --- | --- | --- |
| technical_program_manager | Manages cross-team dependencies in spreadsheets. | "Owns the system level: the system requirements and interfaces, the subsystem SRDs that say who owns what, the label schema, and the policy. Declares each system release before its three-week period and pushes objectives to the subsystem owners and the test subsystem. Owns clarity and process, not the delivery outcomes of the subsystems." | [plan, tasks] |
| product_owner | Writes human-facing stories and manages the backlog. | Owns a subsystem. Writes the requirements that satisfy the objectives pushed to it, with acceptance criteria that name a test and an environment, using do-work on its docs; places them in the subsystem's road-map releases; pushes objectives to downstream component repositories when it has them; and judges the Increment. | [plan, tasks, do-work, sprint-review] |
| developer | Writes code. | Owns components. Turns the plan into issues with make-work, works them through the inner loop, and reviews branches for architecture, security, and anything the tests cannot see. In a program of developers and researchers, also runs experiments, writes reports, and reviews others' artifacts; the kind label says which. | [make-work, daily-scrum, specify, implement, do-work, tpm release] |
| scrum_master | Runs standups, removes blockers, counts points. | "Owns the test subsystem: the environments of the chain, the pipelines that promote pinned sets between them, and the suites that produce evidence, each specified as an SRD. Owns the audits, track-goals among them, and facilitates the Retrospective. There is no scrum master agent." | [sprint-retrospective, track-goals, audits] |

<!-- field: relationships -->
## Relationships

Each role's relationship to the Scrum Guide (2020); docs/guides/agile-mapping.md maps every event, artifact, and commitment.

| role | scrum_guide | fit | rationale |
| --- | --- | --- | --- |
| technical_program_manager | none; the Guide has no role above one team | addition | A program of several teams needs one owner of the system's requirements, interfaces, and releases, who pushes each team the objectives that become its Sprint Goals; the nearest counterparts are Nexus's integration work and SAFe's Release Train Engineer with Solution Management. It orders the program's backlog and never assigns work. |
| product_owner | Product Owner | aligned, with an addition | Orders the subsystem's backlog, owns its goals, and judges the Increment; also writes requirements with an agent, because acceptance criteria that name a test and an environment are what make done checkable. |
| developer | Developers | aligned | Create the Increment, own the Sprint Backlog through make-work, hold to the Definition of Done, and adapt daily. |
| scrum_master | Scrum Master | deviation | The administrative half of the role (standups, the board, counting points, chasing blockers) is scripts; what remains is the part of the team's effectiveness that is a system to own: the environments, suites, and audits that decide done, and the Retrospective. Coaching stays a person's work. |

<!-- field: actor_boundaries | form: "tree" -->
## Actor boundaries

- interface: "What crosses between levels: a requirement, an interface, its released version, and the repositories that cite it by pinned URL. Owned by the level above."
- interior: "How a level meets the objectives pushed to it: its own requirements, issues, and implementation. Owned by that level's owner."
- rule: An owner specifies interfaces for the level below and never its interior; a level below changes an interface only through a plane::management or plane::control issue in the repository that owns it.
- push_and_pull: "Intent is pushed down as objectives and work is pulled. plan declares the next release at each level; tasks pushes one objective per downstream owner, citing the requirements it must satisfy at the release tag; the owner below writes requirements against it and Developers turn those into issues with make-work. Nobody assigns an issue: people pop issues for themselves. Objectives close on evidence, collected by track-goals."
