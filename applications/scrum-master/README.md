<!-- Copyright (c) 2026 Gray Lake Labs -->
<!-- SPDX-License-Identifier: MIT-0 -->
# scrum-master

A technical program manager as a toolchain: `scrum-master`, a Go CLI that does Scrum's administrative work against a GitLab tracker, and the agent skills that drive it.

Most of what Scrum asks of people outside the work itself is moving facts the tracker already holds: standup updates, board changes, status reports, release bookkeeping. `scrum-master` reads and writes those facts directly — objectives, sprint ends, weekly reports, release gates and tags — so people spend their process time on judgment, not administration. The repository is specification-driven: the documentation corpus under `docs/` is the source of truth, and release gates hold the code to it.

## Layout

| Path | Role |
|---|---|
| `words/` | The words: every step a program can do, which the agent runs in its own process |
| `internal/` | Tracker, planning, reporting, and release logic |
| `plugin/` | Agent Plugins 1.0 package: the profiles in `plugin/agents/`, and the skills rendered from them, which the init and onboard profiles install |
| `docs/` | The specification corpus: SRDs, use cases, test suites, constitutions, guides |
| `infra/` | Terraform module and the regression-rig root |
| `magefiles/` | Build, gates, corpus audit, and integration targets |

## Install

Each release of this repository publishes the agent for Linux and macOS on amd64 and arm64, built with the scrum-master words, and the profiles archive — the profiles and the agent-core declarations they read — in one `SHA256SUMS`, on its [GitHub release](https://github.com/gray-lake-labs/declarative-agents/releases). Install the latest, or name a tag, with the script, which checks both against `SHA256SUMS`, installs the agent in `$HOME/.local/bin` (`AGENT_PREFIX` changes that), and unpacks the profiles in `$HOME/.local/share/scrum-master` (`SCRUM_MASTER_HOME` changes that):

```bash
curl -fsSL https://raw.githubusercontent.com/gray-lake-labs/declarative-agents/main/applications/scrum-master/install.sh | sh
```

No Go toolchain is needed: the agent runs the words, and the corpus gate's shape check, in its own process.

To install from a GitLab fork's releases, such as an internal one, set `SCRUM_MASTER_RELEASE_FORGE=gitlab` and `SCRUM_MASTER_RELEASE_URL` to the project's URL, with `GITLAB_TOKEN` for a private project; `docs/engineering/eng03-releasing-scrum-master.md` says how the fork publishes them.

## Start a program with the agent

A program repository runs its workflows with the agent: each installed skill runs `agent --profile .scrum-master/agents/<name>/profile.yaml --core-root .scrum-master/agent-core --directory .` over the profiles the repository vendored. The one workflow a person runs directly is `init` in a system repository, or `onboard` in a repository joining a program: it lays the repository down, installs the skills, and vendors the profiles and the agent-core declarations they read into `.scrum-master`, with a manifest naming the release. Commit `.scrum-master` with the rest; rerun the same profile to check the repository and to take a new release, which keeps every file a person edited and reports what drifted.

Run `init` from the installed profiles (`SCRUM_MASTER_SYSTEM` and `SCRUM_MASTER_SUBSYSTEM` name where an `onboard` joins):

```bash
home="${SCRUM_MASTER_HOME:-$HOME/.local/share/scrum-master}"
agent --profile "$home/agents/init/profile.yaml" --core-root "$home/agent-core" --directory .
```

A run's request file (`--request`, `{"parameters": {...}}`) names the agents whose skills install (`agent`, default `all`) and, for `init`, the tracker to seed (`tracker`, `group`, `prefix`).

## Run in CI

Each release also publishes an image, `ghcr.io/gray-lake-labs/scrum-master:<tag>`, for linux/amd64 and linux/arm64: the agent, built with the scrum-master words, and the release's profiles under `SCRUM_MASTER_HOME`, with git and git-crypt, no Go toolchain, running as a non-root user. Pin it by the digest the release's notes list. A CI job runs the repository's vendored check profile with the agent over its checkout: the corpus gate, the vendored profiles held to their manifest, and doctor's drift report, writing nothing. A GitLab CI job uses it directly:

```yaml
scrum-master-check:
  image: ghcr.io/gray-lake-labs/scrum-master@<digest>
  script:
    - agent --profile .scrum-master/agents/check/profile.yaml --core-root .scrum-master/agent-core --directory .
```

To pull from the program's own registry instead, copy it there once per release, for example with `docker buildx imagetools create --tag <registry>/scrum-master:<tag> ghcr.io/gray-lake-labs/scrum-master@<digest>`. Tracker credentials (`SCRUM_MASTER_TRACKER`, `SCRUM_MASTER_USER`) and any model credentials a profile names come from masked CI variables; `SCRUM_MASTER_USER` names a service user, never a person.

## Seed a program repository from an architecture document

A program's system repository can be generated from one document that describes the system the program builds: its purpose, layers, parts, and a build order. The init profile lays down the corpus, and a coding agent given the brief below writes the program's intent on top of it. Keep the seed in the repository (for example under `seed/`), so the program can be regenerated whenever the seed or the release changes.

The tracker repository comes first, so every step after it is recorded there. The init profile's output is the one commit made to `main` directly: the issue workflow reads the control files from `main`, so they must be there before the first issue is filed. The generated program then lands like any other change, as an issue that is filed, popped, approved by a second person, and merged.

The init profile writes thirteen files that need no further work, besides the vendored profiles in `.scrum-master`: `AGENTS.md`, `control/labels.md`, `control/policy.md`, `control/states.md`, `docs/SPECIFICATIONS.md`, `docs/constitutions/design.md`, `docs/corpus-charter.yaml`, `docs/external-pins.yaml`, `docs/road-map.md`, `docs/specs/software-requirements/srd001-system.md`, and a `README.md` in each of `docs/specs/themes/`, `initiatives/`, and `epics/`. It writes `docs/VISION.md` and `docs/ARCHITECTURE.md` as templates, and appends a managed block between `managed:begin` and `managed:end` markers to the repository's `README.md`. The agent writes the two templates and the planning hierarchy.

1. Create an empty tracker repository for the program (for example `<program>-tracker`). Then, in the program's repository, with the seed committed, lay down the corpus, seed the tracker, and commit the init profile's output:

   ```bash
   export SCRUM_MASTER_TRACKER=<tracker remote> SCRUM_MASTER_USER=<your name in the tracker>
   home="${SCRUM_MASTER_HOME:-$HOME/.local/share/scrum-master}"
   printf '{"parameters": {"tracker": "%s", "group": "<program group>", "agent": "<cursor, claude, copilot, or all>"}}' "$SCRUM_MASTER_TRACKER" > /tmp/init.json
   agent --profile "$home/agents/init/profile.yaml" --core-root "$home/agent-core" --directory . --request /tmp/init.json
   git add -A && git commit -m "Lay down the program" && git push origin main
   ```

   Without `agent`, the init profile installs the skills of the agents the repository already uses (a `.cursor` or `.claude` directory, `CLAUDE.md`, or copilot's instructions under `.github`), and refuses when it uses none. It refuses, writing nothing, when no release's profiles are installed. The init profile makes the tracker's first commit and creates the program's labels there. Without a tracker in its request it says that no tracker was seeded, and no issue can be filed until one is.

   The environment variables were named `TPM_` until GH-38. A `TPM_` name is still read where its `SCRUM_MASTER_` name is unset, and the check skill names each one a setup still sets, so a CI job can move at its own pace.

   A repository can use GitHub or GitLab instead of a tracker repository: name it `gh` or `glab` in the system repository's `control/trackers.md`, or run with `SCRUM_MASTER_TRACKER=gh` or `SCRUM_MASTER_TRACKER=glab`. scrum-master then drives that CLI with your existing login: `gh`'s token needs the `repo` and `read:org` scopes, and `glab`'s the `api` scope (`gh auth status` and `glab auth status` show them); the check skill reports a missing or logged-out CLI.

2. File the generation issue with the specify skill and pop it into its own branch and worktree with the pop skill. The issue names the documents it changes; a planning document among them makes it a `plane::management` issue. Its body:

   ```bash
   cat > /tmp/generate.md <<'EOF'
   ## Requirements

   1. Generate the program from <seed path>: docs/VISION.md, docs/ARCHITECTURE.md, and the planning hierarchy under docs/specs/.

   ## Acceptance Criteria

   - The hierarchy gate passes (test: scrum-master backlog check; environment: local)
   EOF
   agent --profile .scrum-master/agents/pop/profile.yaml --core-root .scrum-master/agent-core --directory .   # with SCRUM_MASTER_PROJECT and SCRUM_MASTER_ISSUE set
   ```

   The issue needs no `component::` label: the program's subsystem SRDs declare no component until the hierarchy and its SRDs exist. The pop workflow reports the worktree it opened, a sibling of the checkout. Steps 3 to 5 run there.

3. In that worktree, give a coding agent this brief, with the angle-bracket placeholders filled in:

   > You are generating the program for `<program name>` in this repository, which the init profile has just laid down. Read `<seed path>` (the system description), and use the backlog templates the draft skill shows for the document formats; `docs/constitutions/design.md` declares the fields each document may hold. Write only the files listed here and change nothing else.
   >
   > **Files to write.**
   > 1. `docs/VISION.md`: fill every field the init profile laid down (executive summary, problem, what this does, why we build this, success criteria as named `key: outcome` items, not).
   > 2. `docs/ARCHITECTURE.md`: fill every field the init profile laid down. The overview has a summary and a layer diagram as an inline Mermaid block, marking each part built, partly built, or deferred in this program. Interfaces are `interface:` nodes for each interface the program owns. Components are `component:` nodes, one per layer and per service, each with capabilities and ownership, plus one for the deferred parts. Design decisions are numbered `decision:` nodes, each saying what it rejected. Also fill technology choices, project structure, implementation status, and related documents. Draw the delivery chain, if the program has one, as a second Mermaid block.
   > 3. The planning hierarchy under `docs/specs/`: one theme (`themes/th001-<name>.md`), one initiative (`initiatives/in001-<name>.md`), and the epics (`epics/ep001-<name>.md` onward).
   >
   > **Program parameters (these are not in the seed).**
   > - Platform and hosting: `<cloud and delivery tools; the interfaces the program owns>`.
   > - Platforms to integrate rather than rebuild: `<existing systems, and the capabilities each already provides>`. Wherever a capability might duplicate one, a requirement item says that platform's coverage is confirmed first.
   > - Teams that own epics: `<team names>`. An epic's owner is `Product Owner, <team>`; the theme's and initiative's owner is `technical program manager`.
   > - Cadence: `<sprint length and alignment, weeks off>`. The theme's horizon is `<ISO week range>`. The initiative's horizon is `<ISO week range>`, and it has one requirement group per sprint, due on `<each sprint's due date>`.
   > - Sprint milestones, following the seed's build order: `<one observable milestone per sprint>`.
   > - Theme: `<the strategic area>`. Its requirement groups follow `<the seed's principles or practices>`.
   > - Epics, platform setup first: `<one epic per major part of the system, each with its owning team>`. Each epic has a horizon longer than one sprint, `depends_on` links to the epics it needs, and acceptance criteria that name a test and an environment (`local`, `dev`, or `staging`).
   > - Deferred: `<the parts of the system this program does not build>`, and the documents say so.
   >
   > **Rules.** Every hierarchy document has status `to_be_agreed`. Every acceptance criterion traces only to items in its own document. Leave `delivered_by` out until the teams' requirement documents exist. Diagrams are inline Mermaid, with no image files. Write no personal names and no links to private repositories. When the files are written, run `scrum-master backlog check` and fix every finding.

4. Run the check in the worktree, and fix every finding until it passes:

   ```bash
   agent --profile .scrum-master/agents/check/profile.yaml --core-root .scrum-master/agent-core --directory .
   ```

5. Commit the generated program and push its branch. The merge runs the hierarchy gate as its audit, since a program repository has no `mage audit`. When the program's policy sets `require_approval`, someone other than the issue's assignee first approves the exact commit:

   ```bash
   git add -A && git commit -m "Generate the program from the seed" && git push -u origin HEAD
   SCRUM_MASTER_USER=<reviewer> agent --profile .scrum-master/agents/approve/profile.yaml --core-root .scrum-master/agent-core --directory .   # the reviewer, under require_approval
   agent --profile .scrum-master/agents/merge/profile.yaml --core-root .scrum-master/agent-core --directory .
   ```

   The merge pushes to `main` and closes the issue.

6. Tag the merge, so later runs and regenerations can be compared against it. In the program's checkout:

   ```bash
   git pull --ff-only origin main
   git tag -a "baseline/$(date -u +%Y%m%dT%H%M%SZ)" -m "Program generated from the seed"
   ```

Regenerating from a changed seed is the same loop from step 2: one issue, one branch, one merge.

Every hierarchy document starts as `to_be_agreed`; agreeing them is the program's first planning decision, and the plan skill reads only agreed documents. To add or rework one document at a time afterwards, use the `draft` skill, which drafts any level from a description in conversation with its owner.

## Build and test

```
go test ./...
mage -d magefiles -l
```

The agent this repository runs is agent-core's, built with the words (`mage -d magefiles integration:popProfile` builds it into the release cache); the rig drives the words through its harness, `magefiles/words`.

Releases are cut by `mage -d magefiles tag`, which runs the gates in `release-gates.md`, tags `v0.YYYYMMDD.N`, and once the tag is pushed publishes the agent and the profiles with `mage -d magefiles publish` (`docs/engineering/eng03-releasing-scrum-master.md`). The corpus audit requires a clone cache of the parent specification repository; `docs/guides/corpus-audit.md` describes it.

Issue tracking lives in the repository via [beads](https://github.com/steveyegge/beads); the GitHub tracker is unused.

## License

[MIT No Attribution (MIT-0)](LICENSE).
