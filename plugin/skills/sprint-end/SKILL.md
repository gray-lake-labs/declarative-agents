---
name: sprint-end
description: Work the period's sprint-end issue, summarizing the period, asking the owner what to change, merging the summary, and releasing on confirmation (srd038 R1, R6). Use when the owner takes a sprint-end issue or asks to close the period.
---

# sprint-end

The sprint-end issue's Requirements are the steps; this skill names the
tpm command for each. The model writes only the summary; every fact comes
from tpm (design.md D10, srd038 R1).

Input: the project path, the sprint-end issue, the program's system
repository, a reports checkout, the period's first and last ISO weeks, and
the artifact the repository releases.

1. Take the issue with implement and work in its worktree.
2. Run `tpm sprint-end facts --project <project> --reports <checkout> --from <week> --to <week> --artifact <artifact> --control <system> --json`.
3. Model step (combine): draft the period's summary from the reports and
   those facts.
4. Ask the owner what to change in the summary, the Sprint Goals, or the
   release, which Sprint Goals are complete, and which stories each Sprint
   Goal lists (facts names them) are done. Apply the answers in the worktree.
5. Write the summary at docs/sprint-ends/<release>.md, run
   `tpm sprint-end stories --dir . --release <release> --project <project> --done <ids> --json`
   (a story of another team as project#story),
   commit it, and run
   `tpm issue merge --project <project> --issue <iid> --closes <project#iid> --control <system> --json`
   with one --closes per Sprint Goal the owner reports complete; the merge
   closes the issue and those Sprint Goals.
6. Ask the owner to confirm the release and its version (facts recommends
   one). With no confirmation, or nothing to release, stop here.
7. Once the summary merges, on main: `tpm release gate --artifact <artifact> --version <version> --issue <project#iid> --control <system>`,
   then `tpm release tag` with the same flags. Show the tag.
