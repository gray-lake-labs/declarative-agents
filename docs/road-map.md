---
id: ai-native-scrum-roadmap
title: AI-Native Scrum Release Road Map
---

<!-- field: overview -->
## Overview

Release 00 is the inner loop on GitLab in four dot releases: 00.0 labels
and push, 00.1 pop and merge, 00.2 changes under docs/ through the audit
gate, and 00.3 tpm install for Cursor, Claude Code, and GitHub Copilot.
Every issue, project work, a design change, or a planning change, has one
path to a merged change. Release 01 builds the interaction between the technical program manager
and the Product Owners (srd034) in three dot releases: 01.0 Sprint Goals
reach the Product Owner as dated issues, 01.1 status comes back weekly,
and 01.2 the sprint end releases and plans the next period. 01.4, the
agent-legible CLI, ships before 01.3. Each use case cites only the
requirement items its release delivers, so the SRDs it touches are partly
delivered; the rest waits for 02.0. Release 04.0 aligns the planning
documents with the agile hierarchy (theme, initiative, epic, feature,
user story) and renames the objective issue the Sprint Goal. Release 04.1 makes a
program repository explain itself and adds tpm doctor to bring one an
older tpm laid down up to date. Release 04.3 ships tpm to machines
and to CI: release binaries, a signed container image, and the
declarative-agents runtime at the release tpm pins. Every release proves itself with mage
audit, which runs the parent's specification-critic corpus-audit agent
(docs/guides/corpus-audit.md), and with its test suite.

<!-- release: 00.0 | status: implemented -->
## 00.0 - Labels and push

tpm issue labels creates the program's labels, kind::spec among
them, and specify turns a request into one admitted, unassigned
issue whose plane and kind say what it is. The model writes the body;
tpm picks the labels, files, and lints.

<!-- field: srds -->
### SRDs

- srd001-labels
- srd013-program
- srd035-specify

<!-- field: use_cases -->
### Use cases

| id | summary | status | priority |
| --- | --- | --- | --- |
| [rel00.0-uc001-push-project-work](specs/use-cases/rel00.0-uc001-push-project-work.md) | A Developer's request becomes one admitted, unassigned data issue; the labels exist first. | implemented | P1 |

<!-- field: test_suites -->
### Test suites

| id | summary | status |
| --- | --- | --- |
| [test-rel00.0-push](specs/test-suites/test-rel00.0-push.md) | Labels created once; a request becomes an admitted data issue with no confirmation step. | implemented |

<!-- field: proof -->
### Proof

- mage audit
- mage regression:all
- mage integration:labels
- mage integration:push

<!-- release: 00.1 | status: implemented -->
## 00.1 - Pop and merge

implement claims an issue for the person who works it, on its own
branch in its own worktree, and the merge request tpm opens closes it.
Working a story's issues in one branch is planned for implement.

<!-- field: srds -->
### SRDs

- srd036-implement
- srd041-states

<!-- field: use_cases -->
### Use cases

| id | summary | status | priority |
| --- | --- | --- | --- |
| [rel00.1-uc001-pop-and-merge](specs/use-cases/rel00.1-uc001-pop-and-merge.md) | A Developer pops an admitted issue into a worktree and its merge request closes it. | implemented | P1 |

<!-- field: test_suites -->
### Test suites

| id | summary | status |
| --- | --- | --- |
| [test-rel00.1-pop](specs/test-suites/test-rel00.1-pop.md) | Pop assigns only the caller, opens the branch and worktree, refuses held or unadmitted issues, and the merge closes the issue. | implemented |

<!-- field: proof -->
### Proof

- mage audit
- mage regression:all
- mage integration:pop

<!-- release: 00.2 | status: implemented -->
## 00.2 - Changes under docs/

Any merge request that changes docs/ merges only through mage audit: a
kind::spec design change anyone files, and a plane::management planning
change a lead files. Sprint Goals are worked this way.

<!-- field: srds | form: "empty list" -->
### SRDs

None

<!-- field: use_cases -->
### Use cases

