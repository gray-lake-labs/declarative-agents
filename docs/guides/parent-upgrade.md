# Parent Upgrade

This corpus derives some of its requirements from the parent,
gray-lake-labs/declarative-agents. Each such requirement carries a
`derived_from` entry naming the parent release it was validated against,
and `docs/external-pins.yaml` names the release the corpus tracks. When the
parent ships a new release, we move the pin, and every requirement whose
cited parent text changed has to be looked at again by a person. This guide
is that workflow, invoked by name as **parent-upgrade**.

## Where things live

| What | Where |
|---|---|
| Tracked pin | `docs/external-pins.yaml`, key `github.com/gray-lake-labs/declarative-agents` |
| Release cache | the user cache directory, `scrum-master/release/<tag>`: each release's agent and source, downloaded and checksum-verified on first use, and the spec tool built from that source |
| Provenance entries | the `derived_from` attribute of an SRD's `req` marker, a JSON list of `repo`, `release`, `srd`, `item`, `reason` |
| Corpus format | the parent's Markdown format (its eng10), checked by the audit's `markdown-format` and `not-canonical` checks |
| Format tools | `mage convert`, `mage fmt`, `mage specCheck` (`magefiles/spec.go`), this repository's spec tool, built once per pin against the pinned release's Markdown library |
| Audit that reports staleness | the parent corpus audit, `docs/guides/corpus-audit.md` |
| Next bump | the open issue `Recurring: bump the declarative-agents pin` |

## The resolution idiom

A stale entry is one whose cited parent item changed between the release
the entry was validated against and the tracked pin. It is resolved in one
change: revalidate the local requirement against the parent's new text (or
rewrite it if the parent's change demands it), then set the entry's
`release` to the pin. Never re-pin without reading the new text, and never
rewrite the requirement without re-pinning; either half alone leaves the
corpus claiming a provenance nobody checked.

## Steps

1. **Check the release is published.** The new tag lists its binaries and
   `SHA256SUMS` on the parent's releases page; mage downloads them on first
   use after the pin moves.

2. **Move the pin.** Set the key in `docs/external-pins.yaml` to the new tag.

3. **Convert and format.** The format tools rebuild against the new pin on
   first use. `mage convert all` converts any corpus document still in YAML
   to Markdown in place and prints what the conversion reported; it has
   work only when this corpus gained a YAML document or the release
   declares a new document type. `mage fmt all` rewrites every document in
   the release's canonical form, which a release that changes the format
   also changes. `mage specCheck all` lists what is left, each finding with
   its fix hint. Commit the result with the pin: the audit's
   `not-canonical` check fails on any document `mage fmt` would still
   change.

4. **Re-pin what did not change.** For every `derived_from` entry, compare the
   cited item's text at the entry's `release` and at the new pin
   (the file under each release's cached `src/`, `scrum-master/release/<release>`
   in the user cache, which `mage audit` downloads for the pin; download an
   older one the same way. The parent's SRDs are Markdown, each item under
   its own `req` marker). Where
   the text is byte-identical, set `release` to the pin. This is the
   mechanical half the parent's `mage upgrade` performs for its own modules;
   that target is wired to the parent's module list and does not reach this
   repository, so we do it here, by script or by hand.

5. **Read the worklist.** Run the corpus audit. Every
   `stale-external-reference` warning is one entry whose cited text changed;
   it names the child requirement, the parent SRD and item, the validated
   release, and the pin. This list is the equivalent of the parent's
   `mage upgrade:status`. An entry citing an item the parent retired shows as
   `retired-external-reference` and joins the list.

6. **The judgment pass.** Group the findings by child requirement and file
   them in the tracker:
   - one epic titled `Upgrade to <release>`, whose body names the old and new
     pin and lists the findings;
   - one sub-issue per child requirement, carrying the child requirement's
     current text, each cited parent item's text at the validated release and
     at the pin, and these acceptance criteria: the requirement is
     revalidated against the new text or rewritten; every entry of that
     requirement is re-pinned to the new release in the same change; the
     corpus audit reports no stale finding for it.

   An empty worklist needs no epic: commit the pin bump with the mechanical
   re-pins, then verify.

