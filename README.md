# scrum-master

A technical program manager as a toolchain: `tpm`, a Go CLI that does Scrum's administrative work against a GitLab tracker, and the agent skills that drive it.

Most of what Scrum asks of people outside the work itself is moving facts the tracker already holds: standup updates, board changes, status reports, release bookkeeping. `tpm` reads and writes those facts directly — objectives, sprint ends, weekly reports, release gates and tags — so people spend their process time on judgment, not administration. The repository is specification-driven: the documentation corpus under `docs/` is the source of truth, and release gates hold the code to it.

## Layout

| Path | Role |
|---|---|
| `cmd/tpm` | The CLI |
| `internal/` | Tracker, planning, reporting, and release logic |
| `plugin/` | Agent Plugins 1.0 package: the process skills `tpm install` writes |
| `docs/` | The specification corpus: SRDs, use cases, test suites, constitutions, guides |
| `infra/` | Terraform module and the regression-rig root |
| `magefiles/` | Build, gates, corpus audit, and integration targets |

## Build and test

```
go build ./cmd/tpm
go test ./...
mage -d magefiles -l
```

Releases are cut by `mage -d magefiles tag`, which runs the gates in `release-gates.md` and tags `v0.YYYYMMDD.N`. The corpus audit requires a clone cache of the parent specification repository; `docs/guides/corpus-audit.md` describes it.

Issue tracking lives in the repository via [beads](https://github.com/steveyegge/beads); the GitHub tracker is unused.
