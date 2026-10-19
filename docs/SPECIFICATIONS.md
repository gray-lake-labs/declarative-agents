---
id: ai-native-scrum-specifications
title: AI-Native Scrum Specification Index
---

<!-- field: overview -->
## Overview

Specification index for the AI-native scrum process. The requirements are
complete: 23 SRDs, six config formats, and four semantic models. srd001 to
srd008 define the control plane, the objective ledger, pinned references,
and done as evidence in a named environment; srd010 to srd019 define one
script each as a tpm subcommand group and contain no question to a person,
srd019 being the track-goals audit; srd020 to srd033 define the five
outer-loop skills, each stating what it reads, what it asks, and which
issues it pushes or events it posts. srd034 makes an objective a document
in the system repository before it becomes an issue and files the dated
objective issues; the weekly status issue and the release summary are
srd018's, the sprint-end issue that pushes the release is srd038's, and
the reference-update issue on a downstream release is srd014's. srd035 and srd036 specify the
inner loop on GitLab: issue-push files one admitted issue whose plane says
whether it is project work or a requirements change, and issue-pop takes
it into its own branch and worktree until its merge request closes it.
srd037 is the Agent Plugins 1.0 package of the skills and tpm install,
which writes the skills embedded in tpm for Cursor, Claude Code, or
GitHub Copilot. srd039 is the shared reader of corpus and config
documents (internal/corpus), srd040 the format tools and format gates in
magefiles, and srd041 the states and transitions of an issue. srd043 is
tpm init and tpm onboard, which lay down a program's corpus, and srd044
tpm help, which teaches the process from the binary. srd039 and
srd040 specify shared code and tooling no use case cites, so the road map
lists them under the release that built them. An SRD whose release is
not yet planned holds its direction only (design.md D12): its items are
retired, and an interim rule says what tpm does until they are rewritten.
make-work, do-work, and release-push are still referenced as the existing
skills. Every repository of a consuming program carries the same
corpus shape (design.md D7). Release 00 (00.0 to 00.3) and releases 01.0 to 01.2 are implemented.

Releases are planned by use case, and a use case cites only the
requirement items its release delivers. Release 00 is the inner loop
on GitLab in four dot releases: 00.0 labels and push, 00.1 pop and merge,
00.2 changes under docs/ through the audit gate, 00.3 tpm install. Release 01 builds the
interaction between the technical program manager and the Product
Owners in three dot releases: 01.0 objectives reach the Product Owner, 01.1 status comes back
weekly, 01.2 the sprint end releases and plans the next period. 01.4, the
agent-legible CLI, ships before 01.3: tpm help, the issue group, and tpm
init and onboard. 02.0 holds
evidence, handoff, and cadence, without use cases yet. Every SRD is listed
under the first release whose use case touches it. See
docs/road-map.md.

<!-- field: roadmap_summary | form: "typed table" -->
## Roadmap summary

| version | name | use_cases_done | use_cases_total | status |
| --- | --- | --- | --- | --- |
| "00.0" | Labels and push | 1 | 1 | implemented |
| "00.1" | Pop and merge | 1 | 1 | implemented |
| "00.2" | Changes under docs/ | 2 | 2 | implemented |
| "00.3" | Install the skills | 1 | 1 | implemented |
| "01.0" | Objectives reach the Product Owner | 2 | 2 | implemented |
| "01.1" | Status comes back weekly | 2 | 2 | implemented |
| "01.2" | Sprint end closes the loop | 2 | 2 | implemented |
| "01.3" | Tags trigger handoff and the summary | 0 | 2 | specified |
| "01.4" | The agent-legible CLI | 0 | 2 | specified |
| "02.0" | Evidence, handoff, and cadence | 0 | 0 | specified |
| "03.0" | The tracker becomes beads | 2 | 2 | implemented |

<!-- field: foundation_document_index -->
## Foundation document index

