---
id: srd036-implement
title: "Skill: implement"
---

<!-- field: problem -->
## Problem

An issue that is worked on the default branch, or in a checkout shared
with other work, mixes changes nobody can review apart, and closing it by
hand says nothing about what changed. The process has one path from an
issue to a change: the person who works it takes it, works on its own
branch in its own worktree, and the merge that lands the branch
closes it.

implement is that path. tpm claims the issue for the caller and opens its
branch and worktree (srd013 R7); the work in the worktree is the person's
or their agent's; tpm merges the branch, which closes the issue
(srd013 R5). A change under docs/, a design change or a planning change,
passes the corpus gate before it merges (srd034 R2.2).

<!-- field: goals -->
## Goals

- G1: Every issue is worked on its own branch in its own worktree, by the person who took it.
- G2: Only the merge of its branch closes the issue.
- G3: A change under docs/ merges only through the corpus gate, whatever the issue's plane or kind.

<!-- req: R1 -->
## R1 - Trigger

<!-- req: R1.1 -->
### R1.1

Started by the person who will work an open issue, in the issue's repository.

<!-- req: R2 -->
## R2 - Take

<!-- req: R2.1 -->
### R2.1

Runs tpm issue pop (srd013 R7), which assigns only the caller and creates or reuses the branch <iid>-<slug> and its worktree. A refusal is shown with its reason and the skill stops.

<!-- req: R2.2 -->
### R2.2

Every change for the issue is made in that worktree.

<!-- req: R2.3 -->
### R2.3

The worktree carries the skills of every agent installed in the checkout pop ran from (srd037), so a person or agent working there can finish the issue with implement.

<!-- req: R3 -->
## R3 - Finish

<!-- req: R3.1 -->
### R3.1

When the work is committed and pushed and a reviewer has approved its tip with tpm issue approve, runs tpm issue merge (srd013 R5), which runs the corpus gate for a docs/ change, merges the branch, and closes the issue; without the approval it refuses, and the skill asks for one.

<!-- req: R3.2 -->
### R3.2

For any issue whose branch changes docs/, a kind::spec design change or a plane::management planning change alike, runs mage audit before merging and reports its findings instead of merging when it fails.

<!-- req: R3.3 -->
### R3.3

The issue closes when its branch merges (srd041 R7.1); the skill closes nothing itself.

<!-- req: R4 -->
## R4 - Asks and refusals

<!-- req: R4.1 -->
### R4.1

Asks nothing: tpm decides whether the issue can be taken, and the issue body holds the steps. A question an issue's own Requirements ask, such as a sprint-end issue's confirmation (srd038 R1.3), belongs to the work, not to pop.

<!-- req: R4.2 -->
### R4.2

As a skill, is at most 50 lines and has no model step; the work in the worktree is the person's or their agent's, not the skill's (design.md D10).

<!-- field: non_goals -->
## Non-goals

- Splitting an issue into sub-issues and working them one by one; that arrives with tracking issues and do-work in a later release.
- Re-creating a recurring issue after its run; that is release 01.1 (tpm issue recur).
- Approving; a reviewer reads the branch and approves it (srd013 R5.6).

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 -->
### AC1

Popping an open, admitted issue assigns only the caller and returns its branch and worktree; a refusal stops the skill with its reason.

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
- [R2.1](#r21)
- [R2.2](#r22)

<!-- ac: AC2 -->
### AC2

Finishing merges the branch once, which closes the issue; an issue whose branch changes docs/ merges it only when mage audit passes; the skill closes nothing itself.

<!-- field: traces -->
#### Traces

- [R3.1](#r31)
- [R3.2](#r32)
- [R3.3](#r33)

<!-- ac: AC3 -->
### AC3

The skill asks nothing, is at most 50 lines, and has no model step.

<!-- field: traces -->
#### Traces

- [R4.1](#r41)
- [R4.2](#r42)

<!-- ac: AC4 -->
### AC4

A worktree pop creates from a checkout with installed skills holds the same agents' skills, stamped and ignored by git; a checkout with none installs none.

<!-- field: traces -->
#### Traces

- [R2.3](#r23)
