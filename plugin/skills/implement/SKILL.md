---
name: implement
description: Take an issue into its own branch and worktree, and finish with the merge that closes it (srd036). Use when a person asks to work, take, or pop an issue in this repository, or to finish and merge the work.
---

# implement

Invoking this skill takes the issue. tpm decides whether it can be taken
and does every step around the work; this skill asks nothing and closes
nothing itself (design.md D10, srd036).

Input: the project path of this repository, the issue number, and the
program's system repository, whose control/ files hold the program's policy.

1. Run `tpm issue pop --project <project> --issue <iid> --control <system> --json`.
   When the issue's Requirements name `--issue-project` and `--branch` (a
   weekly status issue, whose report lands in the reports repository),
   pass them to pop, approve, and merge as written there.
   On a refusal, show its reason and stop.
2. Work in the worktree pop returned, and nowhere else. The issue body is
   the instructions: its Requirements are the steps and its Acceptance
   Criteria are what must hold. A question the Requirements ask, such as a
   sprint-end issue's confirmation, is part of the work (srd036 R4.1).
3. Commit in the worktree and push the branch (`git push -u origin HEAD`).
4. Ask a reviewer, someone other than you, to read the branch against the
   Acceptance Criteria and run
   `tpm issue approve --project <project> --issue <iid>` in their
   checkout. The approval names the commit they read.
5. Run `tpm issue merge --project <project> --issue <iid> --control <system> --json` from
   the worktree, adding `--closes <project#iid>` for each issue the
   Requirements say the merge also closes. It runs the corpus audit for a
   change under docs/, then merges only a branch approved at its tip,
   pushes, and closes each issue (srd042 R2.3, srd013 R5.5). On a refusal
   (no commits, the issue closed, the audit failing, or no approval of the
   tip), show its reason and stop; after a new commit, ask again.
6. Show the merge commit and the issues it closed.
