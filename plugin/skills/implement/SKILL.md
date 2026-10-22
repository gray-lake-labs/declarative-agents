---
name: implement
description: Take an issue, or a story's issues, into a branch and worktree, build them, and finish with the one merge that closes them (srd036). Use when a person asks to work, take, or pop an issue or a story in this repository, or to finish and merge the work.
---

# implement

Invoking this skill takes the work. tpm decides whether it can be taken and
does every step around the work; this skill closes nothing itself (design.md
D10, srd036).

Input: this repository's project path; an issue number, or a story id (an
epic id only when the person asks for the whole epic); and the system
repository, whose control/ files hold the policy. Every tpm command below
takes `--project <project> --json`; pop and merge also take
`--control <system>`.

1. For a story, run `tpm issue list --implements <story>` (or
   `--epic <epic> --dir <checkout>`). Its `issues` are the work, in order;
   its `blocked` wait for a later run. None means the story needs `tasks`.
2. Run `tpm issue pop --issue <iid>` for the issue, or the story's first.
   When an issue's Requirements name `--issue-project` and `--branch` (a
   weekly status issue), pass them to pop, approve, and merge as written.
   Claim each further story issue into the same worktree and branch with
   `tpm issue pop --issue <iid> --worktree <worktree>`. On a refusal, show
   its reason; for a further issue, go on without it.
3. Work in that worktree only. Each issue's Requirements are its steps and
   its Acceptance Criteria what must hold; a question they ask is part of
   the work (srd036 R4.1). Commit each issue separately, its message
   holding its Requirements and Acceptance Criteria and ending
   `Closes <project#iid>`, and post its `Actual LOC:` line against its
   `Estimated LOC:` with `tpm issue comment --issue <iid> --body "..."`. An issue too big for one pass is split into
   sibling issues with `tasks`, never nested.
4. Push (`git push -u origin HEAD`). When the policy requires approval
   (`require_approval`; merge `--dry-run` refuses without one), ask a
   reviewer, not you, to read the branch against the Acceptance Criteria
   and run `tpm issue approve --project <project> --issue <first iid>` in
   their checkout; the approval names the commit they read.
5. From the worktree, run `tpm issue merge --issue <first iid>` with
   `--closes <project#iid>` for every other claimed issue and any issue
   the Requirements say the merge closes. It audits a docs/ change, merges
   (under `require_approval`, only a branch approved at its tip), pushes,
   and closes each issue (srd013 R5.5). On a refusal, show the reason and
   stop; after a new commit, ask for approval again.
6. Show the merge commit and the issues it closed.