| path | id |
| --- | --- |
| docs/VISION.md | [ai-native-scrum](VISION.md) |
| docs/ARCHITECTURE.md | [architecture-ai-native-scrum](ARCHITECTURE.md) |
| docs/constitutions/design.md | [design](constitutions/design.md) |
| docs/road-map.md | [ai-native-scrum-roadmap](road-map.md) |
| docs/engineering/eng01-regression-rig.md | [eng01-regression-rig](engineering/eng01-regression-rig.md) |
| docs/engineering/eng02-tpm-and-skills.md | [eng02-tpm-and-skills](engineering/eng02-tpm-and-skills.md) |
| docs/engineering/eng03-releasing-scrum-master.md | [eng03-releasing-scrum-master](engineering/eng03-releasing-scrum-master.md) |
| docs/engineering/eng04-beads-sync.md | [eng04-beads-sync](engineering/eng04-beads-sync.md) |
| docs/guides/ai-native-sdlc.md | ai-native-sdlc |
| docs/guides/corpus-audit.md | corpus-audit |
| docs/guides/parent-upgrade.md | parent-upgrade |
| docs/guides/process.md | process |
| docs/guides/sdlc-playbook.md | sdlc-playbook |
| docs/guides/spec-kit.md | spec-kit |

<!-- field: srd_index -->
## SRD index

| id | path | title | release |
| --- | --- | --- | --- |
| [srd001-labels](specs/software-requirements/srd001-labels.md) | docs/specs/software-requirements/srd001-labels.md | The label scheme: plane, type, kind, component, and state | 00.0 |
| [srd006-events](specs/software-requirements/srd006-events.md) | docs/specs/software-requirements/srd006-events.md | Comment events | 01.0 |
| [srd007-links](specs/software-requirements/srd007-links.md) | docs/specs/software-requirements/srd007-links.md | Dependency links | 01.0 |
| [srd008-gates](specs/software-requirements/srd008-gates.md) | docs/specs/software-requirements/srd008-gates.md | Definition of done: evidence in a named environment | 02.0 |
| [srd010-standup](specs/software-requirements/srd010-standup.md) | docs/specs/software-requirements/srd010-standup.md | Script: standup (tpm standup ...) | 01.1 |
| [srd013-program](specs/software-requirements/srd013-program.md) | docs/specs/software-requirements/srd013-program.md | Script: issue (tpm issue ...) | 00.0 |
| [srd014-handoff](specs/software-requirements/srd014-handoff.md) | docs/specs/software-requirements/srd014-handoff.md | Script: handoff (tpm handoff ...) | 01.3 |
| [srd015-release](specs/software-requirements/srd015-release.md) | docs/specs/software-requirements/srd015-release.md | Script: release (tpm release ...) | 01.2 |
| [srd018-reports](specs/software-requirements/srd018-reports.md) | docs/specs/software-requirements/srd018-reports.md | Script: reports (tpm reports ...) | 01.1 |
| [srd019-track-goals](specs/software-requirements/srd019-track-goals.md) | docs/specs/software-requirements/srd019-track-goals.md | Script: track-goals (tpm goals ...) | 02.0 |
| [srd020-daily-scrum](specs/software-requirements/srd020-daily-scrum.md) | docs/specs/software-requirements/srd020-daily-scrum.md | Skill: daily-scrum | 02.0 |
| [srd022-create-goals](specs/software-requirements/srd022-create-goals.md) | docs/specs/software-requirements/srd022-create-goals.md | Skill: create-goals | 01.0 |
| [srd031-plan-release](specs/software-requirements/srd031-plan-release.md) | docs/specs/software-requirements/srd031-plan-release.md | Skill: plan-release | 01.2 |
| [srd032-sprint-review](specs/software-requirements/srd032-sprint-review.md) | docs/specs/software-requirements/srd032-sprint-review.md | Skill: sprint-review | 02.0 |
| [srd033-sprint-retrospective](specs/software-requirements/srd033-sprint-retrospective.md) | docs/specs/software-requirements/srd033-sprint-retrospective.md | Skill: sprint-retrospective | 02.0 |
| [srd034-objectives](specs/software-requirements/srd034-objectives.md) | docs/specs/software-requirements/srd034-objectives.md | Objective documents and the dated objective loop | 01.0 |
| [srd035-issue-push](specs/software-requirements/srd035-issue-push.md) | docs/specs/software-requirements/srd035-issue-push.md | Skill: issue-push | 00.0 |
| [srd036-issue-pop](specs/software-requirements/srd036-issue-pop.md) | docs/specs/software-requirements/srd036-issue-pop.md | Skill: issue-pop | 00.1 |
| [srd037-install](specs/software-requirements/srd037-install.md) | docs/specs/software-requirements/srd037-install.md | Script: install (tpm install) | 00.3 |
| [srd038-sprint-end](specs/software-requirements/srd038-sprint-end.md) | docs/specs/software-requirements/srd038-sprint-end.md | Skill: sprint-end | 01.2 |
| [srd039-corpus](specs/software-requirements/srd039-corpus.md) | docs/specs/software-requirements/srd039-corpus.md | Shared: corpus and config documents (internal/corpus) | 01.0 |
| [srd040-format-tools](specs/software-requirements/srd040-format-tools.md) | docs/specs/software-requirements/srd040-format-tools.md | Tooling: mage convert, fmt, specCheck, and the format gates | 01.2 |
| [srd041-states](specs/software-requirements/srd041-states.md) | docs/specs/software-requirements/srd041-states.md | States and transitions of an issue | 00.1 |
| [srd042-beads-tracker](specs/software-requirements/srd042-beads-tracker.md) | docs/specs/software-requirements/srd042-beads-tracker.md | Beads tracker: the program's facts live in a repository | 03.0 |
| [srd043-init-onboard](specs/software-requirements/srd043-init-onboard.md) | docs/specs/software-requirements/srd043-init-onboard.md | Script: init and onboard (tpm init, tpm onboard) | 01.4 |
| [srd044-help](specs/software-requirements/srd044-help.md) | docs/specs/software-requirements/srd044-help.md | Script: help (tpm help) | 01.4 |

