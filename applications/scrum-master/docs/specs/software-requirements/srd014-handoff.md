---
id: srd014-handoff
title: "Script: handoff (scrum-master handoff ...)"
---
<!-- Copyright (c) 2026 Gray Lake Labs -->
<!-- SPDX-License-Identifier: MIT-0 -->

<!-- field: problem -->
## Problem

When one component's work lands, other components often owe something
because of it: an upstream issue's author should hear the downstream work
finished, consumers of a new release must adapt or at least know, older
adaptation work may be overtaken. Done by hand, these transfers are
forgotten or duplicated, and the person who must act finds out late.

Handoff turns two tracker events, a merge and a release tag, into the
comments and issues those events imply. It never assigns anyone; what it
creates enters the receiving repository's backlog. Who consumes an
interface is not a hand-kept map: it is the set of repositories that cite
the interface by pinned URL (srd007 R5.4), which track-goals collects.

<!-- field: goals -->
## Goals

- G1: A merge notifies the issue upstream of what it closed.
- G2: A release creates exactly the work and the notices its consumers are owed.
- G3: Running handoff twice over the same event changes nothing.

<!-- req: R1 -->
## R1 - scrum-master handoff on-merge

<!-- req: R1.1 -->
### R1.1

For a merged request, finds the issues it closed. For each closed issue with an Upstream line, posts a downstream-done event on the upstream issue (srd006 R3).

<!-- req: R1.2 -->
### R1.2

Creates no issue. A merge means done, not dependable; fan-out waits for a release (srd008 R2).

<!-- req: R1.3 -->
### R1.3

For each closed issue with a Downstream section that requests a review, creates one kind::review issue per requested component in the repository its subsystem SRD declares for that component, unassigned, with an Upstream line to the closed issue.

<!-- req: R2 -->
## R2 - scrum-master handoff on-release

<!-- req: R2.1 -->
### R2.1

For a new release tag, identifies the artifact and version from the tag format in policy and classifies the release as breaking or compatible by policy.releases.breaking against the previous tag.

<!-- req: R2.2 -->
### R2.2

Determines consumers from the open recurring issues subscribed to the artifact (srd001 R8.4) and from the repositories citing the artifact's interface by pinned URL, as track-goals last collected them (srd019).

<!-- req: R2.3 -->
### R2.3

For a breaking release, sets needs-run on each subscribing recurring issue (srd041 R10); for a citing repository without a subscription, creates one kind::consume issue in it, unassigned, with a Requires line at the new version, a Cites line pinned to the new tag, and an Upstream line to the release issue.

<!-- req: R2.4 -->
### R2.4

For every release, posts a release event on each subscribing recurring issue and each open consume issue for an earlier version of the artifact (srd006 R2).

<!-- req: R2.5 -->
### R2.5

For each open consume issue for an earlier version, also posts a superseded event (srd006 R4) and creates no second consume issue for the same consumer.

<!-- req: R2.6 -->
### R2.6

For a citing repository whose citation asks for a review or a handoff decision (a Cites line ending in | review or | handoff, config format issue-body), creates one issue of that kind in it, unassigned, with an Upstream line to the release issue.

<!-- req: R3 -->
## R3 - Tracking-issue membership

<!-- req: R3.1 -->
### R3.1

When the issue that caused a handoff is a member of a tracking issue, every issue handoff creates is appended to that tracking issue's Members section (issue-body tracking_task_list).

<!-- req: R4 -->
## R4 - scrum-master handoff sweep

<!-- req: R4.1 -->
### R4.1

Runs on-merge for every merged request and on-release for every release tag in the group since a given date. It is the fallback when a project's CI did not run handoff.

<!-- req: R5 -->
## R5 - Idempotency and findings

<!-- req: R5.1 -->
### R5.1

Every issue handoff creates carries a handoff marker naming the merge or tag that caused it. Before creating, handoff searches the target project for that marker and consumer and skips creation when it exists.

