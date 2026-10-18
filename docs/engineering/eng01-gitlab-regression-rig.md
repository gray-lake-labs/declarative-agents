---
id: eng01-gitlab-regression-rig
title: "GitLab Regression Rig: Real Repositories Per Run"
---

<!-- field: introduction -->
## Introduction

The process lives in the tracker: objective issues, due dates, merge
requests, tags, and the pipelines tags trigger. A test that mocks the
tracker proves the mock. We run every release's regression against real
repositories that the run creates and destroys, so each run starts from a
known layout and leaves nothing behind.

We follow the rig pattern of the parent, declarative-agents (its kindrig
and gcprig packages): a Go package in magefiles/, here gitlabrig, that
creates resources by name, observes readiness instead of assuming it,
removes only what it created, reads settings from an optional checked-in
file with literal defaults in Go, and is unit-tested with a recording
runner. Unlike gcprig, gitlabrig drives Terraform, because the program's
GitLab layout is already Terraform (infra/platform) and the rig must build
the same layout, not a second description of it.

<!-- section: lifecycle -->
## Lifecycle

A run has four mage targets. mage regression:up runs preflight, then
applies infra/regression into a subgroup named rig-<run-id> under the
configured group: a system repository, a reports repository, and N
Product Owner repositories, with the program labels and the seeded
recurring issues. GitLab sometimes creates a project without the
initial commit initialize_with_readme asks for, so up then checks every
repository and commits a README to main where one is missing, since
later targets branch from main. It writes the Terraform outputs (group path,
repository paths and ids, web URLs) as JSON to the run directory,
where every integration target reads them. mage regression:down
deletes the run's subgroup through the API and removes the run
directory. mage regression:all is up, then the release's
integration targets, then down. down always runs: the target defers it,
so a failing integration target still leaves the group empty. mage
regression:sweep deletes rig-* subgroups older than the configured
age, for runs that died before down.

A test suite states which repositories it needs in its preconditions
(for example, system and two Product Owner repositories), and the
integration targets take N from there. A run creates only what its
suite needs.

<!-- section: terraform-layout -->
## Terraform layout

The program layout lives in one module, infra/modules/program: the
subgroups, projects, branch protection, labels, seeded recurring
issues, and optional service accounts, with the structure passed in as
variables. Two roots use it. infra/platform is production: every
project carries prevent_destroy, because project paths are targets of
pinned citations (D9), and its state lives in GitLab-managed Terraform
state. infra/regression is disposable: no prevent_destroy, and local
state in the run directory, one state per run.

Terraform does not accept a variable in a lifecycle block, so the
module declares two project resources, one with prevent_destroy and
one without, selected by a variable, and exposes one merged map to
everything that references a project. moved blocks in infra/platform
keep production's state addresses, so the split plans as no change.

<!-- section: naming-and-teardown -->
## Naming and teardown

Every resource the rig creates sits under rig-<run-id>, where run-id is
the UTC timestamp and a short random suffix. down deletes that one
subgroup, which takes every project, label, and issue in it, and
nothing else; sweep removes only subgroups whose names match rig-* and
whose creation time is older than the configured age.

down does not use terraform destroy for the projects. On instances
that delay deletion, the provider's permanent project deletion is
refused until the project is marked for deletion, so destroy fails.
The rig instead marks the subgroup for deletion, reads it again
(marking renames it to <path>-deletion_scheduled-<id>), and removes it
permanently under the new path; the instance finishes the removal
within seconds. Service accounts live on the top-level group, outside
the subgroup, so when a run has them, down destroys those resources
with a targeted terraform destroy first. A failed apply in up is
cleaned the same way before up returns.

The rig never touches production state, the configured group itself,
or anything outside a rig-* subgroup. The configured group is created
once by hand, as the production program group is.

<!-- section: preflight-and-configuration -->
## Preflight and configuration

Settings come from literal defaults in Go, overridden by the optional
file regression.md at the repository root, a config document (design
D11) that git ignores; regression.example.md is its tracked template.
It holds the group path, the instance URL, N, the sweep age, and whether service accounts
are on. An absent file is the defaults; a malformed one is a named
error. The group has no default: preflight refuses an empty one by
name rather than guessing.

Preflight checks, in order, that terraform is on PATH, that a token is
available (GITLAB_TOKEN, or the token glab holds for the instance),
and that the token can create subgroups in the configured group. Each
failure names the missing tool, credential, or permission. Nothing
skips silently: an operator who asked for a regression run gets one or
an error.

<!-- section: tiers -->
## Tiers

Service accounts are off by default, so the rig runs on a Free group.
A Premium or Ultimate group turns them on in regression.md to test
the agent identities. Group service accounts live on the top-level
group and are removed asynchronously, so down waits for their removal
and sweep collects any that remain.

<!-- section: unit-tests -->
## Unit tests

gitlabrig's own tests run without a network or a GitLab instance. A
recording runner captures every terraform and API call and answers
from a script keyed on the command, so the tests prove the sequences:
preflight refusals, up writing the outputs, down running after a
failing target, and sweep selecting only old rig-* subgroups. mage
audit runs these tests; mage regression:all is the only target that
reaches a real instance.

<!-- field: references -->
## References

- docs/specs/test-suites/test-rel01.0-objectives.md (first suite to run on the rig)
- infra/platform/README.md (the production root and its bootstrap)
- docs/constitutions/design.md D9 (why project paths are protected in production)
- Parent rig pattern: declarative-agents docs/engineering/eng01-kind-test-demo-rig.md and eng08-gcp-demo-rig.md
- GitLab Terraform provider, https://registry.terraform.io/providers/gitlabhq/gitlab/latest/docs
