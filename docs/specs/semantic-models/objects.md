---
id: objects
title: Objects of the process
---

<!-- field: purpose -->
## Purpose

Defines every object the process uses by its relation to the issue, the only unit of work (design.md D4). Anything not on this list is not part of the process.

<!-- field: vocabulary -->
## Vocabulary

| term | definition | specified_in | relation |
| --- | --- | --- | --- |
| issue | A commitment to produce one thing. Carries a plane, a kind, a component, and a milestone. | srd001 |  |
| objective | An issue tasks pushes to a downstream owner, citing the upstream requirements it must satisfy by pinned URL and naming the environment where its evidence counts. Its comment events are its ledger; it closes when the ledger replays to met, or, in a program that does not yet run track-goals, by the merge of its repository's sprint-end issue (srd038 R2). | srd006 R8, srd006 R9, srd006 R10, srd006 R11, srd019, srd022, srd034 | an issue whose events record planned, evidence, and met |
| objective_document | A document in the system repository stating what the program commits to for a horizon of periods, in the shape of an SRD. Its items are what objective issues cite by pinned URL. | srd034 R1, design.md objective | cited by objective issues; not itself a unit of work |
| epic | A document in the system repository holding the Product Backlog for a body of work spanning several sprints: backlog items in the owner's order, each with story points, the items it depends on, and an optional anchor. | srd045, design.md epic | read by plan, which derives objectives from it; drawn down by objective items that cite its items; not itself a unit of work |
| subsystem | A part of the system with one owner (a Product Owner), declared by a subsystem SRD in the system repository that lists its components. | design.md subsystem_srd | owns the objectives pushed to it; groups components |
| component | A repository path declared by a subsystem SRD, owned by Developers. Its name is a component:: label value. | srd001 R21, design.md subsystem_srd | every issue carries exactly one component label |
| environment | One stage of the chain (local, dev, staging, production PoC, customer R&D), declared by an environment SRD in the test-subsystem repository with its owner, trigger, and suites. | srd008, design.md environment_srd | named by every acceptance criterion and every evidence event |
| pinned_set | One release of each subsystem and of the test subsystem, promoted together along the environment chain. | srd008 | the version an evidence event records |
| evidence | An evidence event on an objective: a named test's result in a named environment on a pinned set, for one requirement. The only input to done. | srd006 R9, srd008 | a comment event on an objective issue |
| task | A step inside one issue; a child work item. No labels of its own; never in the backlog. | srd001 R9 | belongs to exactly one issue |
| tracking_issue | An issue labelled type::tracking (type::epic is read as its alias) whose description lists member issues. Never worked directly. | srd001 R6 | an issue that contains references to issues |
| sprint | A three-week period ending in a road-map release; a group milestone and one type::sprint issue holding the exit checklist. | srd001 R7, srd031 | an issue plus the milestone every planned issue of the period carries |
| recurring_issue | A consumer's standing subscription to one artifact; closed and reopened per run. | srd001 R8 | an issue labelled type::recurring |
| audit | An issue with a Recurrence section, worked by the inner loop on a schedule or when a person triggers it; implement re-creates it after each run. Its findings are pushed as issues. | agents, srd018 R6 | an issue re-created per run |
| comment_event | A comment with a machine marker; information, never a commitment and never a state change. | srd006 | attached to exactly one issue |
| link | An Upstream line to another issue, a Requires line to a release, or a Cites line to a requirement or interface pinned to a tag. | srd007 | from one issue to an issue, a tag, or a pinned file anchor |
| release_tag | A version tag on the producing repository, created only after the artifact's gate passed. | srd008, srd015 | created by a kind::release issue; read by Requires lines and by handoff |
| merge_request | The only way an issue is closed as delivered; closes its issue and that issue's tasks. | srd041 R7, srd008 | closes exactly one issue |
| product_backlog | Not an object. The epic documents and the road-map's releases, ordered by the owner of the level: road-map release order first, then each epic's item order. The Product Owner orders a subsystem's, the technical program manager the program's; plan's ranking proposes and the owner decides. | srd045 R1.1, srd031 R3.1, srd031 R6.2, docs/guides/agile-mapping.md | documents that objective issues cite |
| backlog | Not an object. The query for open, unassigned, admitted, backlog-kind data issues (the ready state); the Product Backlog is product_backlog. | srd041 R3 | a view over issues |
| dashboard | Not an object. A rendering of the ledger and evidence committed to a repository; never edited. | srd019 | a view over issues |

<!-- field: taxonomy_rules -->
## Taxonomy rules

Flat by design: tracking issue, issue, task. Levels (system, subsystem, component) are repositories, not issue types; they connect through pinned Cites lines and objectives. Dependencies cross-cut the hierarchy through links; events attach to issues; sprints group issues by milestone.