7. **Work the epic** through the inner loop like any other: pop it with implement,
   build each issue, merge. The audit after the merge shows no stale
   finding.

8. **Verify.** A green audit is not green tests: run `go test ./...` and
   `mage regression:all`, whose rig seeds a Markdown road map and subsystem
   SRDs, and record both in the bump's merged change. A bump that cannot
   reach the rig says so there rather than leaving it implied.

## Recurrence

The repository keeps one open issue for the next bump, titled
`Recurring: bump the declarative-agents pin`. The change that moves the
pin closes it, and whoever closes it opens a fresh copy with the pinned
release updated (gh-issue-pop Phase 6), so the pin history reads from the
issue chain and the next release always has somewhere to land. A larger
adoption, such as a format change, is an epic whose merge closes the
recurring issue too.

## Exercise, 2026-10-15

We ran the workflow once with a real parent bump to confirm it produces the
right worklist. Between `v0.20260923.3` and `v0.20261014.0` no requirement
item of the parent changed text (only new items and path fixes), so that
bump yields no stale entry; the earliest release before it with changed
items is `v0.20260917.2`, with 57.

In a scratch copy of this corpus, we gave `srd013-program` R1.1 a drill entry
citing the parent's `srd006-yaml-machine-config` R1.9 validated at
`v0.20260917.2`, with the pin at `v0.20260917.2`, and kept no other entry.
The audit passed with no finding. Moving the pin to `v0.20261014.0` produced
exactly one `stale-external-reference` warning on that entry: the parent
renamed the `instantiate` field of R1.9 to `expand`. The judgment pass turned
it into one `Upgrade to v0.20261014.0` epic with one sub-issue, filed as a
drill (#109, sub-issue #110) and closed. The drill entry was never part of this corpus.

## Format change, 2026-10-16

`v0.20261016.1` was the first parent release whose corpus is Markdown, and
it reads no YAML corpus document. The bump to it was epic #211: the YAML
was first reshaped so every document converts, the format tools were added,
and the pin moved with the conversion of all 56 documents in one change,
re-pinning three `derived_from` entries whose cited text had not changed.
tpm reads a program's Markdown road map and SRDs and still reads YAML ones,
so a program converts on its own schedule. Objective documents stayed YAML
at that pin; the next bump wrote them as SRDs.

## Declared fields, 2026-10-17

`v0.20261017.1` declares every field a document may hold, in order, and
drops from the corpus any document with a field its slot does not list.
The declarations are built into the release and this repository does not
change the parent, so every field moved onto one the release declares
(epic #219). This corpus reordered 54 fields and moved 56 undeclared
ones into declared fields or into their record's prose; a map under a
field that declares sub-fields became a table, because the formatter
prints such a map as headings the check then rejects. References now
render as links, which the format tools write by indexing the corpus.

A program's documents use the same declared fields for the data the
process needs (design.md D2 and document types):

| Document | Data | Declared field |
| --- | --- | --- |
| objective, an SRD | owner, horizon | `context` items |
| objective | due, delivered_by per group | the group's `applicability` items |
| objective | environment per criterion | the criterion's `applicability` |
| objective | tests per criterion | `tests` |
| subsystem SRD | owner | `context` item |
| subsystem SRD | components | `implementation_targets` table |
| road map release | objective items | `deliverables` |
| road map release | tag, milestone | `release` > `publication` items |

tpm reads these documents only in Markdown. `mage convert` converts a
program still in YAML once its data is in the declared fields, so such a
program moves owner, horizon, due, delivered_by, test, and environment
first.

A later release that renames or drops one of these fields moves the
mapping in the same change as the pin.

## No YAML, 2026-10-17

After the declared-fields bump, everything this repository holds became
Markdown with tags (epic #230): the design constitution is a constitution
document, the configuration files scrum-master reads are config documents
(design.md D11), and tpm no longer reads a YAML corpus document. Five files
stay as they are because a consumer outside the repository requires their
format, as design.md D2 lists, and a gate test in mage audit fails on any
other YAML or JSON file. A bump that renames a field the gates file or a
config format uses moves those files in the same change.
