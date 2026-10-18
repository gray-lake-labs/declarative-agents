---
id: test-rel01.1-weekly-status
title: Test Suite for Release 01.1 Status Comes Back Weekly
release: "01.1"
status: implemented
---

<!-- field: overview -->
## Overview

Each run starts with mage regression:up creating the repositories below in
a fresh rig subgroup and ends with mage regression:down removing them.
The weekly stage scaffolds a reports repository in the rig subgroup,
files a status issue for a listed Product Owner, works it to a validated
report, and proves the next instance with the next due date. The roll-up
stage aggregates fixture reports and proves every signal, its phrasing by
component, and that an actionable signal becomes an issue.

<!-- field: traces -->
## Traces

- [rel01.1-uc001-weekly-status](../use-cases/rel01.1-uc001-weekly-status.md)
- [rel01.1-uc002-period-rollup](../use-cases/rel01.1-uc002-period-rollup.md)

<!-- field: preconditions -->
## Preconditions

- mage regression:up creates a rig-<run-id> subgroup with a system repository, a reports repository, and one Product Owner repository with open objective issue fixtures, and writes their paths as JSON for the integration targets (eng01).
- mage regression:down destroys the subgroup after the targets run, including when they fail.
- Fixture reports cover a week and a period, including a shared blocker, an aging blocker, a slipped commitment, a missing report, and a reported-complete objective with no met event.

<!-- field: test_cases -->
## Test cases

<!-- case: tpm reports init scaffolds once | status: implemented -->
### tpm reports init scaffolds once

<!-- field: use_case -->
#### Use case

[rel01.1-uc001-weekly-status](../use-cases/rel01.1-uc001-weekly-status.md)

<!-- field: integration_target -->
#### Integration target

mage integration:reports

<!-- field: go_test -->
#### Go test

go test ./internal/reports -run '^TestInitScaffoldsOnceAndNeverOverwrites$'

<!-- field: expected_outputs | form: "tree" -->
#### Expected outputs

- layout: reports directory per quarter, schema, summaries directory, validate CI job, deck template, configuration
- second_run: no file changed

<!-- field: traces -->
#### Traces

- [srd018-reports](../software-requirements/srd018-reports.md) [AC1](../software-requirements/srd018-reports.md#ac1)

<!-- case: The status issue is dated and mentions without assigning | status: implemented -->
### The status issue is dated and mentions without assigning

<!-- field: use_case -->
#### Use case

[rel01.1-uc001-weekly-status](../use-cases/rel01.1-uc001-weekly-status.md)

<!-- field: integration_target -->
#### Integration target

mage integration:weeklyStatus

<!-- field: go_test -->
#### Go test

go test ./internal/reports -run '^Test(StatusIssueDueAfterReminderWeekdayMentionsOwner|StatusIssuesIdempotentPerWeek)$'

<!-- field: expected_outputs | form: "tree" -->
#### Expected outputs

- due_date: the day after the configured reminder weekday
- mention: the listed person
- assignee: none
- recurrence_section: present
- second_run: no issue filed

<!-- field: traces -->
#### Traces

- [srd018-reports](../software-requirements/srd018-reports.md) [AC7](../software-requirements/srd018-reports.md#ac7)
- [srd018-reports](../software-requirements/srd018-reports.md) [AC5](../software-requirements/srd018-reports.md#ac5)
- [srd018-reports](../software-requirements/srd018-reports.md) [AC6](../software-requirements/srd018-reports.md#ac6)

<!-- case: The worked status issue yields a valid per-objective report | status: implemented -->
### The worked status issue yields a valid per-objective report

<!-- field: use_case -->
#### Use case

[rel01.1-uc001-weekly-status](../use-cases/rel01.1-uc001-weekly-status.md)

<!-- field: integration_target -->
#### Integration target

mage integration:weeklyStatus

<!-- field: go_test -->
#### Go test

go test ./internal/reports -run '^Test(ValidatePassesPerObjectiveReport|ValidateFailsFrontMatterMismatch|WeekListsOpenObjectivesAndActivity)$'

<!-- field: expected_outputs | form: "tree" -->
#### Expected outputs

- report_items: status, problems, and complete or not complete for every open objective
- validate_well_formed: exit 0
- validate_mismatch: non-zero exit naming the report
- week: open objective issues with due dates and the week's activity

<!-- field: traces -->
#### Traces

- [srd018-reports](../software-requirements/srd018-reports.md) [AC7](../software-requirements/srd018-reports.md#ac7)
- [srd018-reports](../software-requirements/srd018-reports.md) [AC2](../software-requirements/srd018-reports.md#ac2)
- [srd018-reports](../software-requirements/srd018-reports.md) [AC6](../software-requirements/srd018-reports.md#ac6)

<!-- case: The next instance exists with the next due date | status: implemented -->
### The next instance exists with the next due date

<!-- field: use_case -->
#### Use case

[rel01.1-uc001-weekly-status](../use-cases/rel01.1-uc001-weekly-status.md)

<!-- field: integration_target -->
#### Integration target

mage integration:weeklyStatus

<!-- field: go_test -->
#### Go test

go test ./internal/standup -run '^TestRecurReopensWithNextDueDateAndPreviousRuns$'

<!-- field: expected_outputs | form: "tree" -->
#### Expected outputs

- closed_issue: closed by the merge request
- next_instance: same title and labels, due one week later, run appended to Previous Runs

<!-- field: traces -->
#### Traces

- [srd018-reports](../software-requirements/srd018-reports.md) [AC7](../software-requirements/srd018-reports.md#ac7)
- [srd010-standup](../software-requirements/srd010-standup.md) [AC6](../software-requirements/srd010-standup.md#ac6)

<!-- case: aggregate reports every signal by component | status: implemented -->
### aggregate reports every signal by component

<!-- field: use_case -->
#### Use case

[rel01.1-uc002-period-rollup](../use-cases/rel01.1-uc002-period-rollup.md)

<!-- field: integration_target -->
#### Integration target

mage integration:reports

<!-- field: go_test -->
#### Go test

go test ./internal/reports -run '^TestAggregateSignalsByComponentNeverByPerson$'

<!-- field: expected_outputs | form: "tree" -->
#### Expected outputs

- signals: shared blocker, aging blocker, slipped commitment, missing report, tracker contradiction, unreported merged work
- phrasing: by component or blocker; no person ranked or compared
- progress: per objective, reported complete distinct from met

<!-- field: traces -->
#### Traces

- [srd018-reports](../software-requirements/srd018-reports.md) [AC3](../software-requirements/srd018-reports.md#ac3)

<!-- case: aggregate is non-interactive and actionable signals become issues | status: implemented -->
### aggregate is non-interactive and actionable signals become issues

<!-- field: use_case -->
#### Use case

[rel01.1-uc002-period-rollup](../use-cases/rel01.1-uc002-period-rollup.md)

<!-- field: integration_target -->
#### Integration target

mage integration:reports

<!-- field: go_test -->
#### Go test

go test ./internal/reports -run '^Test(AggregateHonoursJSONAndDryRun|ActionableSignalIsPushedAsIssue)$'

<!-- field: expected_outputs | form: "tree" -->
#### Expected outputs

- flags: --json and --dry-run honoured; no prompt
- actionable_signal: one issue in do-work's input format

<!-- field: traces -->
#### Traces

- [srd018-reports](../software-requirements/srd018-reports.md) [AC4](../software-requirements/srd018-reports.md#ac4)
- [srd018-reports](../software-requirements/srd018-reports.md) [AC5](../software-requirements/srd018-reports.md#ac5)
