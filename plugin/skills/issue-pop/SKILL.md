---
name: issue-pop
description: Take an issue into its own branch and worktree, and finish with the merge request that closes it (srd036). Use when a person asks to work, take, or pop an issue in this repository, or to finish the work with a merge request.
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
3. Commit in the worktree and push the branch (`git push`); the worktree
   already tracks the branch on the remote.
4. Run `tpm program open-merge-request --project <project> --issue <iid> --json` from
   the worktree, adding `--closes <project#iid>` for each issue the
   Requirements say the merge also closes. For a change under docs/ it runs the corpus audit there
   first. On a refusal (no commits yet, the issue closed, or the audit
   failing), show its reason and the audit's findings, and stop.
5. Show the merge request URL. A reviewer merges it; the merge closes the
   issue.

On the beads tracker (pop's `web_url` starts with `beads:`) there is no
merge request; delivery is a local merge (srd042 R2.3). Replace steps 4
and 5: from the checkout pop returned, run
`git merge --no-ff <branch> -m "Merge <branch> (<project>#<iid>)"`, then
`tpm program close --project <project> --issue <iid> --json`. On a refusal,
show its reason and stop; otherwise show the closing commit.
