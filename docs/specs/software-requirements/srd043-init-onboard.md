---
id: srd043-init-onboard
title: "Script: init, onboard, and doctor (tpm init, tpm onboard, tpm doctor)"
---

<!-- field: design_notes -->
## Design notes

- Scaffolding and enforcement are separate. init and onboard make a corpus complete when it is created; keeping it complete afterwards is the audit's job. A check that a corpus holds its required documents belongs in the parent's corpus audit, so it gates every repository that carries a charter; a required-documents key in the policy was rejected because it would duplicate the auditor in tpm and fire only on tpm's paths.
- The registration of a member repository is filed, not written. The subsystem SRD that lists a component belongs to the system repository's owner and changes through that repository's gate, so onboard files the change as an issue and never commits there (design.md D4, D7).
- The registration is a plane::management issue: it changes a subsystem SRD, which is intent. Filed through tpm issue file under the system repository's policy, it is refused like any management issue when the policy names who may file one and the caller is not among them; the refusal says who may.
- tpm reports init (srd018 R1) stays its own subcommand. The reports repository has its own layout and owner, and folding it into onboard would change an implemented command for no reader's benefit.
- The templates are embedded in the binary beside the skills and written against the parent format release tpm was built with, so a tpm release and the corpus it scaffolds always agree.
- doctor is its own command, not a second run of init or onboard. init is for an empty repository and onboard files a registration, so neither is the step to take after upgrading tpm; doctor reads the level from the files it finds, touches only what tpm owns or can repair without changing meaning, and files and commits nothing.
- Only a tpm-managed block is refreshed. The orientation files are tpm's account of its own process, so a newer tpm updates them; text a person writes around the block is theirs.
- A block is appended to an orientation file that has none, rather than the file being kept as edited. AGENTS.md at the top level is the one place every agent reads, so tpm stays agent-agnostic by writing there, and a repository that joins a program usually already has its own AGENTS.md and README.md; keeping them untouched would leave tpm's text out, and --force would replace the owner's. An owner who deletes the block gets it back on the next run. Every other skeleton file belongs to its owner once written, so it is never refreshed and R3.2 holds for it.
- Repair builds the spec tool at run time. The formatter is the pinned parent release's own library, built from that release's source as mage fmt builds it (srd040 R1.1). Embedding the library would need a replace directive in tpm's go.mod, which go install does not honour, and would format every repository to tpm's release rather than to its own pin. The cost is a Go toolchain on the machine; without one, doctor skips repair and says so.

<!-- field: problem -->
## Problem

Nothing creates a complete program repository. The corpus skeleton (VISION,
ARCHITECTURE, SPECIFICATIONS, road-map, the design constitution, the corpus
charter, and the parent pins), the control files, and the skills are
assembled by hand in every repository. A document left out is never
noticed: the audit checks the documents that exist, and the formatter
quietly unlinks an index row whose document is missing. Seeding a program
for a regression run took a dozen manual steps that tpm should own.

Every repository of a program carries the same corpus (design.md D7), so
the corpus can be written once, embedded in tpm, and laid down by one
command: init for the system repository a program starts from, onboard
for each repository that joins it.

A repository laid down this way does not explain itself. A first-time
reader finds docs/specs/themes/, initiatives/, and epics/ full of numbered
documents and nothing saying what each level is, who owns it, or which tpm
command reads it; the answer is in the design constitution, which nobody
reads first. An agent opening the repository is not told which tpm
commands and skills operate there. And a repository laid down by an older
tpm cannot catch up: a newer tpm scaffolds files the older one did not,
and documents drift from the pinned format with no formatter in reach,
since the format tools run only in this repository's mage (srd040).

<!-- field: goals -->
## Goals

- G1: One command turns an empty git repository into a program's system repository whose corpus passes the audit as written.
- G2: One command turns an existing repository into a member of a program, and files its registration with the system repository's owner.
- G3: Running either command again changes nothing, and nothing a person wrote is overwritten without --force.
- G4: A first-time reader, person or agent, learns from the repository itself what each hierarchy level is, who owns it, and which tpm commands and skills operate on it.
- G5: One command brings a repository an older tpm laid down up to date: missing files added, tpm's orientation text refreshed, documents repaired to the pinned format, and nothing a person wrote lost.

<!-- req: R1 -->
## R1 - tpm init

<!-- req: R1.1 -->
### R1.1

