# scrum-master

## Workflow

All work goes through GitHub issues and pull requests: `/gh-issue-push` to file an issue, `/gh-issue-pop` to pop it into a worktree branch (`../gh-<number>-<slug>`) and finish with the pull request that closes it. Never commit to `main` directly. Beads is the product's tracker backend (srd042, internal/beads), not this repository's tracker.

## Every gh call runs as the organization

The repository is public, so any GitHub account can open issues and pull requests on it, and gh's active account on this machine is a personal one. An issue, pull request, comment, or review made with it would publish that person's name against this repository. Therefore every `gh` command here, including those the skills run, carries the organization's token in the same shell command:

```bash
GH_TOKEN=$(gh auth token --user gray-lake-labs-owner) gh <command>
```

Before the first `gh` write in a session, check the account: `GH_TOKEN=$(gh auth token --user gray-lake-labs-owner) gh api user -q .login` must print `gray-lake-labs-owner`; if it prints anything else, or the token is missing, stop and say so. Never run `gh auth switch` for this repository: it changes the account for every other repository on the machine. Git pushes need no token in the environment: they authenticate as `gray-lake-labs-owner` through the repository-local `credential.https://github.com.username`.

## Anonymity

The repository is published by the organization, not a person. Nothing committed or written to GitHub (code, docs, commit messages, issues, pull requests, comments, the repository's description or topics) may name an individual, their accounts, or their other repositories. Commits author as `Gray Lake Labs <344459311+gray-lake-labs-owner@users.noreply.github.com>`, repository-local git config, set in every fresh clone before committing.

## Local conventions

- `.claude` is a gitignored symlink to shared local tooling; keep it out of tracking (`.git/info/exclude`).
- `.regression/` holds the regression rig's runs and stays untracked.
