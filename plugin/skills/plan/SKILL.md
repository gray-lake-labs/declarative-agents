---
name: plan
description: Plan the next release in the system repository, choosing its initiative items, publishing the plan as a tag, dating the period, and filing its sprint-end issues; or, in a team's repository, run Sprint Planning from the ranked stories (srd031, srd034 R3.1, srd045 R5). Use when the technical program manager asks to plan the next period or release, or a Product Owner asks to plan the team's sprint.
---

# plan

Invoking this skill is the decision to plan the next release. Every step
is a tpm command; this skill asks only the items, the version, and the end
date (design.md D10, srd031 R3).

Input: the system repository path, and the period's `tpm reports aggregate
--json` output when a reports repository exists.

1. File a plane::management planning issue with specify and take it
   with implement; work in its worktree.
2. Run `tpm plan candidates --dir . --system <system> --aggregate <aggregate.json> --json`.
3. Show the ranked items (slipped first) and the held-back items with what
   they lack. Ask which items enter the release, and confirm or change the
   recommended version, end date, and tag. Ask nothing else.
4. Run `tpm plan write --dir . --system <system> --version <version> --tag <tag> --items <a,b> --issue <planning issue> --json`.
   Show the items it returned for a missing test or environment.
5. Commit docs/road-map.md and run `tpm issue merge` as implement does.
6. Once it merges, on main: `tpm release gate --artifact corpus --version <the tag's version> --issue <planning issue>`,
   then `tpm release tag` with the same flags. That tag publishes the plan.
7. Run `tpm plan push --dir . --system <system> --version <version> --end <end date> --json`.
   Show the milestone, the sprint issue, the sprint-end issues, the
   planned events, and anything skipped.
8. Run tasks for the release.

Sprint Planning, in a team's repository once tasks filed its Sprint Goal:

1. Run `tpm plan candidates --dir . --json` and show the backlog: the
   stories by rank with points, the velocity, and the recommended stories.
   When velocity is not measured, ask the Product Owner for an assumed one
   and run it again with `--velocity <points>`.
2. Ask which stories enter the sprint. Ask nothing else.
3. Run `tpm plan backlog --project <team repository> --release <version> --stories <a,b> --dir . --json`
   and show the Sprint Goal it updated.
4. Run make-work for each chosen story; each issue carries `Implements: <story>`.
