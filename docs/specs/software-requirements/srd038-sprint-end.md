---
id: srd038-sprint-end
title: "Skill: sprint-end"
---

<!-- field: problem -->
## Problem

At a period's end each releasing repository has to say what the period
produced, let its owner change what should change, and release only what
the owner confirms. Until track-goals runs, an objective the owner reports
complete also has to close somewhere without passing for evidence. A
sprint-end issue does both: tpm files it when the period is planned, its
steps run through the inner loop, and its merge request closes it and the
objectives the owner reports complete.

<!-- field: goals -->
## Goals

- G1: Each releasing repository ends each period with one sprint-end issue that summarizes the period, asks the owner, and releases only on confirmation, after its summary merges.
- G2: Before track-goals runs, an objective reported complete closes by the sprint-end merge as reported complete, never as met.

<!-- req: R1 -->
## R1 - Sprint-end issue

<!-- req: R1.1 -->
### R1.1

A sprint-end issue is a recurring audit filed once per period in each repository that releases, by tpm plan push when the period is planned (srd031 R4.6), due on the milestone end date. In the system repository it references the sprint issue (srd001 R7) and leaves its checklist to the program lead.

<!-- req: R1.2 -->
### R1.2

Popped, its Requirements direct the work in order: summarize the period from tpm sprint-end facts, which returns the period's aggregate (srd018 R3), each objective issue's ledger and reported status, and the recommended version (srd015 R4.1); ask the owner whether anything in the summary, the objectives, or the release should change; apply the changes the owner names in the worktree; commit the summary at docs/sprint-ends/RELEASE.md; open the merge request, listing each objective the owner reports complete with Closes (R2.1); once it merges, push the release from main with tpm release gate and tpm release tag, the sprint-end issue being the release issue.

<!-- req: R1.3 -->
### R1.3

The owner's confirmation of the release and its version is asked before the release is pushed and never assumed. The release is gated and tagged on main after the sprint-end merge request merges, so the tag names a merged revision; the gate is the policy's default definition_of_done unless the repository releases an artifact it declares (srd008 R2). A repository with nothing to release closes the sprint-end issue by that merge alone, with the summary committed and no tag (a merge close, srd041 R9.3).

<!-- req: R2 -->
## R2 - Closing an objective before track-goals runs

<!-- req: R2.1 -->
### R2.1

In a program that does not yet run track-goals, an objective issue closes by the merge request of its repository's sprint-end issue when the owner reports it complete; that merge request lists it with Closes. This is a merge close (srd041 R9.3).

<!-- req: R2.2 -->
### R2.2

Such a close posts no met event. The roll-up states the objective as reported complete, distinct from met (srd018 R3.5), so a reported close is never shown as evidence, and the aggregate does not count it as a contradiction of the ledger (srd018 R3.2).

<!-- req: R2.3 -->
### R2.3

Once track-goals runs in the program, R2.1 no longer applies and objectives close on met (srd019).

<!-- field: non_goals -->
## Non-goals

- Closing objectives on evidence; that is track-goals (srd019).
- Planning the next period; that is plan-release (srd031).

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 -->
### AC1

Each releasing repository has one sprint-end issue per period, filed by tpm plan push and due on the milestone end date; worked, it summarizes the period, asks the owner what to change, and closes by the merge request that commits the summary; the release is gated and tagged on main only after that merge and the owner's confirmation, or not at all.

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
- [R1.2](#r12)
- [R1.3](#r13)

<!-- ac: AC2 -->
### AC2

Without track-goals, a reported-complete objective closes by the sprint-end merge request, carries no met event, appears in the roll-up as reported complete, not met, and raises no contradiction signal; with track-goals it closes only on met.

<!-- field: traces -->
#### Traces

- [R2.1](#r21)
- [R2.2](#r22)
- [R2.3](#r23)
