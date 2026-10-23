# Corpus Audit

We audit this corpus with the specification-corpus audit that ships with
the parent, gray-lake-labs/declarative-agents, at the release
`docs/external-pins.yaml` pins. The audit is the parent's
`specification-critic` corpus-audit profile run over this repository:
the profile binds `docs/corpus-charter.yaml` relative to the directory it
audits, so the charter here decides which checks gate this corpus.

## Where things live

| What | Where |
|---|---|
| Checks that gate this corpus | `docs/corpus-charter.yaml` |
| Tracked parent release | `docs/external-pins.yaml` |
| The release's agent and source, and the spec tool built from it | the user cache directory, `scrum-master/release/<tag>`, fetched on first use |
| Parent requirements this corpus derives from | `derived_from` entries in the SRDs |

The parent publishes each tag's `agent` for darwin and linux on amd64 and
arm64, with `SHA256SUMS`. `mage audit` downloads the pinned tag's agent for
the host, refuses it if its checksum does not match, and keeps it in the
cache. It also lays out the tag's source archive there as a git repository
tagged with the release, which is the clone the audit resolves `derived_from`
entries against. A host the parent publishes no binary for fails with an
error that names it.

The spec tool behind `mage fmt`, `mage convert`, and `mage specCheck` is this
repository's own (`magefiles/spec.go`): it lists the tracked documents, reads
initiatives as SRDs, and formats each with the parent's Markdown
library. It is built once per tag against that library in the downloaded
source.

## Running the audit

The entry point is `mage audit`. It runs the repository gate tests
(`go test ./magefiles/...`: gofmt, the 50-line skill limit, the
hierarchy gate, the help pages, and the rest), then runs the pinned
release's agent over the repository root. For that run it writes
`docs/external-checkouts.yaml`, pointing the parent at the cached source, and
removes it afterwards. The file is ignored by git; a checkouts file a person
wrote is left as it is and used instead.

The run exits zero and ends in the `Passed` state when no check in the
charter reports an error. A malformed SRD (an acceptance criterion tracing a
requirement item that does not exist, for example) fails at load with exit
2. After the first download the audit is offline: every `derived_from`
entry resolves through local git calls against the cached source.

The parent audit reads structured citations (traces, touchpoints,
`derived_from`). Citations written inside requirement prose, such as
"(srd010 R5)", are not checked by it; keep them resolvable by hand or with a
local check until the parent covers them.

## When the release is missing or a new one ships

Nothing is built or cloned by hand. Deleting `scrum-master/release/<tag>`
from the user cache makes the next `mage audit` download it again. Moving the
pin to a new release is the parent-upgrade workflow
(docs/guides/parent-upgrade.md); the new release downloads on first use.
