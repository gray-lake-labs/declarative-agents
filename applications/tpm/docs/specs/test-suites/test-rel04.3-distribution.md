---
id: test-rel04.3-distribution
title: Test Suite for Release 04.3 tpm Ships to Machines and to CI
release: "04.3"
status: implemented
---
<!-- Copyright (c) 2026 Gray Lake Labs -->
<!-- SPDX-License-Identifier: MIT-0 -->

<!-- field: overview -->
## Overview

The Go tests run offline against a local release server that serves a
fixture release: binaries, SHA256SUMS, and a source archive. The
integration targets build tpm and run it against the real release tpm
pins, so they need network access to github.com, as mage audit does;
mage integration:image also needs docker. Together they check that the
runtime is fetched and verified once per tag into the cache mage audit
uses, that TPM_AGENT is held to the release's checksums, that tpm doctor
reports the runtime and repairs with a prebuilt spec tool when Go is
absent, and that the image carries tpm, the runtime, and the spec tool at
the pin.

<!-- field: traces -->
## Traces

- [rel04.3-uc001-install-without-go](../use-cases/rel04.3-uc001-install-without-go.md)
- [rel04.3-uc002-ci-job-on-the-image](../use-cases/rel04.3-uc002-ci-job-on-the-image.md)

<!-- field: preconditions -->
## Preconditions

- A tpm binary built from this repository, so its pins file is embedded.
- For the Go tests: a temporary cache directory and a local release server; no network.
- For the integration targets: network access to github.com; docker with buildx for mage integration:image.

<!-- field: test_cases -->
## Test cases

<!-- case: the runtime is fetched once into the shared cache | status: implemented -->
### the runtime is fetched once into the shared cache

<!-- field: use_case -->
#### Use case

[rel04.3-uc001-install-without-go](../use-cases/rel04.3-uc001-install-without-go.md)

<!-- field: integration_target -->
#### Integration target

mage integration:runtime

<!-- field: go_test -->
#### Go test

go test ./internal/runtime -run '^TestFetchLaysOut|^TestFetchDryRun|^TestBuildSpecTool|^TestPin'

<!-- field: expected_outputs | form: "tree" -->
#### Expected outputs

- fetched: the pinned release's agent for the host and the release's source in scrum-master/release/<tag>/ under the user cache directory
- shared: mage audit resolves the same agent and source paths
- second_run: nothing downloaded
- spec_tool: built when Go is on PATH; reported not built, with the reason, otherwise
- report: release, agent path and source, source root, and spec tool in --json
- dry_run: the downloads named and nothing written

<!-- field: traces -->
#### Traces

- [srd037-install](../software-requirements/srd037-install.md) [AC7](../software-requirements/srd037-install.md#ac7)

<!-- case: the runtime is held to the release's checksums | status: implemented -->
### the runtime is held to the release's checksums

<!-- field: use_case -->
#### Use case

[rel04.3-uc001-install-without-go](../use-cases/rel04.3-uc001-install-without-go.md)

<!-- field: integration_target -->
#### Integration target

mage integration:runtime

<!-- field: go_test -->
#### Go test

go test ./internal/runtime -run '^TestReleaseBinary|^TestHostAsset|^TestFetchReplaces|^TestFetchRefuses|^TestTPMAgent|^TestInspect'

<!-- field: expected_outputs | form: "tree" -->
#### Expected outputs

- tampered_cache: a cached binary that no longer matches is downloaded again
- tampered_download: refused naming the asset and both checksums; nothing written executable
- platform: an unpublished platform refused naming darwin and linux on amd64 and arm64
- tpm_agent_match: TPM_AGENT used when its SHA-256 is the release's for the host
- tpm_agent_mismatch: any other binary refused naming the pinned release
- offline: TPM_AGENT with no cached or reachable checksums refused naming them

<!-- field: traces -->
#### Traces

- [srd037-install](../software-requirements/srd037-install.md) [AC7](../software-requirements/srd037-install.md#ac7)

<!-- case: doctor reports the runtime and repairs without Go | status: implemented -->
### doctor reports the runtime and repairs without Go

<!-- field: use_case -->
#### Use case

[rel04.3-uc001-install-without-go](../use-cases/rel04.3-uc001-install-without-go.md)

<!-- field: integration_target -->
#### Integration target

mage integration:doctor

<!-- field: go_test -->
#### Go test

go test ./internal/scaffold -run '^TestDoctorRuntime|^TestRepairPrebuilt'

<!-- field: expected_outputs | form: "tree" -->
#### Expected outputs

- missing: no runtime named with tpm install --runtime as the fix
- found: the cached or TPM_AGENT runtime named with its path when the checksums match
- refused: a runtime the checksums refuse reported, with the fix
- no_fetch: doctor downloads nothing, on a run or a dry run
- prebuilt: with no Go on PATH and a spec tool built at the pin, a document repaired to canonical form
- neither: with no Go and no built tool, the repair skipped and the reason named

<!-- field: traces -->
#### Traces

- [srd043-init-onboard](../software-requirements/srd043-init-onboard.md) [AC8](../software-requirements/srd043-init-onboard.md#ac8)

<!-- case: the image carries tpm and the runtime at the pin | status: implemented -->
### the image carries tpm and the runtime at the pin

<!-- field: use_case -->
#### Use case

[rel04.3-uc002-ci-job-on-the-image](../use-cases/rel04.3-uc002-ci-job-on-the-image.md)

<!-- field: integration_target -->
#### Integration target

mage integration:image

<!-- field: go_test -->
#### Go test

go test ./internal/release -run '^TestImage|^TestPublishPushes|^TestPublishKeepsAPushed'

<!-- field: expected_outputs | form: "tree" -->
#### Expected outputs

- versions: tpm --version in the image names the build, and the agent's SHA-256 is the pinned release's for linux on the image's architecture
- no_go: the image carries no Go toolchain and no mage
- doctor: tpm doctor --dry-run in a corpus at tpm's pin reports the runtime and the repair, with no Go
- backlog: tpm backlog check runs over a mounted checkout as the image's non-root user
- validate: agent --validate-config runs against the image's source root

<!-- field: traces -->
#### Traces

- [srd037-install](../software-requirements/srd037-install.md) [AC7](../software-requirements/srd037-install.md#ac7)
- [srd043-init-onboard](../software-requirements/srd043-init-onboard.md) [AC8](../software-requirements/srd043-init-onboard.md#ac8)
