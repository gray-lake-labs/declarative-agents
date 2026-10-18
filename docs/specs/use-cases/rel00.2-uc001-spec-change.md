---
id: rel00.2-uc001-spec-change
title: A Developer Changes a Requirement During Design Through the Audit Gate
status: implemented
priority: P1
test_suite: test-rel00.2-docs-change
---

<!-- field: summary -->
## Summary

Requirements change during design: a Developer adds a component SRD,
rewrites a use case, or adjusts a test suite. issue-push files the request
as a kind::spec data issue, anyone's and in the backlog. The Developer pops
it and changes docs/ in the worktree; issue-pop runs mage audit and opens
no merge request while it fails; once it passes, the merge request names
mage audit and its merge closes the issue.

<!-- field: actor -->
## Actor

A Developer (a member of the program).

<!-- field: trigger -->
## Trigger

A design change to a component SRD, use case, test suite, or the architecture.

<!-- field: preconditions -->
## Preconditions

- The repository carries the docs/ corpus and runs mage audit.
- Push and pop work (rel00.0-uc001-push-project-work, rel00.1-uc001-pop-and-merge).

<!-- field: flow -->
## Flow

- F1: The Developer runs issue-push with a request that names a docs/ path other than a planning document.
- F2: Model step (structure): the model turns the request into the issue body.
- F3: tpm sets plane::data and kind::spec and files the issue unassigned through tpm program file; it is admitted into the backlog.
- F4: The Developer pops it into its worktree and changes the documents there.
- F5: issue-pop runs mage audit; a failure is reported with its findings and no merge request is opened.
- F6: With the audit passing, tpm program open-merge-request opens the merge request, whose reviewer focus names mage audit; its merge closes the issue.

<!-- field: touchpoints -->
## Touchpoints

| id | target | reason |
| --- | --- | --- |
| T1 | [srd001-labels](../software-requirements/srd001-labels.md) [R33.1](../software-requirements/srd001-labels.md#r331), [R33.2](../software-requirements/srd001-labels.md#r332), [R33.3](../software-requirements/srd001-labels.md#r333) | S1 traces kind::spec. |
| T2 | [srd035-issue-push](../software-requirements/srd035-issue-push.md) [R2.2](../software-requirements/srd035-issue-push.md#r22) | S1 traces how push picks the kind. |
| T3 | [srd036-issue-pop](../software-requirements/srd036-issue-pop.md) [R3.2](../software-requirements/srd036-issue-pop.md#r32) | S2 traces the audit gate. |
| T4 | [srd013-program](../software-requirements/srd013-program.md) [R5.3](../software-requirements/srd013-program.md#r53) | S2 traces the reviewer focus. |
| T5 | [srd034-objectives](../software-requirements/srd034-objectives.md) [R2.1](../software-requirements/srd034-objectives.md#r21), [R2.2](../software-requirements/srd034-objectives.md#r22) | S2 traces the corpus gate. |
| T6 | [srd001-labels](../software-requirements/srd001-labels.md) [AC24](../software-requirements/srd001-labels.md#ac24) | S1 traces this acceptance criterion. |
| T7 | [srd035-issue-push](../software-requirements/srd035-issue-push.md) [AC1](../software-requirements/srd035-issue-push.md#ac1) | S1 traces this acceptance criterion. |
| T8 | [srd036-issue-pop](../software-requirements/srd036-issue-pop.md) [AC2](../software-requirements/srd036-issue-pop.md#ac2) | S2 traces this acceptance criterion. |
| T9 | [srd013-program](../software-requirements/srd013-program.md) [AC5](../software-requirements/srd013-program.md#ac5) | S2 traces this acceptance criterion. |
| T10 | [srd034-objectives](../software-requirements/srd034-objectives.md) [AC2](../software-requirements/srd034-objectives.md#ac2) | S2 traces this acceptance criterion. |

<!-- field: success_criteria -->
## Success criteria

<!-- sc: S1 -->
### S1

A request naming a docs/ path other than a planning document becomes an admitted kind::spec data issue that any member may file and pop.

<!-- field: traces -->
#### Traces

- [srd001-labels](../software-requirements/srd001-labels.md) [R33.1](../software-requirements/srd001-labels.md#r331)
- [srd001-labels](../software-requirements/srd001-labels.md) [R33.2](../software-requirements/srd001-labels.md#r332)
- [srd001-labels](../software-requirements/srd001-labels.md) [R33.3](../software-requirements/srd001-labels.md#r333)
- [srd001-labels](../software-requirements/srd001-labels.md) [AC24](../software-requirements/srd001-labels.md#ac24)
- [srd035-issue-push](../software-requirements/srd035-issue-push.md) [R2.2](../software-requirements/srd035-issue-push.md#r22)
- [srd035-issue-push](../software-requirements/srd035-issue-push.md) [AC1](../software-requirements/srd035-issue-push.md#ac1)

<!-- sc: S2 -->
### S2

The merge request opens only when mage audit passes, names mage audit in its reviewer focus, and closes the issue when merged.

<!-- field: traces -->
#### Traces

- [srd036-issue-pop](../software-requirements/srd036-issue-pop.md) [R3.2](../software-requirements/srd036-issue-pop.md#r32)
- [srd036-issue-pop](../software-requirements/srd036-issue-pop.md) [AC2](../software-requirements/srd036-issue-pop.md#ac2)
- [srd013-program](../software-requirements/srd013-program.md) [R5.3](../software-requirements/srd013-program.md#r53)
- [srd013-program](../software-requirements/srd013-program.md) [AC5](../software-requirements/srd013-program.md#ac5)
- [srd034-objectives](../software-requirements/srd034-objectives.md) [R2.1](../software-requirements/srd034-objectives.md#r21)
- [srd034-objectives](../software-requirements/srd034-objectives.md) [R2.2](../software-requirements/srd034-objectives.md#r22)
- [srd034-objectives](../software-requirements/srd034-objectives.md) [AC2](../software-requirements/srd034-objectives.md#ac2)

<!-- field: out_of_scope -->
## Out of scope

- Planning documents; that is rel00.2-uc002-planning-change.
- Running mage audit in CI on the merge request; filed as GH-150.

<!-- field: assumptions -->
## Assumptions

- A1: Steps not marked as model steps are tpm subcommands, CI, or the inner loop, and spend no model tokens (design.md D10).
