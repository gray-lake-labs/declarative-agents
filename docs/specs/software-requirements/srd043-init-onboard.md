---
id: srd043-init-onboard
title: "Script: init and onboard (tpm init, tpm onboard)"
---

<!-- field: design_notes -->
## Design notes

- Scaffolding and enforcement are separate. init and onboard make a corpus complete when it is created; keeping it complete afterwards is the audit's job. A check that a corpus holds its required documents belongs in the parent's corpus audit, so it gates every repository that carries a charter; a required-documents key in the policy was rejected because it would duplicate the auditor in tpm and fire only on tpm's paths.
- The registration of a member repository is filed, not written. The subsystem SRD that lists a component belongs to the system repository's owner and changes through that repository's gate, so onboard files the change as an issue and never commits there (design.md D4, D7).
- The registration is a plane::management issue: it changes a subsystem SRD, which is intent. Filed through tpm issue file under the system repository's policy, it is refused like any management issue when the policy names who may file one and the caller is not among them; the refusal says who may.
- tpm reports init (srd018 R1) stays its own subcommand. The reports repository has its own layout and owner, and folding it into onboard would change an implemented command for no reader's benefit.
- The templates are embedded in the binary beside the skills and written against the parent format release tpm was built with, so a tpm release and the corpus it scaffolds always agree.

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

<!-- field: goals -->
## Goals

- G1: One command turns an empty git repository into a program's system repository whose corpus passes the audit as written.
- G2: One command turns an existing repository into a member of a program, and files its registration with the system repository's owner.
- G3: Running either command again changes nothing, and nothing a person wrote is overwritten without --force.

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

<!-- req: R3 -->
## R3 - Common behavior

<!-- req: R3.1 -->
### R3.1

init and onboard honour --json and --dry-run, never prompt (srd010 R1), and refuse a --dir that is not a git repository.

<!-- req: R3.2 -->
### R3.2

Each file is reported as created, kept, or overwritten. A file that exists is kept and reported unless --force replaces it, so a second run changes nothing.

<!-- req: R3.3 -->
### R3.3

The templates are embedded in the tpm binary beside the skills. init and onboard write into docs/external-pins.yaml the parent release the templates were written against.

<!-- req: R3.4 -->
### R3.4

The templates name no person, organization, product, or project (design.md D6); what a program supplies, its subsystems, components, and environments, the owner writes into the skeleton.

<!-- field: non_goals -->
## Non-goals

- Creating repositories, groups, or permissions on a hosting service; the host or its provisioning does that before init runs.
- Keeping a corpus complete after it is created; that is the audit's.
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
