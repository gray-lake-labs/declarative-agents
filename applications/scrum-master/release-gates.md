---
id: release-gates
title: Release gates (eng03)
---
<!-- Copyright (c) 2026 Gray Lake Labs -->
<!-- SPDX-License-Identifier: MIT-0 -->

<!-- field: push -->
## Push

ask

<!-- field: stages -->
## Stages

<!-- stage: audit -->
### audit

<!-- field: gates -->
#### Gates

| name | command |
| --- | --- |
| audit | mage audit |

<!-- stage: test -->
### test

<!-- field: gates -->
#### Gates

| name | command |
| --- | --- |
| unit tests | go test ./... |
| install integration | mage integration:install |
| cross build | mage crossBuild |
