---
id: eng04-beads-sync
title: "Beads Sync: One Tracker Repository, Pull Before Read, Push After Write"
---

<!-- field: introduction -->
## Introduction

srd042 puts the program's facts in one tracker repository that every
working clone synchronizes with through git. A hosted tracker gives
every reader the same view because there is one server; a repository
gives every reader the same view only if the discipline around pulling
and pushing is stated and held by the tools, not by memory. This
guideline states that discipline: when tpm synchronizes, how a conflict
resolves, and what offline operation promises.

<!-- section: the-tracker-repository -->
## The tracker repository

The tracker repository holds the beads database and nothing else: no
code, no documents, no CI. Issues are .beads/issues.jsonl, bd's own
interchange file; milestones are .beads/milestones.jsonl and the label
schema .beads/labels.jsonl beside it, each record belonging to a group or
project path, merged a record at a time with the later writer winning
when both clones changed one. Every program repository's configuration
document names it by URL. Access to it is access to the program's facts,
so it carries the same visibility as the most restricted repository it
tracks. A working clone keeps its local beads state as a cache under the
code repository; the cache is disposable and never the thing another
clone reads.

<!-- section: pull-before-read-push-after-write -->
## Pull before read, push after write

Every tpm command that reads the tracker pulls the tracker repository
first; every command that writes pushes immediately after the write, in
the same invocation. A command is not done until its push lands. The
rule is in tpm, not in the skills: a skill cannot forget it, and a
person running tpm by hand gets it for free. Two consequences follow.
First, a command's output always reflects the tracker as of its own
start, named by the tracker repository commit it pulled. Second, the
window for conflicting writes is the length of one command, not the
length of a working session.

<!-- section: conflicts -->
## Conflicts

Two clones that write between each other's synchronizations produce a
merge in the tracker repository, and the merge resolves by rules, not
by hand. Field edits on different issues merge trivially. Field edits
on the same issue merge field-wise, last writer wins, except for the
two fields the state machine guards: a state transition replays through
the srd041 table, and a replay the table refuses is dropped with the
refusal recorded as a comment on the issue; an assignee conflict keeps
the earlier assignment, and the later actor's pop is reported back to
them as lost to an earlier taker. Comments are append-only and never
conflict. A conflict the rules cannot resolve stops the push and
reports, which in practice means a malformed database, not a
disagreement about facts.

<!-- section: offline -->
## Offline

A clone that cannot reach the tracker repository still works: reads
serve the cache and say so, naming the last pulled commit and its time
(srd042 R3.3); writes land in the cache and are pushed by the next
successful synchronization of any command. What offline operation never
does is silently present the cache as current, or let two offline
sessions believe they both hold an assignment: an assignment taken
offline is provisional, is marked so in the pop's own output, and
confirms or reports as lost when tpm tracker reconcile runs back online. Every command's output names the tracker commit it read, and says when that was the cache.

<!-- section: choosing-the-backend -->
## Choosing the backend

TPM_TRACKER, set to the tracker repository's URL, selects the beads
backend; unset, tpm talks to GitLab as before. The tracker clone lives at
TPM_TRACKER_CLONE, or under the user cache directory, cloned on first use.
The person is TPM_USER, or git's user.name. Issues come from the tracker
repository; files, branches, tags, and comparisons come from git: a
project's checkout is the sibling of the command's checkout named for the
project's last path segment, as repositories sit side by side under their
owner, and a project with no such sibling is served by the command's own
checkout and its origin. Delivery is a merge commit on the default
branch citing project#iid, made by tpm program merge after the audit gate
for a docs/ change; tpm program close finds it and closes the issue on it, and refuses a fast-forward or an empty
branch, because neither is a delivery a reviewer can point to. A merge
also closes every other issue its message cites, such as the objective a
sprint-end merge closes; tpm program close closes such an issue, which has
no pop branch, on the first merge commit that cites it, matching the
number exactly. A project's merged history, what reports read as merged merge requests,
is the merge commits on its default branch: each numbered by its place in
the branch's first-parent history, by its author on its commit date, and
closing every project#iid its message cites.

<!-- section: what-this-replaces -->
## What this replaces

The GitLab instance was the serializer: every write landed on one
server in one order. This guideline moves that job to git plus the
rules above, which is a weaker serializer with a longer window, bought
deliberately for the absence of the server. The program accepts the
trade because its write pattern is a handful of actors making small,
mostly disjoint writes, with the one genuinely contended field —
assignee — already guarded by the pop discipline of srd041 R9.

<!-- field: references -->
## References

- srd042-beads-tracker: the requirements this guideline serves
- srd041-states: the state machine conflicts replay through
- eng01-regression-rig: the rig this protocol's fixtures will replace when 03.0 is planned