| id | summary | status | priority |
| --- | --- | --- | --- |
| [rel00.2-uc001-spec-change](specs/use-cases/rel00.2-uc001-spec-change.md) | A Developer changes a requirement during design through a kind::spec issue and the audit gate. | implemented | P1 |
| [rel00.2-uc002-planning-change](specs/use-cases/rel00.2-uc002-planning-change.md) | A lead changes a planning document through a management issue and the same gate. | implemented | P2 |

<!-- field: test_suites -->
### Test suites

| id | summary | status |
| --- | --- | --- |
| [test-rel00.2-docs-change](specs/test-suites/test-rel00.2-docs-change.md) | kind::spec is anyone's and in the backlog, planning is lead-only, and both merge only through mage audit. | implemented |

<!-- field: proof -->
### Proof

- mage audit
- mage regression:all
- mage integration:docsChange

<!-- release: 00.3 | status: implemented -->
## 00.3 - Install the skills

tpm install writes the skills of the Agent Plugins 1.0 package plugin/,
embedded in the tpm binary, where Cursor, Claude Code, or GitHub
Copilot reads them, stamped with the tpm
version, kept out of the repository by .git/info/exclude, and tpm install
--check reports drift locally.

<!-- field: srds -->
### SRDs

- srd037-install

<!-- field: use_cases -->
### Use cases

| id | summary | status | priority |
| --- | --- | --- | --- |
| [rel00.3-uc001-install-skills](specs/use-cases/rel00.3-uc001-install-skills.md) | A Developer installs the skills for their agent and --check guards them after a tpm upgrade. | implemented | P1 |

<!-- field: test_suites -->
### Test suites

| id | summary | status |
| --- | --- | --- |
| [test-rel00.3-install](specs/test-suites/test-rel00.3-install.md) | Paths and headers per agent, stamps, re-install, --force, stale skills, and --check, offline. | implemented |

<!-- field: proof -->
### Proof

- mage audit
- mage regression:all
- mage integration:install

<!-- release: 01.0 | status: implemented -->
## 01.0 - Sprint Goals reach the Product Owner

The technical program manager writes the program's Sprint Goals as an
initiative that mage audit checks through the
specification-critic agent, and tasks files one dated, pinned,
admitted Sprint Goal in each Product Owner's repository. The
release also builds what that path needs: the magefile that runs the
agent audit, tpm issue file and lint, the planned event, and the
regression rig: the Terraform split into infra/modules/program with
the production and disposable roots, and gitlabrig (eng01). Its
Sprint Goals are plane::management issues the receiving Product
Owner works through release 00.1's implement and 00.2's audit gate.

<!-- field: srds -->
### SRDs

- srd006-events
- srd007-links
- srd022-tasks
- srd034-initiatives
- srd039-corpus

<!-- field: semantic_models -->
### Semantic models

- planes
- roles
- objects
- agents

<!-- field: config_formats -->
### Config formats

- labels
- policy
- person-config
- issue-body

<!-- field: use_cases -->
### Use cases

| id | summary | status | priority |
| --- | --- | --- | --- |
| [rel01.0-uc001-author-objectives](specs/use-cases/rel01.0-uc001-author-objectives.md) | The technical program manager merges an initiative that the specification-critic agent audit passes. | implemented | P1 |
| [rel01.0-uc002-push-objective-issues](specs/use-cases/rel01.0-uc002-push-objective-issues.md) | tasks files one dated, pinned, admitted Sprint Goal per Product Owner. | implemented | P1 |

<!-- field: test_suites -->
### Test suites

| id | summary | status |
| --- | --- | --- |
| [test-rel01.0-objectives](specs/test-suites/test-rel01.0-objectives.md) | The agent audit passes a well-formed initiative and fails a malformed one; tasks files dated Sprint Goals once. | implemented |

<!-- field: proof -->
### Proof

- mage audit
- mage regression:all
- mage integration:objectivesAudit
- mage integration:tasks
- mage integration:program

<!-- release: 01.1 | status: implemented -->
## 01.1 - Status comes back weekly

A dated weekly status issue in each Product Owner's repository walks
the owner through every open Sprint Goal and yields a validated report;
the next week's issue follows with its new due date. The technical
program manager aggregates the reports by component and by Sprint Goal,
never by person.