<!-- field: config_format_index -->
## Config format index

| id | path | title |
| --- | --- | --- |
| [report](specs/config-formats/report.md) | docs/specs/config-formats/report.md | Weekly status report |
| [reports-config](specs/config-formats/reports-config.md) | docs/specs/config-formats/reports-config.md | Reports repository configuration |
| [issue-body](specs/config-formats/issue-body.md) | docs/specs/config-formats/issue-body.md | Issue body conventions |
| [labels](specs/config-formats/labels.md) | docs/specs/config-formats/labels.md | Label schema |
| [person-config](specs/config-formats/person-config.md) | docs/specs/config-formats/person-config.md | Person configuration |
| [policy](specs/config-formats/policy.md) | docs/specs/config-formats/policy.md | Process policy |

<!-- field: semantic_model_index -->
## Semantic model index

| id | path | title |
| --- | --- | --- |
| [agents](specs/semantic-models/agents.md) | docs/specs/semantic-models/agents.md | Agents and the loops they run |
| [objects](specs/semantic-models/objects.md) | docs/specs/semantic-models/objects.md | Objects of the process |
| [planes](specs/semantic-models/planes.md) | docs/specs/semantic-models/planes.md | The three planes |
| [roles](specs/semantic-models/roles.md) | docs/specs/semantic-models/roles.md | Roles and levels |

<!-- field: use_case_index -->
## Use case index

