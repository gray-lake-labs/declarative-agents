---
id: eng03-releasing-scrum-master
title: "Releasing scrum-master: Gates, Then a Date Tag"
---
<!-- Copyright (c) 2026 Gray Lake Labs -->
<!-- SPDX-License-Identifier: MIT-0 -->

<!-- field: introduction -->
## Introduction

A scrum-master tag is what tpm is installed from: a developer downloads
the tag's release binary or runs go install
github.com/gray-lake-labs/scrum-master/cmd/tpm@<tag>, and a CI job runs on
the tag's container image. tpm install stamps that version into every
skill it writes (srd037). A tag that nothing checked would put an
unverified tpm and its skills into every repository that installs it.
This guideline states how we cut a scrum-master release: the gates file,
the sequence mage tag runs, the binaries and the image it publishes, and
why the release depends on nothing outside this repository. A program repository's release is a
different process (srd015, tpm release) and shares only the tagging code.

<!-- section: the-gates-file -->
## The gates file

release-gates.md at the repository root declares what a release must
pass. It is a config document (design D11): push is a field, ask or auto,
and stages holds one stage record per stage, in order, whose gates field
is a table of name, command, and an optional directory relative to the
root.

| Stage | Gates |
|---|---|
| audit | mage audit: the repository gate tests, then the specification-critic corpus audit |
| test | go test ./..., mage integration:install, and mage crossBuild, which builds tpm for every release platform |

Table: scrum-master's release gates.

mage regression:all is not a gate. It needs the GitLab trial group and a
token, takes minutes, and depends on gitlab.com's availability, so a
release would fail for reasons unrelated to the commit. We run it
before merging changes that touch the tracker, not at release time.
Push is ask: the person cutting the release sees the tag and its
annotation before it leaves the machine.

<!-- section: the-sequence -->
## The sequence

mage tag runs these steps, and mage tagDryRun runs the first four and
creates nothing.

| Step | What happens |
|---|---|
| Lock | A directory under the git dir (scrum-master-release.lock) holds the release, so two releases of one clone cannot overlap |
| Preflight | The release runs on main with a clean worktree; a HEAD already carrying a v0.* tag has nothing to release; gh answers as the organization account and docker buildx is on PATH, so a tag never lands without the means to publish it |
| Gates | Stages run in order and the gates of a stage at the same time, each line of output prefixed with the gate's name; the first stage with a failed gate stops the release and names every failed gate |
| Recheck | HEAD and the worktree must be unchanged after the gates; a gate that commits or edits files voids the run |
| Tag | v0.YYYYMMDD.N, N one past the highest revision for the day across local tags and origin's; the annotation lists the gates that passed and git log --oneline since the previous release |
| Push | push auto pushes the tag to origin; push ask asks y/N and otherwise prints the push command |
| Publish | Once the tag is on origin, mage publish runs: the release binaries, then the container image; a tag left unpushed prints mage publish as the step to run after pushing it |

Table: What mage tag does.

mage publish publishes the tag at HEAD and nothing else. Each part it
finds already published (an asset on the GitHub release, the image at
the tag in the registry) is checked and kept, so rerunning it after a
failed upload or push completes the release rather than repeating it.

The /gh-release-push skill calls mage tag on main. A failed gate is
never fixed inside the release: we file an issue, merge the fix, and
release the green main.

<!-- section: release-binaries -->
## Release binaries

mage publishDryRun builds the binaries and names what mage publish would
upload and push, creating nothing. mage integration:image builds the
image for this machine's architecture and checks it without pushing. The first part of mage publish builds ./cmd/tpm at the tag with
CGO_ENABLED=0 for linux, darwin, and windows on amd64 and arm64, writes
SHA256SUMS over them, creates the tag's GitHub release, and uploads the
six binaries and SHA256SUMS to it. The assets are named tpm_<os>_<arch>,
with .exe on windows, the convention the declarative-agents releases
follow, so one naming rule and one checksum check serve both.
install.sh at the repository root downloads the asset for the machine
from a named release, or the latest, verifies it against SHA256SUMS, and
installs it; the README gives that, a manual download, and go install,
each followed by tpm install --runtime (srd037 R7).

The release is created with the organization's token
(GH_TOKEN=$(gh auth token --user gray-lake-labs-owner)), never a
person's account, so nothing on the release names an individual.