<!-- req: R5.2 -->
### R5.2

Events follow srd006 R7.2: no duplicate event with the same name and keys on the same issue.

<!-- req: R5.3 -->
### R5.3

A Downstream section naming an artifact no SRD declares, or a release of an artifact with neither subscriptions nor citing repositories, creates nothing for it, records a finding the dashboard shows (srd019), and makes the subcommand exit non-zero.

<!-- req: R6 -->
## R6 - Never assign, never close

<!-- req: R6.1 -->
### R6.1

Handoff sets no assignee, closes no issue, and changes no label except setting needs-run on recurring issues.

<!-- req: R7 -->
## R7 - Triggers and common behavior

<!-- req: R7.1 -->
### R7.1

The program's CI runs scrum-master handoff on-merge on merge to a default branch and scrum-master handoff on-release on a new tag, with a token scoped to the program group. Subcommands are in the scrum-master group handoff, honour --json and --dry-run, and never prompt.

<!-- req: R8 -->
## R8 - Reference update on a downstream release

<!-- req: R8.1 -->
### R8.1

A release tag in a downstream repository files one issue in each upstream repository whose initiatives list that repository in delivered_by, or whose Sprint Goals the release closed, naming the old and new tag.

<!-- req: R8.2 -->
### R8.2

The issue asks for the pinned references to move to the new tag through the inner loop (srd007 R5.5); no script edits a reference in place. Its Requirements name the files and fields to change.

<!-- req: R8.3 -->
### R8.3

It carries a handoff marker naming the tag, so a second run files nothing (R5.1). This is the slice of srd014 on-release a program runs before the rest of handoff.

<!-- field: non_goals -->
## Non-goals

- Deciding what a consumer must change; the consume issue's assignee decides.
- Notifying outside the tracker.
- Assigning or prioritising created issues.

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 -->
### AC1

A merge that closes an issue with an Upstream line posts one downstream-done event upstream, creates no consume issue, and creates one review issue per requested review component.

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
- [R1.2](#r12)
- [R1.3](#r13)

<!-- ac: AC2 -->
### AC2

A breaking tag marks every subscription needs-run, creates one unassigned consume issue per unsubscribed citing repository with Requires, Cites, and Upstream lines, posts release events, and posts superseded on older open consume issues without creating duplicates; a compatible tag only posts events; review and handoff consumers get one issue each.

<!-- field: traces -->
#### Traces

- [R2.1](#r21)
- [R2.2](#r22)
- [R2.3](#r23)
- [R2.4](#r24)
- [R2.5](#r25)
- [R2.6](#r26)

<!-- ac: AC3 -->
### AC3

Issues created because of a tracking-issue member are appended to the tracking issue's Members section.

<!-- field: traces -->
#### Traces

- [R3.1](#r31)

<!-- ac: AC4 -->
### AC4

sweep over a date range produces the same result as the CI runs would have; running sweep after CI creates and posts nothing new.

<!-- field: traces -->
#### Traces

- [R4.1](#r41)
- [R5.1](#r51)
- [R5.2](#r52)

<!-- ac: AC5 -->
### AC5

An undeclared or uncited artifact creates nothing, records a dashboard finding, and exits non-zero; no handoff run sets an assignee, closes an issue, or changes a label other than needs-run.

<!-- field: traces -->
#### Traces

- [R5.3](#r53)
- [R6.1](#r61)

<!-- ac: AC6 -->
### AC6

CI runs on-merge on merge and on-release on tag with a group-scoped token; subcommands honour --json and --dry-run and never prompt.

<!-- field: traces -->
#### Traces

- [R7.1](#r71)

<!-- ac: AC7 -->
### AC7

A downstream release tag files one reference-update issue in each upstream repository that lists or is closed by it, naming old and new tags and the fields to change; a second run files nothing; no reference is edited outside the inner loop.

<!-- field: traces -->
#### Traces

- [R8.1](#r81)
- [R8.2](#r82)
- [R8.3](#r83)