<!-- field: srds -->
### SRDs

- srd010-standup
- srd018-reports

<!-- field: use_cases -->
### Use cases

| id | summary | status | priority |
| --- | --- | --- | --- |
| [rel01.1-uc001-weekly-status](specs/use-cases/rel01.1-uc001-weekly-status.md) | A Product Owner works a dated status issue into a validated per-Sprint-Goal report. | implemented | P1 |
| [rel01.1-uc002-period-rollup](specs/use-cases/rel01.1-uc002-period-rollup.md) | The technical program manager aggregates a week or period of reports into signals by component. | implemented | P2 |

<!-- field: test_suites -->
### Test suites

| id | summary | status |
| --- | --- | --- |
| [test-rel01.1-weekly-status](specs/test-suites/test-rel01.1-weekly-status.md) | Reports repository scaffold, the dated status issue, validation, recurrence, and aggregate signals. | implemented |

<!-- field: proof -->
### Proof

- mage audit
- mage regression:all
- mage integration:weeklyStatus
- mage integration:reports

<!-- release: 01.2 | status: implemented -->
## 01.2 - Sprint end closes the loop

A sprint-end issue summarizes the period, asks the owner what to
change, and releases behind the gate once its merge request merges;
Sprint Goals reported complete close by that merge without a met event.
plan records the next period's initiative items, publishes the
plan as a documents-only system tag, dates the milestone and the sprint
issue, and files the next period's sprint-end issues.

<!-- field: srds -->
### SRDs

- srd001-labels
- srd015-release
- srd018-reports
- srd031-plan
- srd034-initiatives
- srd038-sprint-end
- srd040-format-tools

<!-- field: use_cases -->
### Use cases

| id | summary | status | priority |
| --- | --- | --- | --- |
| [rel01.2-uc001-sprint-end-release](specs/use-cases/rel01.2-uc001-sprint-end-release.md) | A sprint-end issue summarizes the period, asks, and releases on confirmation. | implemented | P1 |
| [rel01.2-uc002-plan-next-period](specs/use-cases/rel01.2-uc002-plan-next-period.md) | The technical program manager plans the next period's initiative items with plan. | implemented | P1 |

<!-- field: test_suites -->
### Test suites

| id | summary | status |
| --- | --- | --- |
| [test-rel01.2-sprint-end](specs/test-suites/test-rel01.2-sprint-end.md) | Confirmed release after the sprint-end merge, reported-complete close, and next-period planning with its plan tag. | implemented |

<!-- field: proof -->
### Proof

- mage audit
- mage regression:all
- mage integration:release
- mage integration:sprintEnd
- mage integration:plan

<!-- release: 01.3 | status: specified -->
## 01.3 - Tags trigger handoff and the summary

A downstream release tag files one reference-update issue upstream, and
a system release tag opens the computed aggregate for the lead's review.
Moved from 01.2 on 2026-10-18 so the period loop ships first. Open before
planning: the deck renders from an approved summary (srd018 R4.1, R6.2)
but the tag pipeline renders it from the raw aggregate; the pipeline and
the lead each open a change for review for the stakeholder update; the
reports configuration has no slide limit; and aggregation runs over ISO
weeks, not a tag range.

<!-- field: srds -->
### SRDs

- srd007-links
- srd014-handoff
- srd018-reports

<!-- field: use_cases -->
### Use cases

| id | summary | status | priority |
| --- | --- | --- | --- |
| [rel01.3-uc001-reference-update](specs/use-cases/rel01.3-uc001-reference-update.md) | A downstream release tag files one reference-update issue upstream. | specified | P2 |
| [rel01.3-uc002-release-summary](specs/use-cases/rel01.3-uc002-release-summary.md) | A system release tag opens one summary for review with the computed aggregate and deck; the model drafts the prose during the lead's review. | specified | P2 |

<!-- field: test_suites -->
### Test suites

| id | summary | status |
| --- | --- | --- |
| [test-rel01.3-tag-triggered](specs/test-suites/test-rel01.3-tag-triggered.md) | Idempotent tag-triggered reference-update issues and summaries for review. | specified |

