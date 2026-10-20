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

## Seed a program repository from an architecture document

A program's system repository can be generated from one document that describes the system the program builds: its purpose, layers, parts, and a build order. `tpm init` lays down the corpus, and a coding agent given the brief below writes the program's intent on top of it. Keep the seed in the repository (for example under `seed/`), so the program can be regenerated whenever the seed or tpm changes.

The tracker repository comes first, so every step after it is recorded there. `tpm init`'s output is the one commit made to `main` directly: the issue workflow reads the control files from `main`, so they must be there before the first issue is filed. The generated program then lands like any other change, as an issue that is filed, popped, approved by a second person, and merged.

`tpm init` writes twelve files that need no further work: `AGENTS.md`, `control/labels.md`, `control/policy.md`, `docs/SPECIFICATIONS.md`, `docs/constitutions/design.md`, `docs/corpus-charter.yaml`, `docs/external-pins.yaml`, `docs/road-map.md`, `docs/specs/software-requirements/srd001-system.md`, and a `README.md` in each of `docs/specs/themes/`, `initiatives/`, and `epics/`. It writes `docs/VISION.md` and `docs/ARCHITECTURE.md` as templates, and appends a managed block between `tpm:begin` and `tpm:end` markers to the repository's `README.md`. The agent writes the two templates and the planning hierarchy.

1. Create an empty tracker repository for the program (for example `<program>-tracker`). Then, in the program's repository, with the seed committed, lay down the corpus, seed the tracker, and commit `tpm init`'s output:

   ```bash
   (cd <scrum-master checkout> && go build -o /tmp/tpm ./cmd/tpm)
   export TPM_TRACKER=<tracker remote> TPM_USER=<your name in the tracker>
   /tmp/tpm init --dir . --tracker "$TPM_TRACKER" --group <program group>
   git add -A && git commit -m "tpm init: lay down the program" && git push origin main
   ```

   `tpm init` makes the tracker's first commit and creates the program's labels there. Without `--tracker` it says that no tracker was seeded, and no issue can be filed until one is.

2. File the generation issue and pop it into its own branch and worktree. Name the documents it changes with `--path`; a planning document among them makes it a `plane::management` issue:

   ```bash
   cat > /tmp/generate.md <<'EOF'
   ## Requirements

   1. Generate the program from <seed path>: docs/VISION.md, docs/ARCHITECTURE.md, and the planning hierarchy under docs/specs/.

   ## Acceptance Criteria

   - The hierarchy gate passes (test: tpm backlog check; environment: local)
   EOF
   /tmp/tpm issue file --project <program path> --system <program path> --title "Generate the program from the seed" \
     --body-file /tmp/generate.md --path docs/VISION.md --path docs/ARCHITECTURE.md --path docs/specs/themes/th001-<name>.md
   /tmp/tpm issue pop --project <program path> --issue <issue number> --dir . --control <program path>
   ```

   The issue needs no `component::` label: the program's subsystem SRDs declare no component until the hierarchy and its SRDs exist. `tpm issue pop` prints the worktree it opened, a sibling of the checkout. Steps 3 to 5 run there.

3. In that worktree, give a coding agent this brief, with the angle-bracket placeholders filled in:

   > You are generating the program for `<program name>` in this repository, which `tpm init` has just laid down. Read `<seed path>` (the system description) and run `tpm help backlog` for the document formats; `docs/constitutions/design.md` declares the fields each document may hold. Write only the files listed here and change nothing else.
   >
   > **Files to write.**
   > 1. `docs/VISION.md`: fill every field `tpm init` laid down (executive summary, problem, what this does, why we build this, success criteria as named `key: outcome` items, not).
   > 2. `docs/ARCHITECTURE.md`: fill every field `tpm init` laid down. The overview has a summary and a layer diagram as an inline Mermaid block, marking each part built, partly built, or deferred in this program. Interfaces are `interface:` nodes for each interface the program owns. Components are `component:` nodes, one per layer and per service, each with capabilities and ownership, plus one for the deferred parts. Design decisions are numbered `decision:` nodes, each saying what it rejected. Also fill technology choices, project structure, implementation status, and related documents. Draw the delivery chain, if the program has one, as a second Mermaid block.
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
   > **Rules.** Every hierarchy document has status `to_be_agreed`. Every acceptance criterion traces only to items in its own document. Leave `delivered_by` out until the teams' requirement documents exist. Diagrams are inline Mermaid, with no image files. Write no personal names and no links to private repositories. When the files are written, run `tpm backlog check` and fix every finding.

4. Run the hierarchy gate in the worktree, and fix every finding until it passes:

   ```bash
   /tmp/tpm backlog check
   ```

5. Commit the generated program and push its branch. Someone other than the issue's assignee approves the exact commit, and the merge runs the hierarchy gate as its audit, since a program repository has no `mage audit`:

   ```bash
   git add -A && git commit -m "Generate the program from the seed" && git push -u origin HEAD
   TPM_USER=<reviewer> /tmp/tpm issue approve --project <program path> --issue <issue number> --dir .   # the reviewer runs this
   /tmp/tpm issue merge --project <program path> --issue <issue number> --dir . --audit "/tmp/tpm backlog check"
   ```

   The merge pushes to `main` and closes the issue.

6. Tag the merge, so later runs and regenerations can be compared against it. In the program's checkout:

   ```bash
   git pull --ff-only origin main
   git tag -a "baseline/$(date -u +%Y%m%dT%H%M%SZ)" -m "Program generated from the seed"
   ```

Regenerating from a changed seed is the same loop from step 2: one issue, one branch, one merge.

Every hierarchy document starts as `to_be_agreed`; agreeing them is the program's first planning decision, and `tpm plan` reads only agreed documents. To add or rework one document at a time afterwards, use the `draft` skill, which drafts any level from a description in conversation with its owner.

## Build and test

```
go build ./cmd/tpm
go test ./...
mage -d magefiles -l
```

Releases are cut by `mage -d magefiles tag`, which runs the gates in `release-gates.md` and tags `v0.YYYYMMDD.N`. The corpus audit requires a clone cache of the parent specification repository; `docs/guides/corpus-audit.md` describes it.

Issue tracking lives in the repository via [beads](https://github.com/steveyegge/beads); the GitHub tracker is unused.

## License

[MIT No Attribution (MIT-0)](LICENSE).
