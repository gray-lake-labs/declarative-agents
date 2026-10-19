---
name: issue-push
description: Turn one request into one admitted issue in this repository (srd035). Use when a person describes work, a bug, or a requirements change and asks to file, push, or open an issue for it.
---

# issue-push

Invoking this skill files the issue. tpm picks the labels, files, and lints;
the one model step is writing the body (design.md D10, srd035).

Input: the request, the project path of this repository, and the system
repository path.

1. Model step (structure): write the request as a title and a body with
   `## Requirements` (numbered steps) and `## Acceptance Criteria`, each
   criterion ending `(test: <name>; environment: <local|dev|staging>)`.
   Add `## Design Decisions` only for choices already made. Save the body
   to issue.md.
2. List the paths the request changes, as the request names them.
3. Ask the person only for what is missing: the kind of project work when
   no path is under docs/ or control/, and the test of a criterion that
   names none. Ask nothing else and no confirmation.
4. Run `tpm issue file --project <project> --system <system> --title "<title>"
   --body-file issue.md --path <path> ... [--kind <kind>] --json`.
   tpm sets the plane, kind, and component from the paths.
5. Show the issue URL, its plane, kind, and component, and whether
   admission admitted it. When it did not, show the reasons; fix the body and run
   `tpm issue lint --project <project> --issue <iid>`.