<!-- field: proof -->
### Proof

- mage audit
- mage regression:all
- mage integration:referenceUpdate
- mage integration:releaseSummary

<!-- release: 01.4 | status: implemented -->
## 01.4 - The agent-legible CLI

An agent with only tpm on PATH learns the process from it, and a program
starts from one command. tpm help explains the process, each group, and
each load-bearing concept; the program group becomes the issue group,
named for what every subcommand acts on, with recur moved in from
standup; tpm init lays down a system repository's corpus, control files,
and skills, and tpm onboard adds a repository to the program and files
its registration. Planned on 2026-10-18 to ship before 01.3, so it is the
next release (design.md D12): every later release's skills then call the
renamed commands, and the help is written once against them.

<!-- field: srds -->
### SRDs

- srd043-init-onboard
- srd044-help

<!-- field: use_cases -->
### Use cases

| id | summary | status | priority |
| --- | --- | --- | --- |
| [rel01.4-uc001-bootstrap-a-program](specs/use-cases/rel01.4-uc001-bootstrap-a-program.md) | tpm init starts a program the audit passes as written; tpm onboard adds a repository and files its registration. | implemented | P1 |
| [rel01.4-uc002-agent-learns-the-cli](specs/use-cases/rel01.4-uc002-agent-learns-the-cli.md) | An agent learns the process from tpm help and works an issue under tpm issue; the old group name names its fix. | implemented | P1 |

<!-- field: test_suites -->
### Test suites

| id | summary | status |
| --- | --- | --- |
| [test-rel01.4-agent-legible-cli](specs/test-suites/test-rel01.4-agent-legible-cli.md) | init and onboard over empty repositories and the audit, help pages and their drift gate, the issue group, and recur across the move. | implemented |

<!-- field: proof -->
### Proof

- mage audit
- mage regression:all
- mage integration:init
- mage integration:onboard
- mage integration:issueGroup

<!-- release: 02.0 | status: specified -->
## 02.0 - Evidence, handoff, and cadence

What the TPM-to-Product-Owner loop does not need yet: done as evidence
in a named environment and track-goals closing Sprint Goals on it, the
full handoff and release fan-out, the daily standup and Daily Scrum,
the sprint review and retrospective as skills, forecasts and
burn-ups rolled up from story evidence, and the requirement
items of earlier releases' SRDs that their use cases do not cite. Use
cases are written when the release is planned.

<!-- field: srds -->
### SRDs

- srd008-gates
- srd019-track-goals
- srd020-daily-scrum
- srd032-sprint-review
- srd033-sprint-retrospective
- srd041-states
- srd045-backlog

<!-- release: 03.0 | status: implemented -->
## 03.0 - The tracker becomes beads

The program's facts move from the hosted GitLab instance to beads in a
dedicated tracker repository, synchronized through git: pull before
read, push after write, conflicts resolved by rule (eng04-beads-sync).
tpm gains a beads backend behind the tracker boundary, the platform
Terraform and the GitLab client leave, and the regression rig provisions
fixture tracker repositories. srd042 holds the direction.

<!-- field: srds -->
### SRDs

- srd042-beads-tracker

<!-- field: use_cases -->
### Use cases

| id | summary | status | priority |
| --- | --- | --- | --- |
| [rel03.0-uc001-pop-and-close-on-beads](specs/use-cases/rel03.0-uc001-pop-and-close-on-beads.md) | An issue is popped, delivered by a local merge, and closed on beads with the merge commit cited. | implemented | P2 |
| [rel03.0-uc002-two-clones-synchronize](specs/use-cases/rel03.0-uc002-two-clones-synchronize.md) | Two clones pull before reading and push after writing; a contended pop keeps the earlier taker; an offline clone says so and reconciles. | implemented | P2 |

<!-- field: test_suites -->
### Test suites

| id | summary | status |
| --- | --- | --- |
| [test-rel03.0-beads-tracker](specs/test-suites/test-rel03.0-beads-tracker.md) | One issue's life, two clones, a contended pop, and an offline session against fixture tracker repositories. | implemented |

