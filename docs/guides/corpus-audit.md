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
| The release's agent, spec tool, and source | the user cache directory, `scrum-master/release/<tag>`, downloaded on first use |
| Parent requirements this corpus derives from | `derived_from` entries in the SRDs |

The parent publishes each tag's `agent` and `spectool` for darwin and linux
on amd64 and arm64, with `SHA256SUMS`. `mage audit` and `mage fmt` download
the pinned tag's binaries for the host, refuse any whose checksum does not
match, and keep them in the cache. They also lay out the tag's source archive
there as a git repository tagged with the release, which is the clone the
audit resolves `derived_from` entries against. A host the parent publishes no
binary for fails with an error that names it.

## Running the audit

The entry point is `mage audit`. It runs the repository gate tests
(`go test ./magefiles/...`: gofmt, the 50-line skill limit, the
objective-document gate, the help pages, and the rest), then runs the pinned
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
