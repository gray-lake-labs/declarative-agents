---
name: draft
description: Draft one planning document, a theme, initiative, epic, feature, or story, or an engineering document, from a description, in conversation with its owner, then check it and offer its children (srd046). Use when a person has notes or a description and asks to write, draft, or plan a theme, initiative, epic, feature, or story, or to describe how a system or subsystem is built.
---

# draft

Invoking this skill starts the conversation. tpm reads and checks the
hierarchy; the model writes only the document and iterates on it with the
person, asking what the description leaves open (design.md D10, srd046).

Input: the description (a file or text) and the level to write.

1. Run `tpm backlog list --json` and `tpm help backlog`, and read the
   level's fields in `docs/constitutions/design.md`.
2. Model step (structure): say in a few lines what you understood the item
   to be, and propose its id, title, and parent. Name any document it
   overlaps, and a scope too large or too small for the level, with a split,
   merge, or other level.
3. Ask only what the description and the repository leave open, at most
   five questions at a time, each with your recommended answer:

   | Level | Ask about |
   |---|---|
   | theme | horizon; the outcomes that show progress |
   | initiative | horizon; one requirement group per milestone and its due date; owning teams |
   | epic | owner; a horizon longer than one sprint; dependencies; non-goals |
   | feature | owner; its boundary; its tests |
   | story | as a, I want, so that; points and rank; each criterion's test and environment; cites |
   | engineering | which part; its structure, runtime, and deployment; the decisions it explains; one section each |

   Write nothing until the person answers or says to use your
   recommendations. Ask for no other confirmation.
4. Model step (structure): write the document at the level's location with
   status `to_be_agreed` (an engineering document has no status; name it
   in ARCHITECTURE.md's related documents). Run `tpm backlog check` and fix
   every finding.
5. Model step (iterate): show the document and revise it with the person,
   running `tpm backlog check` after each revision.
6. Offer the next level down (initiatives under a theme, epics under an
   initiative, features or stories under an epic) and draft each child
   with this skill.

Never set another status, commit, or merge: the document reaches main
through the repository's own issue. Never estimate, rank, or assign a
person; the person gives points and rank.