| id | title | release | status | test_suite | path |
| --- | --- | --- | --- | --- | --- |
| [rel00.0-uc001-push-project-work](specs/use-cases/rel00.0-uc001-push-project-work.md) | A Developer Pushes a Request and It Becomes an Admitted Issue | 00.0 | implemented | [test-rel00.0-push](specs/test-suites/test-rel00.0-push.md) | docs/specs/use-cases/rel00.0-uc001-push-project-work.md |
| [rel00.1-uc001-pop-and-merge](specs/use-cases/rel00.1-uc001-pop-and-merge.md) | A Developer Pops an Issue into a Worktree and Its Merge Request Closes It | 00.1 | implemented | [test-rel00.1-pop](specs/test-suites/test-rel00.1-pop.md) | docs/specs/use-cases/rel00.1-uc001-pop-and-merge.md |
| [rel00.2-uc001-spec-change](specs/use-cases/rel00.2-uc001-spec-change.md) | A Developer Changes a Requirement During Design Through the Audit Gate | 00.2 | implemented | [test-rel00.2-docs-change](specs/test-suites/test-rel00.2-docs-change.md) | docs/specs/use-cases/rel00.2-uc001-spec-change.md |
| [rel00.2-uc002-planning-change](specs/use-cases/rel00.2-uc002-planning-change.md) | A Lead Changes the Plan Through a Management Issue and the Same Gate | 00.2 | implemented | [test-rel00.2-docs-change](specs/test-suites/test-rel00.2-docs-change.md) | docs/specs/use-cases/rel00.2-uc002-planning-change.md |
| [rel00.3-uc001-install-skills](specs/use-cases/rel00.3-uc001-install-skills.md) | A Developer Installs the Skills for Their Agent | 00.3 | implemented | [test-rel00.3-install](specs/test-suites/test-rel00.3-install.md) | docs/specs/use-cases/rel00.3-uc001-install-skills.md |
| [rel01.0-uc001-author-objectives](specs/use-cases/rel01.0-uc001-author-objectives.md) | The Technical Program Manager Writes an Audited Objective Document | 01.0 | implemented | [test-rel01.0-objectives](specs/test-suites/test-rel01.0-objectives.md) | docs/specs/use-cases/rel01.0-uc001-author-objectives.md |
| [rel01.0-uc002-push-objective-issues](specs/use-cases/rel01.0-uc002-push-objective-issues.md) | Create-Goals Files Dated Objective Issues for the Product Owners | 01.0 | implemented | [test-rel01.0-objectives](specs/test-suites/test-rel01.0-objectives.md) | docs/specs/use-cases/rel01.0-uc002-push-objective-issues.md |
| [rel01.1-uc001-weekly-status](specs/use-cases/rel01.1-uc001-weekly-status.md) | A Product Owner Reports Weekly Status Through a Dated Issue | 01.1 | implemented | [test-rel01.1-weekly-status](specs/test-suites/test-rel01.1-weekly-status.md) | docs/specs/use-cases/rel01.1-uc001-weekly-status.md |
| [rel01.1-uc002-period-rollup](specs/use-cases/rel01.1-uc002-period-rollup.md) | The Technical Program Manager Aggregates a Period of Status | 01.1 | implemented | [test-rel01.1-weekly-status](specs/test-suites/test-rel01.1-weekly-status.md) | docs/specs/use-cases/rel01.1-uc002-period-rollup.md |
| [rel01.2-uc001-sprint-end-release](specs/use-cases/rel01.2-uc001-sprint-end-release.md) | A Sprint-End Issue Summarizes the Period, Asks, and Releases | 01.2 | implemented | [test-rel01.2-sprint-end](specs/test-suites/test-rel01.2-sprint-end.md) | docs/specs/use-cases/rel01.2-uc001-sprint-end-release.md |
| [rel01.2-uc002-plan-next-period](specs/use-cases/rel01.2-uc002-plan-next-period.md) | The Technical Program Manager Plans the Next Period's Objectives | 01.2 | implemented | [test-rel01.2-sprint-end](specs/test-suites/test-rel01.2-sprint-end.md) | docs/specs/use-cases/rel01.2-uc002-plan-next-period.md |
| [rel01.3-uc001-reference-update](specs/use-cases/rel01.3-uc001-reference-update.md) | A Downstream Release Files a Reference-Update Issue Upstream | 01.3 | specified | [test-rel01.3-tag-triggered](specs/test-suites/test-rel01.3-tag-triggered.md) | docs/specs/use-cases/rel01.3-uc001-reference-update.md |
| [rel01.3-uc002-release-summary](specs/use-cases/rel01.3-uc002-release-summary.md) | A System Release Drafts the Summary for Review | 01.3 | specified | [test-rel01.3-tag-triggered](specs/test-suites/test-rel01.3-tag-triggered.md) | docs/specs/use-cases/rel01.3-uc002-release-summary.md |
| [rel01.4-uc001-bootstrap-a-program](specs/use-cases/rel01.4-uc001-bootstrap-a-program.md) | The Technical Program Manager Starts a Program and a Repository Joins It | 01.4 | specified | [test-rel01.4-agent-legible-cli](specs/test-suites/test-rel01.4-agent-legible-cli.md) | docs/specs/use-cases/rel01.4-uc001-bootstrap-a-program.md |
| [rel01.4-uc002-agent-learns-the-cli](specs/use-cases/rel01.4-uc002-agent-learns-the-cli.md) | An Agent Learns the Process from tpm and Works an Issue | 01.4 | specified | [test-rel01.4-agent-legible-cli](specs/test-suites/test-rel01.4-agent-legible-cli.md) | docs/specs/use-cases/rel01.4-uc002-agent-learns-the-cli.md |
| [rel03.0-uc001-pop-and-close-on-beads](specs/use-cases/rel03.0-uc001-pop-and-close-on-beads.md) | An Issue Is Popped, Delivered, and Closed Against Beads | 03.0 | implemented | [test-rel03.0-beads-tracker](specs/test-suites/test-rel03.0-beads-tracker.md) | docs/specs/use-cases/rel03.0-uc001-pop-and-close-on-beads.md |
| [rel03.0-uc002-two-clones-synchronize](specs/use-cases/rel03.0-uc002-two-clones-synchronize.md) | Two Clones Synchronize Through the Tracker Repository | 03.0 | implemented | [test-rel03.0-beads-tracker](specs/test-suites/test-rel03.0-beads-tracker.md) | docs/specs/use-cases/rel03.0-uc002-two-clones-synchronize.md |