<!-- field: proof -->
### Proof

- mage audit
- mage integration:beadsPopClose
- mage integration:beadsStates
- mage integration:beadsSync
- mage integration:beadsOffline

<!-- release: 04.0 | status: implemented -->
## 04.0 - The agile hierarchy

A program's planning documents take the names, structure, and
relationships of the standard agile hierarchy. Theme, initiative, epic,
an optional feature, and user story are document types the corpus gate
checks, each with at most one parent, and the issue that executes a
story and its tasks stay in the tracker. Links between them are the
agile set: parent, blocks and blocked by, depends on, relates to,
duplicates, and implements. The initiative becomes the
initiative and the per-period Sprint Goal becomes the team's Sprint
Goal, which lists the stories its Sprint Planning selected and carries
the ledger, with evidence anchored to the story criterion it proves.
Story points and velocity live on stories. Released behaviour stays
readable: objective documents, issues, and markers are read as their
new names. Until track-goals runs (02.0), a Sprint Goal closes at the
sprint end as reported complete (srd038 R2). Its use cases, authoring
the hierarchy, planning a sprint from stories, and the Sprint Goal
ledger, cite the requirement items they touch.

<!-- field: srds -->
### SRDs

- srd001-labels
- srd006-events
- srd007-links
- srd013-program
- srd022-tasks
- srd031-plan
- srd034-initiatives
- srd038-sprint-end
- srd041-states
- srd045-backlog

<!-- field: semantic_models -->
### Semantic models

- objects
- roles
- planes
- agents

<!-- field: config_formats -->
### Config formats

- issue-body

<!-- field: use_cases -->
### Use cases

| id | summary | status | priority |
| --- | --- | --- | --- |
| [rel04.0-uc001-author-the-hierarchy](specs/use-cases/rel04.0-uc001-author-the-hierarchy.md) | The technical program manager and a Product Owner write a theme, initiative, epic, feature, and stories that the hierarchy gate checks. | implemented | P1 |
| [rel04.0-uc002-plan-a-sprint-from-stories](specs/use-cases/rel04.0-uc002-plan-a-sprint-from-stories.md) | tasks files the team's Sprint Goal; Sprint Planning chooses ranked stories to velocity and lists them on it. | implemented | P1 |
| [rel04.0-uc003-sprint-goal-ledger](specs/use-cases/rel04.0-uc003-sprint-goal-ledger.md) | The sprint-end merge closes a Sprint Goal with its stories reported, and objective issues read as Sprint Goals. | implemented | P1 |

<!-- field: test_suites -->
### Test suites

| id | summary | status |
| --- | --- | --- |
| [test-rel04.0-agile-hierarchy](specs/test-suites/test-rel04.0-agile-hierarchy.md) | The gate passes a hierarchy and fails each malformed document; Sprint Planning lists ranked stories on the Sprint Goal; the sprint end closes it with its stories reported; objective issues read as Sprint Goals. | implemented |

<!-- field: proof -->
### Proof

- mage audit
- mage regression:all
- mage integration:hierarchyGate
- mage integration:tasks
- mage integration:sprintPlanning
- mage integration:program
- mage integration:sprintEnd

<!-- release: 04.1 | status: implemented -->
## 04.1 - A repository that explains itself

A first-time reader of a program repository learns from the repository
what it is and how to read it. tpm init and tpm onboard write a root
README, a README in each hierarchy folder the level holds, and an
AGENTS.md naming the tpm commands and skills at that level and the
working rules, every figure Mermaid in the Markdown among them. tpm
doctor, run after tpm is upgraded, writes the files a newer tpm
scaffolds, refreshes tpm's text inside the orientation files' managed
blocks, reports stale skills, and repairs the documents to the pinned
parent format with the spec tool, now shared by mage and tpm in
internal/spectool. Planned and built on 2026-10-19 (design.md D12).

<!-- field: srds -->
### SRDs

- srd040-format-tools
- srd043-init-onboard

<!-- field: use_cases -->
### Use cases