Writes into --dir, by default the root of the git repository holding the working directory, the system repository's corpus skeleton: docs/VISION.md, docs/ARCHITECTURE.md, docs/SPECIFICATIONS.md, docs/road-map.md with one release, docs/constitutions/design.md, docs/corpus-charter.yaml, docs/external-pins.yaml, and a system SRD, docs/specs/software-requirements/srd001-system.md, which that release lists. The parent's corpus audit loads no corpus without an SRD and no road-map without a release, so the skeleton carries one of each for the owner to rewrite.

<!-- req: R1.2 -->
### R1.2

Every skeleton document is in the format of the parent release tpm was built with, holds only fields that release declares, and states in each field what the owner writes there. The skeleton as written passes the parent's corpus audit (docs/guides/corpus-audit.md) and the hierarchy gate.

<!-- req: R1.3 -->
### R1.3

Writes control/labels.md and control/policy.md holding the built-in defaults, in the config formats labels and policy.

<!-- req: R1.4 -->
### R1.4

Installs the skills for the agents --agent names, through tpm install (srd037).

<!-- req: R1.5 -->
### R1.5

With --tracker naming a tracker repository that has no commits, makes its first commit holding an empty issues file (srd042) and creates the program's labels there (srd013 R6). A tracker repository that already has commits is left as it is.

<!-- req: R1.6 -->
### R1.6

Writes the orientation files at the system level: README.md and AGENTS.md at the root of --dir, and a README.md in docs/specs/themes/, docs/specs/initiatives/, and docs/specs/epics/. The root README says what the repository is, the order to read its corpus in, and the chain theme, initiative, epic, feature, story with each level's owner and timeframe. Each folder README says what its level is, its owner, timeframe, parent kind, file naming, and status values, and which tpm commands and skills read it. AGENTS.md lists the tpm command groups and the skills that apply at the level, read from the binary's dispatch table and embedded skills, and the working rules: intent changes through the gate, work enters as an issue, and every figure is Mermaid in a fenced code block inside the Markdown document, with no generated image file or other diagram source under docs/.

<!-- req: R2 -->
## R2 - tpm onboard

<!-- req: R2.1 -->
### R2.1

Writes into --dir, a checkout of the repository joining the program, the same corpus skeleton as init at the subsystem's level: a subsystem SRD, docs/specs/software-requirements/srd001-subsystem.md, in place of the system SRD, every field naming the Product Owner as the one who writes it, and no control files, which only the system repository holds.

<!-- req: R2.2 -->
### R2.2

Files one issue in the system repository that --system names, through tpm issue file (srd013 R3), asking that the subsystem SRD --subsystem names list the repository in its components table, with the repository's URL and path. It never commits to the system repository.

<!-- req: R2.3 -->
### R2.3

Files no second registration while one for the same repository is open or the subsystem SRD already lists it, and says which.

<!-- req: R2.4 -->
### R2.4

Writes the orientation files at the subsystem level, as R1.6 states them: README.md and AGENTS.md at the root, and a README.md in docs/specs/epics/, docs/specs/features/, and docs/specs/stories/, the levels a team's repository holds (srd045 R2.2), and none in themes/ or initiatives/.

<!-- req: R3 -->
## R3 - Common behavior

<!-- req: R3.1 -->
### R3.1

init, onboard, and doctor honour --json and --dry-run, never prompt (srd010 R1), and refuse a --dir that is not a git repository.

<!-- req: R3.2 -->
### R3.2

Each file is reported as created, kept, appended or refreshed (R3.5), or overwritten. A file that exists is kept and reported unless --force replaces it, so a second run changes nothing.

<!-- req: R3.3 -->
### R3.3

The templates are embedded in the tpm binary beside the skills. init and onboard write into docs/external-pins.yaml the parent release the templates were written against.

<!-- req: R3.4 -->
### R3.4

The templates name no person, organization, product, or project (design.md D6); what a program supplies, its subsystems, components, and environments, the owner writes into the skeleton.

<!-- req: R3.5 -->
### R3.5

An orientation file holds tpm's text between a begin marker and an end marker. A run whose text for the block differs from the file's replaces the block, keeps every line outside it, and reports the file refreshed; a person's text outside the block is theirs. An orientation file that exists without a block gets the block appended after its text, separated by one blank line, and is reported appended. No other file carries a block.

<!-- req: R4 -->
## R4 - tpm doctor

<!-- req: R4.1 -->
### R4.1

Runs in --dir, by default the root of the git repository holding the working directory, and reads the repository's level from the files it finds: system when control/policy.md or the system SRD is there, subsystem when the subsystem SRD is. A directory that holds neither is refused with the reason.

