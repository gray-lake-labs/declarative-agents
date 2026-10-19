---
id: srd007-links
title: Dependency links
---

<!-- field: problem -->
## Problem

Work in one repository often cannot finish until work in another has. The
free tier of the tracker has no blocking links, so a dependency either goes
unrecorded or is recorded as prose nobody's script can read. Then a person
starts work that cannot finish, or waits on something that already landed
without anyone telling them.

Dependencies are two lines in an issue's description: one naming the issue
whose output this issue consumes, one naming a release this issue needs.
track-goals (srd019) reads them, decides whether the issue is waiting, and reports
the moment it is not.

Requirements cross repositories too. A subsystem's requirement satisfies a
system requirement in the TPM repository; an objective cites the
requirements it must prove. A citation by bare path or by branch drifts
as the cited file changes, so every cross-repository reference is a URL
pinned to a release tag.

<!-- field: goals -->
## Goals

- G1: A dependency on an issue or on a release is recorded in a form scripts read.
- G2: Whether an issue is waiting is computed, never declared by hand.
- G3: The person whose issue becomes unblocked hears about it the next morning without asking.
- G4: Every cross-repository reference names an exact released version of what it cites.

<!-- req: R1 -->
## R1 - Upstream line

<!-- req: R1.1 -->
### R1.1

A line of the form Upstream: PROJECT#IID in an issue's description names the issue that produced the artifact this issue consumes or responds to. An issue has at most one Upstream line.

<!-- req: R1.2 -->
### R1.2

Upstream points at the producing issue, not at an epic or a sprint. Handoff writes it on every issue it creates.

<!-- req: R1.3 -->
### R1.3

Every Upstream line is mirrored as a related-issue link between the two issues, so the tracker shows the relation in both directions.

<!-- req: R1.4 -->
### R1.4

An Upstream dependency is satisfied when the named issue is closed.

<!-- req: R2 -->
## R2 - Requires line

<!-- req: R2.1 -->
### R2.1

A line of the form Requires: ARTIFACT >= VERSION names a released artifact and the lowest version this issue can use. An issue may have several Requires lines.

<!-- req: R2.2 -->
### R2.2

A Requires dependency is satisfied when the artifact's producing repository has a version tag at or above VERSION, compared by semantic versioning.

<!-- req: R2.3 -->
### R2.3

A recurring issue's Requires line records the version its consumer is on (srd001 R8), and finishing a consume run updates it.

<!-- req: R3 -->
## R3 - Waiting and unblocked

<!-- req: R3.1 -->
### R3.1

Track-goals sets waiting (srd041 R10) on an open issue while any of its Upstream or Requires dependencies is unsatisfied, and clears it when all are satisfied.

<!-- req: R3.2 -->
### R3.2

An issue whose waiting flag track-goals cleared since the person's previous morning run is reported to its assignee as unblocked since yesterday.

<!-- req: R3.3 -->
### R3.3

A dependency naming an issue or project track-goals cannot read is reported as unresolvable and leaves the issue waiting. Track-goals never guesses that a dependency is satisfied.

<!-- req: R4 -->
## R4 - Cycles

<!-- req: R4.1 -->
### R4.1

Track-goals builds the graph of open issues and their Upstream lines, and reports every cycle it finds as a finding listing the issues in the cycle.

<!-- req: R4.2 -->
### R4.2

A cycle is a finding for the program lead, not an error. Track-goals keeps evaluating every other issue, and issues in the cycle stay waiting.

<!-- req: R4.3 -->
### R4.3

Cycle findings appear in the program's weekly roll-up and dashboard until the cycle is broken.

<!-- req: R5 -->
## R5 - Pinned references

<!-- req: R5.1 -->
### R5.1

A cross-repository reference is a URL of the form https://HOST/GROUP/REPO/-/blob/TAG/PATH#ANCHOR, where TAG is a release tag of REPO, PATH is a file in it, and ANCHOR is a requirement id (Rn or Rn.m) or an interface name in that file. In an issue description it is a line Cites: URL; in a corpus document it is a field value.

<!-- req: R5.2 | derived_from: [{"repo": "github.com/gray-lake-labs/declarative-agents", "release": "v0.20261018.1", "srd": "srd014-spec-graph", "item": "R3.16", "reason": "The parent resolves a cross-repository citation only at its validated release through a local clone cache; a citation here resolves the same way, at its tag, so the parent's audit can check it."}] -->
### R5.2

A reference resolves when the tag exists, the path exists at that tag, and the anchor names an entry of the file at that tag.

<!-- req: R5.3 | derived_from: [{"repo": "github.com/gray-lake-labs/declarative-agents", "release": "v0.20261018.1", "srd": "srd014-spec-graph", "item": "R3.18", "reason": "The parent reports a citation without a release as unpinned; this corpus inherits that rule and makes it fail the audit."}] -->
### R5.3

A reference to a branch, to a commit not reachable from a tag, or with no anchor is unpinned. An unpinned or unresolvable reference is a finding that fails the audit; it is never treated as satisfied.

<!-- req: R5.4 -->
### R5.4

The set of repositories that cite an interface by pinned URL is that interface's list of consumers. Handoff reads consumers from it.

<!-- req: R5.5 -->
### R5.5

Moving a reference to a newer tag is an edit made through the inner loop, with the old and new tags in the merged change, so the change of what is cited is reviewed.

<!-- field: non_goals -->
## Non-goals

- Blocking links of the paid tier.
- Dependencies on tasks; dependencies are between issues or on releases, because a task is a step, not a commitment.
- Ordering work by its dependencies; scheduling is a policy that may read waiting, not part of this SRD.

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 -->
### AC1

Every issue handoff creates carries one Upstream line naming the producing issue and a matching related link; an issue with an Upstream line is waiting until that issue closes.

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
- [R1.2](#r12)
- [R1.3](#r13)
- [R1.4](#r14)

<!-- ac: AC2 -->
### AC2

An issue with Requires ARTIFACT >= 1.2 is waiting while the latest tag is 1.1 and stops waiting when 1.2 or 2.0 is tagged; a finished consume run updates its recurring issue's Requires version.

<!-- field: traces -->
#### Traces

- [R2.1](#r21)
- [R2.2](#r22)
- [R2.3](#r23)

<!-- ac: AC3 -->
### AC3

The waiting flag tracks the dependencies exactly; an issue that stopped waiting since the last morning run is listed as unblocked for its assignee; an unreadable dependency keeps the issue waiting and is reported.

<!-- field: traces -->
#### Traces

- [R3.1](#r31)
- [R3.2](#r32)
- [R3.3](#r33)

<!-- ac: AC4 -->
### AC4

Two issues whose Upstream lines name each other produce one cycle finding listing both, both stay waiting, every other issue is still resolved, and the finding appears in the roll-up until the cycle is broken.

<!-- field: traces -->
#### Traces

- [R4.1](#r41)
- [R4.2](#r42)
- [R4.3](#r43)

<!-- ac: AC5 -->
### AC5

A Cites URL with an existing tag, path, and anchor resolves; one naming a branch, a missing tag, a missing path, or a missing anchor is reported as a finding and fails the audit; the citing repositories of an interface are its consumers; retargeting a citation is a reviewed merge.

<!-- field: traces -->
#### Traces

- [R5.1](#r51)
- [R5.2](#r52)
- [R5.3](#r53)
- [R5.4](#r54)
- [R5.5](#r55)
