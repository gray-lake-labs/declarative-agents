---
name: tasks
description: File the next release's objective issues in each Product Owner's repository (srd022, srd034 R3). Use when the technical program manager has an agreed objective document for the next release and asks to push its objectives downstream.
---

# tasks

Invoking this skill is the decision to file the release's objective issues.
Every step is a tpm command; this skill writes nothing itself and asks only
what tpm cannot compute (design.md D10, srd022 R3).

Input: the system repository path and the road-map release version.

1. Run `tpm goals plan --system <system> --release <version> --json > plan.json`.
   On an error, show it and stop.
2. For each objective in plan.json with an empty repository, ask the person
   which of the objective's candidates owns it, and write the answer into
   that objective's repository. Ask nothing else: tpm has taken every other
   owner and every environment.
3. Run `tpm goals push --system <system> --release <version> --plan plan.json --json`.
4. Show each objective's action (created, updated, unchanged, refused), its
   issue URL, and the reasons for any refusal. A refused objective is fixed
   in the objective document or the subsystem SRDs, and this skill is run
   again; objectives already filed come back unchanged.