<!-- req: R4.2 -->
### R4.2

Writes every file init (at system level) or onboard (at subsystem level) writes that is missing, and appends or refreshes the orientation files' blocks (R3.5). Every other existing file is kept; doctor offers no --force. It files no issue, installs no skill, and touches no tracker.

<!-- req: R4.3 -->
### R4.3

Reports, without changing them, the skills installed for each agent that tpm install --check finds missing, outdated, edited, or stale (srd037 R4.1), and names tpm install as the fix.

<!-- req: R4.4 -->
### R4.4

Repairs the corpus documents, the hierarchy documents among them: rewrites each in the canonical form of the parent release docs/external-pins.yaml pins, with that release's own formatter built as srd040 R1.1 builds it, then reports each document repaired and each finding the formatter cannot repair, with its fix hint. A repair never changes what a field says and never moves the pin.

<!-- req: R4.5 -->
### R4.5

Without a Go toolchain or the pinned release's source, skips the repair, says which is missing, and does the rest.

<!-- req: R4.6 -->
### R4.6

A dry run reports what doctor would write, append, refresh, and repair and changes nothing. A second run reports nothing created, appended, refreshed, or repaired.

<!-- field: non_goals -->
## Non-goals

- Creating repositories, groups, or permissions on a hosting service; the host or its provisioning does that before init runs.
- Enforcing that a corpus stays complete; that is the audit's. doctor adds what is missing when it is run and gates nothing.
- Installing or upgrading skills; that is tpm install (srd037).
- The reports repository; that is tpm reports init (srd018 R1).
- Moving an existing corpus to a newer parent format; that is the parent-upgrade workflow (docs/guides/parent-upgrade.md).

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 -->
### AC1

init in an empty git repository writes VISION, ARCHITECTURE, SPECIFICATIONS, a road-map with one release, the design constitution, the charter, the pins, and the system SRD in the parent's format, the default labels and policy, and the skills for the named agents; the parent's corpus audit and the Sprint Goal gate pass on the result.

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
- [R1.2](#r12)
- [R1.3](#r13)
- [R1.4](#r14)

<!-- ac: AC2 -->
### AC2

init with an empty tracker repository makes its first commit and creates the labels there; with a tracker repository that has commits it changes nothing there.

<!-- field: traces -->
#### Traces

- [R1.5](#r15)

<!-- ac: AC3 -->
### AC3

onboard in a member checkout writes the skeleton without control files and files one registration issue in the system repository, with nothing committed there; a second onboard files none and says why.

<!-- field: traces -->
#### Traces

- [R2.1](#r21)
- [R2.2](#r22)
- [R2.3](#r23)

<!-- ac: AC4 -->
### AC4

Both commands honour --json and --dry-run, never prompt, refuse a directory that is not a git repository, report each file as created, kept, or overwritten, keep an existing file unless --force, change nothing on a second run, and write the templates' parent release into the pins; no template names a person, organization, product, or project.

<!-- field: traces -->
#### Traces

- [R3.1](#r31)
- [R3.2](#r32)
- [R3.3](#r33)
- [R3.4](#r34)

<!-- ac: AC5 -->
### AC5

init writes README.md, AGENTS.md, and README.md in themes/, initiatives/, and epics/; onboard writes README.md, AGENTS.md, and README.md in epics/, features/, and stories/ and none in themes/ or initiatives/; AGENTS.md names every command group and skill at its level and the Mermaid figure rule; the parent's corpus audit and the hierarchy gate pass on both results.

<!-- field: traces -->
#### Traces

- [R1.6](#r16)
- [R2.4](#r24)

<!-- ac: AC6 -->
### AC6

On a repository an older tpm laid down, doctor adds exactly the missing files, refreshes a changed block and keeps the text around it, appends the block to an orientation file that has none, reports stale skills without installing them, refuses a directory that is not a program repository, writes nothing on a dry run, and changes nothing on a second run.

<!-- field: traces -->
#### Traces

- [R3.5](#r35)
- [R4.1](#r41)
- [R4.2](#r42)
- [R4.3](#r43)
- [R4.6](#r46)

<!-- ac: AC7 -->
### AC7

doctor rewrites a document whose fields stray from canonical form so the audit passes, reports a finding it cannot repair with its fix hint and changes nothing else, and without Go on PATH skips the repair, says why, and does the rest.

<!-- field: traces -->
#### Traces

- [R4.4](#r44)
- [R4.5](#r45)
