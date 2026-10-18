---
name: issue-pop
description: Take an issue into its own branch and worktree, and finish with the merge that closes it (srd036). Use when a person asks to work, take, or pop an issue in this repository, or to finish and merge the work.
---

# issue-pop

Invoking this skill takes the issue. tpm decides whether it can be taken
and does every step around the work; this skill asks nothing and closes
nothing itself (design.md D10, srd036).

Input: the project path of this repository and the issue number.

1. Run `tpm program pop --project <project> --issue <iid> --json`.
   On a refusal, show its reason and stop.
2. Work in the worktree pop returned, and nowhere else. The issue body is
   the instructions: its Requirements are the steps and its Acceptance
   Criteria are what must hold. A question the Requirements ask, such as a
   sprint-end issue's confirmation, is part of the work (srd036 R4.1).
3. Commit in the worktree.
4. Run `tpm program merge --project <project> --issue <iid> --json` from
   the worktree, adding `--closes <project#iid>` for each issue the
   Requirements say the merge also closes. For a change under docs/ it
   runs the corpus audit first; then it merges the branch into the default
   branch, pushes, and closes each issue (srd042 R2.3). On a refusal (no
   commits yet, the issue closed, or the audit failing), show its reason
   and the audit's findings, and stop.
5. Show the merge commit and the issues it closed.
