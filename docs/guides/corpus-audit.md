# Corpus Audit

We audit this corpus with the specification-corpus audit that ships with
the parent, Nokia-Bell-Labs/declarative-agents, at the release
`docs/external-pins.yaml` pins. The audit is the parent's
`specification-critic` corpus-audit profile run over this repository:
the profile binds `docs/corpus-charter.yaml` relative to the directory it
audits, so the charter here decides which checks gate this corpus. The
repository holds no Go; the tooling comes from the pinned release.

## Where things live

| What | Where |
|---|---|
| Checks that gate this corpus | `docs/corpus-charter.yaml` |
| Tracked parent release | `docs/external-pins.yaml` |
| Parent clone cache | `../declarative-agents-nokia`, named in `docs/external-checkouts.yaml` |
| Parent requirements this corpus derives from | `derived_from` entries in the SRDs |

The cache is a clone holding only Nokia's `main` and the release tags
reachable from it, with Nokia as its `origin`. The working clone at
`../declarative-agents` carries a fork's tags as well,
and fork and parent use the same `v0.YYYYMMDD.N` scheme, so a shared tag
namespace would make "latest release" follow the fork.

## Running the audit

The entry point is `mage audit`, as in every declarative-agents module. It
runs the repository gate tests (`go test ./magefiles/...`: gofmt, the
50-line skill limit, and the objective-document gate), then builds the agent
from the pinned release once per tag under the user cache directory and runs
the profile against the repository root. The manual steps below are what
`mage audit` does, for a machine without mage: build the agent from the
pinned release outside the repository, then run the profile against the
repository root.

```bash
REL=$(yq '.pins["github.com/Nokia-Bell-Labs/declarative-agents"]' docs/external-pins.yaml)
WORK=$(mktemp -d)
git -C ../declarative-agents-nokia archive "$REL" | tar -x -C "$WORK"
(cd "$WORK/agent-core" && go build -tags production -o "$WORK/agent" ./cmd/agent)
"$WORK/agent" \
  --profile "$WORK/applications/catalog/agents/specification-critic/corpus-audit-profile.yaml" \
  --directory "$PWD" \
  --core-root "$WORK/agent-core" \
  --output "$WORK/out"
```

The run exits zero and ends in the `Passed` state when no check in the
charter reports an error. A malformed SRD (an acceptance criterion tracing a
requirement item that does not exist, for example) fails at load with exit
2. The audit is offline: every `derived_from` entry resolves through local
git calls against the cache, never the network. Building the agent may
download Go modules the first time the module cache is cold; the audit run
itself does not.

The parent audit reads structured citations (traces, touchpoints,
`derived_from`). Citations written inside requirement prose, such as
"(srd010 R5)", are not checked by it; keep them resolvable by hand or with a
local check until the parent covers them.

## When the cache is missing or behind

If the cache does not exist, build it from the working clone without
touching the network:

```bash
git init ../declarative-agents-nokia
cd ../declarative-agents-nokia
git remote add origin https://github.com/Nokia-Bell-Labs/declarative-agents.git
git fetch ../declarative-agents '+refs/remotes/nokia/main:refs/remotes/origin/main'
git update-ref refs/heads/main refs/remotes/origin/main
```

Fetching `main` brings every tag reachable from it and nothing else. When a
new parent release ships, `git -C ../declarative-agents-nokia fetch --tags
origin` brings it in; moving the pin is the upgrade workflow.
