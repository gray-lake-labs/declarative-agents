---
name: tasks
description: File the next release's Sprint Goal for each team, one dated issue per team serving the initiative outcomes confirmed for it (srd022, srd034 R3). Use when the technical program manager has agreed initiatives for the next release and asks to push them downstream.
---

# tasks

Invoking this skill is the decision to file the release's Sprint Goals.
Every step is a tpm command; this skill writes nothing itself and asks only
what tpm cannot compute (design.md D10, srd022 R3).

Input: the system repository path and the road-map release version.

1. Run `tpm goals plan --system <system> --release <version> --json > plan.json`.
   On an error, show it and stop.
2. For each outcome in plan.json with an empty repository, ask the person
   which of its candidates is the team that serves it, and write the answer
   into that outcome's repository. Ask nothing else: tpm has taken every
   other team and every environment.
3. Run `tpm goals push --system <system> --release <version> --plan plan.json --json`.
   tpm files one Sprint Goal per team, serving every outcome confirmed for
   that team's repository.
4. Show each outcome's action (created, updated, unchanged, refused), its
   Sprint Goal's URL, and the reasons for any refusal. An outcome marked
   legacy is served by an objective issue filed before release 04.0, which
   stays as it is. A refused outcome is fixed in the initiative or the
   subsystem SRDs, and this skill is run again; Sprint Goals already filed
   come back unchanged.