| id | summary | status | priority |
| --- | --- | --- | --- |
| [rel04.1-uc001-orient-and-repair](specs/use-cases/rel04.1-uc001-orient-and-repair.md) | A first-time reader orients from the scaffolded READMEs and AGENTS.md; tpm doctor brings a repository an older tpm laid down up to date. | implemented | P1 |

<!-- field: test_suites -->
### Test suites

| id | summary | status |
| --- | --- | --- |
| [test-rel04.1-orientation-doctor](specs/test-suites/test-rel04.1-orientation-doctor.md) | init and onboard write the orientation files; doctor adds missing files, refreshes blocks, reports stale skills, and repairs documents at the pin. | implemented |

<!-- field: proof -->
### Proof

- mage audit
- mage regression:all
- mage integration:init
- mage integration:onboard
- mage integration:doctor

<!-- release: 04.2 | status: implemented -->
## 04.2 - A planning document drafted in conversation

An owner turns a description into one planning document at any level,
theme to story, by talking it through. The draft skill reads the
description and the repository's hierarchy, proposes a parent and an id,
asks the few questions the description leaves open with a recommended
answer each, then writes the document as to_be_agreed, checks it, revises
it with the owner, and offers to draft its children. tpm backlog list
gives the skill the hierarchy, and tpm backlog check runs the hierarchy
gate in any repository, which until now ran only inside this repository's
mage audit. Planned and built on 2026-10-20 (design.md D12).

<!-- field: srds -->
### SRDs

- srd045-backlog
- srd046-draft

<!-- field: use_cases -->
### Use cases

| id | summary | status | priority |
| --- | --- | --- | --- |
| [rel04.2-uc001-draft-a-planning-document](specs/use-cases/rel04.2-uc001-draft-a-planning-document.md) | An owner drafts a planning document from a description in conversation: list, discuss, write, check, revise, and offer the children. | implemented | P1 |

<!-- field: test_suites -->
### Test suites

| id | summary | status |
| --- | --- | --- |
| [test-rel04.2-draft](specs/test-suites/test-rel04.2-draft.md) | tpm backlog list names the hierarchy; tpm backlog check is the gate; the draft skill is installed within its limit. | implemented |

<!-- field: proof -->
### Proof

- mage audit
- mage integration:backlog
- mage integration:install

<!-- release: 04.3 | status: implemented -->
## 04.3 - tpm ships to machines and to CI

tpm and the declarative-agents runtime reach the two places they run:
people's machines and CI jobs. Each tag publishes tpm binaries with
checksums on its GitHub release and a signed multi-arch container image
(eng03). tpm install --runtime fetches the runtime at the release tpm's
own corpus pins, verified against the release's checksums, into the cache
mage audit uses, and builds the spec tool there when Go is present; tpm
doctor reports the runtime and repairs documents with that prebuilt tool
when Go is absent. The image carries both, so a CI job runs tpm and a
profile with no install step. Planned on 2026-10-21 and built on
2026-10-22 (design.md D12).

<!-- field: srds -->
### SRDs

- srd037-install
- srd043-init-onboard

<!-- field: use_cases -->
### Use cases

| id | summary | status | priority |
| --- | --- | --- | --- |
| [rel04.3-uc001-install-without-go](specs/use-cases/rel04.3-uc001-install-without-go.md) | A developer with no Go toolchain installs tpm from a release and readies the pinned runtime with tpm install --runtime. | implemented | P1 |
| [rel04.3-uc002-ci-job-on-the-image](specs/use-cases/rel04.3-uc002-ci-job-on-the-image.md) | A CI job runs tpm and the pinned runtime on the release's image with no install step and no Go toolchain. | implemented | P2 |

<!-- field: test_suites -->
### Test suites

| id | summary | status |
| --- | --- | --- |
| [test-rel04.3-distribution](specs/test-suites/test-rel04.3-distribution.md) | The runtime fetched and verified once per tag in the shared cache; TPM_AGENT held to the checksums; doctor's runtime report and prebuilt repair; the image's tpm, runtime, and spec tool at the pin. | implemented |

<!-- field: proof -->
### Proof

- mage audit
- mage integration:runtime
- mage integration:doctor
- mage integration:image
