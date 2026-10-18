# scrum-master

## Tracker and workflow

This repository uses beads. All work goes through beads issues: `/bd-issue-push` to create an issue, `/bd-issue-pop` to pop it into a worktree branch (`../bd-<id>-<slug>`), never `/gh-issue-push` or `/gh-issue-pop`. Never commit to `main` directly; merge branches locally.

## Anonymity

The repository is published by the organization, not a person. Nothing committed or written to GitHub (code, docs, commit messages, the repository's description or topics) may name an individual, their accounts, or their other repositories. Commits author as `Gray Lake Labs <344459311+gray-lake-labs-owner@users.noreply.github.com>` — repository-local git config, set in every fresh clone and worktree before committing. GitHub issues, pull requests, comments, and releases stay unused: they would be authored by whatever account is signed in, which is the fastest way to break the anonymity. Pushes authenticate as `gray-lake-labs-owner` via the repository-local `credential.https://github.com.username`.

## Local conventions

- `.claude` is a gitignored symlink to shared local tooling; keep it out of tracking (`.git/info/exclude`).
- `regression.md`, `regression.yaml`, and `.regression/` are local rig configuration and stay untracked.
