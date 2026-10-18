---
id: eng03-releasing-scrum-master
title: "Releasing scrum-master: Gates, Then a Date Tag"
---

<!-- field: introduction -->
## Introduction

A scrum-master tag is what a developer installs tpm from (go install
github.com/gray-lake-labs/scrum-master/cmd/tpm@<tag>), and tpm install stamps
that version into every skill it writes (srd037). A tag that nothing
checked would put an unverified tpm and its skills into every repository
that installs it. This guideline states how we cut a scrum-master release:
the gates file, the sequence mage tag runs, and why the release depends on
nothing outside this repository. A program repository's release is a
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
| test | go test ./... and mage integration:install |

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
| Preflight | The release runs on main with a clean worktree; a HEAD already carrying a v0.* tag has nothing to release |
| Gates | Stages run in order and the gates of a stage at the same time, each line of output prefixed with the gate's name; the first stage with a failed gate stops the release and names every failed gate |
| Recheck | HEAD and the worktree must be unchanged after the gates; a gate that commits or edits files voids the run |
| Tag | v0.YYYYMMDD.N, N one past the highest revision for the day across local tags and origin's; the annotation lists the gates that passed and git log --oneline since the previous release |
| Push | push auto pushes the tag to origin; push ask asks y/N and otherwise prints the push command |

Table: What mage tag does.

The /gh-release-push skill calls mage tag on main. A failed gate is
never fixed inside the release: we file an issue, merge the fix, and
release the green main.

<!-- section: versions -->
## Versions

tpm takes its version from Go's version control detection: a build of
./cmd/tpm from a clean checkout at a tag carries that tag, a build with
uncommitted changes carries it with +dirty, and a build between tags
carries a pseudo-version. tpm install stamps that version into every
skill it writes. Nothing passes the version in by hand.

Detection reads go.mod from the git objects at the tagged commit. The
repository is encrypted with git-crypt, and an encrypted go.mod makes
every build report (devel), so .gitattributes keeps go.mod in
plaintext and a repository gate test (TestGoModIsStoredInPlaintext)
fails when it is stored encrypted. Every other file stays encrypted,
so go install from GitHub at a tag does not work; a developer builds
tpm from an unlocked checkout.

<!-- section: no-dependency-outside-the-repository -->
## No dependency outside the repository

The declarative-agents ecosystem has a shared release runner and a
release-manager agent, but only in a private repository that the
environment scrum-master deploys to cannot reach. scrum-master therefore
implements the same contract in internal/release, in plain Go, with
no agent: every step is a rule, and design.md D10 puts rules in code.
The audit agent still comes from the public Nokia-Bell-Labs
declarative-agents tag the corpus pins (docs/guides/corpus-audit.md).

internal/release holds the gates file, the stage runner, and the
tagging (NextDateTag, RemoteTags, Release.Tag), each behind a git
function so tests run against real temporary repositories with a bare
origin. magefiles/tag.go only wires the repository root to it, and
tpm release (srd015) reuses the tagging for program repositories.

<!-- field: references -->
## References

- release-gates.md
- .gitattributes (go.mod in plaintext)
- internal/release
- docs/specs/software-requirements/srd015-release.md (a program repository's release)
- docs/specs/software-requirements/srd037-install.md (the version tpm install stamps)
- docs/constitutions/design.md D10