<!-- section: the-container-image -->
## The container image

The second part builds one image for linux/amd64 and linux/arm64 with
docker buildx from the Dockerfile at the repository root and pushes it as
ghcr.io/gray-lake-labs/tpm:<tag>, and moves latest to it. Its build
context is the directory the first part built the binaries into, and it
copies tpm_linux_<arch> from there, so the image's tpm is the binary on
the GitHub release; Dockerfile.dockerignore admits nothing else. A build
stage with Go runs tpm install --runtime, and the image keeps only the
cache it filled. The image holds
tpm at the tag; the declarative-agents runtime at the release tpm pins,
in the user cache directory tpm install --runtime fills (srd037 R7),
with agent on PATH; the spec tool built at that release, so tpm doctor
repairs a corpus at tpm's pin with no Go toolchain (srd043 R4.5); git,
git-crypt, and CA certificates. It carries no Go toolchain and no mage,
runs as a non-root user, and works in the directory a CI job mounts its
checkout into. A job on the image therefore never runs a profile against
a runtime it was not written for.

The push logs in to ghcr.io as the organization account with its gh
token, which needs the write:packages scope. buildx attaches an SBOM
attestation to the image index, and its provenance is recorded at
mode=min so it names no machine or person. The index carries the source,
description, and license annotations, which link the package to this
repository on ghcr; the Dockerfile's labels repeat them on each
platform's image. The image is published after
the tag is pushed, so its digest cannot be in the tag annotation; mage
publish adds it to the GitHub release's notes, which pipelines pin from;
latest is for people. An image already at the tag is kept.

The image is not signed. Pinning by digest already holds a job to the
image that was published, and the digest's trust is the GitHub release,
the same trust the release binaries and their SHA256SUMS rest on. A
signature adds only a key held apart from the GitHub account, which one
maintainer holding both does not give, and no consumer verifies one yet.
Signing from a person's machine would also need a key pair to keep
secret, since keyless signing there records that person's identity in a
public transparency log. Signing returns when a consumer requires it,
keyless from a CI identity once releases publish from a pipeline.

A CI job reads tracker and model credentials from masked CI variables
(TPM_TRACKER, TPM_USER, and what each profile names), and its writes are
attributed to the service user TPM_USER names, never to a person; srd041's
approval rule still holds, so CI never approves its own merge. The README
shows a GitLab CI job on the image, pulling from ghcr or from a program's
own registry mirror.

<!-- section: versions -->
## Versions

tpm takes its version from Go's version control detection: a build of
./cmd/tpm from a clean checkout at a tag carries that tag, a build with
uncommitted changes carries it with +dirty, and a build between tags
carries a pseudo-version. tpm install stamps that version into every
skill it writes. Nothing passes the version in by hand.

Detection reads go.mod from the git objects at the tagged commit, so a
go.mod stored in any other form makes every build report (devel). The
repository is not encrypted, so go install from GitHub at a tag builds
tpm at that tag, and the release binaries carry the same version. A
repository gate test (TestGoModIsStoredInPlaintext) still fails when
go.mod is stored encrypted, so a return to encryption cannot silently
break the version.

<!-- section: no-dependency-outside-the-repository -->
## No dependency outside the repository

The declarative-agents ecosystem has a shared release runner and a
release-manager agent, but only in a private repository that the
environment scrum-master deploys to cannot reach. scrum-master therefore
implements the same contract in internal/release, in plain Go, with
no agent: every step is a rule, and design.md D10 puts rules in code.
The audit agent still comes from the public gray-lake-labs
declarative-agents release the corpus pins (docs/guides/corpus-audit.md).

internal/release holds the gates file, the stage runner, and the
tagging (NextDateTag, RemoteTags, Release.Tag), each behind a git
function so tests run against real temporary repositories with a bare
origin. magefiles/tag.go only wires the repository root to it, and
tpm release (srd015) reuses the tagging for program repositories.

<!-- field: references -->
## References

- release-gates.md
- Dockerfile
- install.sh
- internal/release
- docs/specs/software-requirements/srd015-release.md (a program repository's release)
- docs/specs/software-requirements/srd037-install.md (the version tpm install stamps)
- docs/constitutions/design.md D10
