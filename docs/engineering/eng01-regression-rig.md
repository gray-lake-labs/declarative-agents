---
id: eng01-regression-rig
title: "Regression Rig: A Program on Local Disk Per Run"
---

<!-- field: introduction -->
## Introduction

The process lives in the tracker and the repositories: objective issues,
due dates, deliveries, tags. A test that mocks them proves the mock. We
run every release's regression against a real program that the run builds
and removes, so each run starts from a known layout and leaves nothing
behind. Since the tracker became beads (srd042), that program needs no
service at all: a tracker repository and the code repositories, as git
repositories on local disk, are everything tpm reads and writes. The rig
therefore needs no network, no credentials, and no paid tier, and a full
regression runs in under a minute.

<!-- section: lifecycle -->
## Lifecycle

mage regression:up builds a run under .regression/<run-id>, which git
ignores, and records it as the current run; every integration target
reads the current run; mage regression:down removes it. mage regression
runs up, every integration target in order, and down, and down runs even
when a target fails, so a failed run leaves nothing behind but its
report. mage regression:sweep removes any run older than a day that an
interrupted regression left.

<!-- section: layout -->
## Layout

A run holds a bare tracker repository, tracker.git, whose first commit is
an empty .beads/issues.jsonl; and, for each rig project, a bare origin
under origins/ and a checkout under checkouts/, all named for the
project's last path segment. The projects are rig/system, rig/reports,
rig/product-owners/po-1, and rig/product-owners/po-2. The checkouts sit
side by side with work/, the checkout tpm runs in, which is how the beads
tracker finds a project's checkout (eng04-beads-sync, Choosing the
backend). Fixtures go on a seed branch apart from main, which the
targets' merges change; a target that clones a repository clones it under
clones/, so pop's worktrees land beside the clone inside the run.

<!-- section: people -->
## People

tpm runs as the rig's operator, with TPM_TRACKER naming the run's tracker
repository and the operator's own clone of it in TPM_TRACKER_CLONE. A
step another person takes, such as a release's peer read, runs tpm as the
reviewer with the reviewer's own clone, so the two people's writes meet
only in the tracker repository, as they would on two machines. A target
reads and writes the tracker beside tpm through the beads tracker in Go,
through yet another clone.

<!-- section: delivery -->
## Delivery

Work reaches a default branch only through tpm program merge, as on any
beads program: the audit gate for a docs/ change, a --no-ff merge citing
the issue and what it closes, a push, and the closes. The rig's
repositories carry no magefile, so a marker file, docs/BROKEN, stands in
for a failing mage audit.

<!-- section: unit-tests -->
## Unit tests

The rig itself is plain git and the beads tracker, which their own tests
cover; the tests every tpm package runs on both trackers through
internal/trackertest build the same kind of program per test. mage
regression is what proves the commands end to end, through the built
binary and the environment a person's machine gives it.

<!-- field: references -->
## References

- docs/engineering/eng04-beads-sync.md (the tracker repository and choosing the backend)
- docs/specs/software-requirements/srd042-beads-tracker.md R4.2 (the rig builds fixture tracker repositories)
- docs/specs/test-suites/test-rel01.0-objectives.md (first suite to run on the rig)