<!-- field: test_suite_index | form: "typed table" -->
## Test suite index

| id | title | release | test_case_count | traces | path |
| --- | --- | --- | --- | --- | --- |
| [test-rel00.0-push](specs/test-suites/test-rel00.0-push.md) | Test Suite for Release 00.0 Labels and Push | "00.0" | 2 | [[rel00.0-uc001-push-project-work](specs/use-cases/rel00.0-uc001-push-project-work.md)] | docs/specs/test-suites/test-rel00.0-push.md |
| [test-rel00.1-pop](specs/test-suites/test-rel00.1-pop.md) | Test Suite for Release 00.1 Pop and Merge | "00.1" | 3 | [[rel00.1-uc001-pop-and-merge](specs/use-cases/rel00.1-uc001-pop-and-merge.md)] | docs/specs/test-suites/test-rel00.1-pop.md |
| [test-rel00.2-docs-change](specs/test-suites/test-rel00.2-docs-change.md) | Test Suite for Release 00.2 Changes Under docs/ | "00.2" | 3 | [[rel00.2-uc001-spec-change](specs/use-cases/rel00.2-uc001-spec-change.md), [rel00.2-uc002-planning-change](specs/use-cases/rel00.2-uc002-planning-change.md)] | docs/specs/test-suites/test-rel00.2-docs-change.md |
| [test-rel00.3-install](specs/test-suites/test-rel00.3-install.md) | Test Suite for Release 00.3 Install the Skills | "00.3" | 6 | [[rel00.3-uc001-install-skills](specs/use-cases/rel00.3-uc001-install-skills.md)] | docs/specs/test-suites/test-rel00.3-install.md |
| [test-rel01.0-objectives](specs/test-suites/test-rel01.0-objectives.md) | Test Suite for Release 01.0 Objectives Reach the Product Owner | "01.0" | 6 | [[rel01.0-uc001-author-objectives](specs/use-cases/rel01.0-uc001-author-objectives.md), [rel01.0-uc002-push-objective-issues](specs/use-cases/rel01.0-uc002-push-objective-issues.md)] | docs/specs/test-suites/test-rel01.0-objectives.md |
| [test-rel01.1-weekly-status](specs/test-suites/test-rel01.1-weekly-status.md) | Test Suite for Release 01.1 Status Comes Back Weekly | "01.1" | 6 | [[rel01.1-uc001-weekly-status](specs/use-cases/rel01.1-uc001-weekly-status.md), [rel01.1-uc002-period-rollup](specs/use-cases/rel01.1-uc002-period-rollup.md)] | docs/specs/test-suites/test-rel01.1-weekly-status.md |
| [test-rel01.2-sprint-end](specs/test-suites/test-rel01.2-sprint-end.md) | Test Suite for Release 01.2 Sprint End Closes the Loop | "01.2" | 5 | [[rel01.2-uc001-sprint-end-release](specs/use-cases/rel01.2-uc001-sprint-end-release.md), [rel01.2-uc002-plan-next-period](specs/use-cases/rel01.2-uc002-plan-next-period.md)] | docs/specs/test-suites/test-rel01.2-sprint-end.md |
| [test-rel01.3-tag-triggered](specs/test-suites/test-rel01.3-tag-triggered.md) | Test Suite for Release 01.3 Tags Trigger Handoff and the Summary | "01.3" | 2 | [[rel01.3-uc001-reference-update](specs/use-cases/rel01.3-uc001-reference-update.md), [rel01.3-uc002-release-summary](specs/use-cases/rel01.3-uc002-release-summary.md)] | docs/specs/test-suites/test-rel01.3-tag-triggered.md |
| [test-rel01.4-agent-legible-cli](specs/test-suites/test-rel01.4-agent-legible-cli.md) | Test Suite for Release 01.4 The Agent-Legible CLI | "01.4" | 8 | [[rel01.4-uc001-bootstrap-a-program](specs/use-cases/rel01.4-uc001-bootstrap-a-program.md), [rel01.4-uc002-agent-learns-the-cli](specs/use-cases/rel01.4-uc002-agent-learns-the-cli.md)] | docs/specs/test-suites/test-rel01.4-agent-legible-cli.md |
| [test-rel03.0-beads-tracker](specs/test-suites/test-rel03.0-beads-tracker.md) | Test Suite for Release 03.0 The Tracker Becomes Beads | "03.0" | 4 | [[rel03.0-uc001-pop-and-close-on-beads](specs/use-cases/rel03.0-uc001-pop-and-close-on-beads.md), [rel03.0-uc002-two-clones-synchronize](specs/use-cases/rel03.0-uc002-two-clones-synchronize.md)] | docs/specs/test-suites/test-rel03.0-beads-tracker.md |

