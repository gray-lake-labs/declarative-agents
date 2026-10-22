---
name: tasks
description: Break a level into the units below it (srd022), a release into one Sprint Goal per team or a story chosen at Sprint Planning into the issues that execute it. Use when the technical program manager pushes agreed initiatives downstream, or a Developer turns a chosen story into issues.
---

# tasks

Invoking this skill is the decision to file the units below. Filing is a
tpm command; the model writes only what tpm cannot compute, and nothing is
filed before the person agrees (design.md D10, srd022).

## A release into Sprint Goals

Input: the system repository and the road-map release version.

1. Run `tpm goals plan --system <system> --release <version> --json > plan.json`.
   On an error, show it and stop.
2. For each outcome with an empty repository, ask which of its candidates
   serves it and write the answer in. Ask nothing else.
3. Run `tpm goals push --system <system> --release <version> --plan plan.json --json`.
4. Show each outcome's action, its Sprint Goal's URL, and any refusal's
   reasons. A refused outcome is fixed in the initiative or the subsystem
   SRDs and this skill run again; filed Sprint Goals come back unchanged.

## A story into issues

Input: this repository's project path, the story id (or the Sprint Goal
listing the chosen stories), its milestone, and the system repository.

1. Read the story with `tpm backlog list --kind story --json`, and the
   issues already implementing it with
   `tpm issue list --project <project> --implements <story> --json`.
2. Propose the issues the story still needs, each a change a reviewer
   reads in one sitting, in build order. Each body is the issue body's
   work format: Requirements, Acceptance Criteria naming a test and an
   environment, Design Decisions where a choice binds, `Estimated LOC:`,
   `Implements: <story>`, and `Blocked-by:` for an issue that must close
   first.
3. Ask the owner to agree to, change, or drop each one.
4. File each agreed issue with
   `tpm issue file --project <project> --system <system> --title "<title>" --body-file <body> --milestone <milestone> --kind <kind> --json`,
   and show its reference and lint verdict; fix a needs-triage body and
   file again.