<!-- field: coverage_gaps -->
## Coverage gaps

| id | gap |
| --- | --- |
| GAP1 | Use cases exist for releases 00 (00.0 to 00.3) and 01 (01.0 to 01.4) only. Release 02.0 has none: srd008, srd019, srd020, srd032, and srd033 hold its direction only (design.md D12), and their items and use cases are written when 02.0 is planned. |
| GAP2 | Test suites exist for releases 00 (00.0 to 00.3) and 01 (01.0 to 01.4) only. test-rel00.0-push, test-rel00.1-pop, test-rel00.2-docs-change, test-rel00.3-install, test-rel01.0-objectives, test-rel01.1-weekly-status, and test-rel01.2-sprint-end are implemented; test-rel01.3-tag-triggered and test-rel01.4-agent-legible-cli are specified, and acceptance criteria outside them have no test case yet. |
| GAP3 | The audit is the parent's specification-critic corpus-audit agent, run by mage audit after the repository gate tests. It does not yet run in CI on merge requests that change docs/ (srd034 R2.2); this repository has no pipeline. |
| GAP4 | Open decision. The promotion trigger for each environment (each subsystem release, nightly from the latest releases, or both) is declared per environment SRD; no default is chosen yet. |
| GAP5 | Open decision. The channel by which evidence from customer R&D environments returns to the objective ledger is not specified (srd008, srd019 non-goals). |
| GAP6 | Open decision. Release 01.4 renames the program group to issue, moves recur into it from standup, and adds init, onboard, and help (srd013, srd043, srd044). The other subcommand groups (standup, goals, handoff, release, reports, plan, sprint-end) are rationalized when 02.0 rewrites track-goals (srd019). |
| GAP7 | The corpus's own SRDs omit test and environment on acceptance criteria (design.md srd acceptance_criteria_note) until the test subsystem's environment SRDs exist. |
| GAP8 | The parent corpus audit does not read docs/specs/objectives/, so a magefiles gate test validates objective documents against srd034 R1 inside mage audit (srd034 R2.1). An objective document is scrum-master's type written as an SRD in the parent's format, so the gate is scrum-master's own, not a gap the parent closes. |
| GAP9 | tpm issue lint does not yet run in CI on issue creation or on a schedule (srd013 R1.3); that needs a released tpm binary the program's pipelines can run. |
| GAP11 | Open decision. srd041 has lint rows only out of open and needs-triage, so an issue already past admission that later fails a re-lint has no transition. On GitLab and on beads lint sets or clears the needs-triage label alone and leaves the state; whether such an issue should leave the backlog is undecided. |
