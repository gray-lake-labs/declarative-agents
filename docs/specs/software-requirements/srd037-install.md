---
id: srd037-install
title: "Script: install (tpm install)"
---

<!-- field: problem -->
## Problem

The skills are what an agent runs: Cursor, Claude Code, or GitHub
Copilot, in the repository a developer works in. Copied by hand, a
skill falls out of step with the tpm it calls: a renamed flag, a new
subcommand, a refusal the skill does not expect. Every repository then
carries a slightly different process.

The skills ship as one Agent Plugins 1.0 package (agent-plugins.org),
plugin/ in this repository: a plugin.json manifest and one Agent Skills
directory per skill (agentskills.io). A client that loads Agent Plugins
can install the package as it is. tpm install makes the binary the one
source of the same skills for every agent: the package is embedded in tpm
when it is built, and tpm install writes each skill as an Agent Skills
directory where each agent reads skills. Every
installed file names the tpm version and the source checksum it came
from, so a later install updates what tpm wrote and keeps what a person
edited. The installed files belong to the tpm each developer has
installed, not to the repository, so git ignores them, and tpm install
--check tells the developer when they no longer match the local binary.

<!-- field: goals -->
## Goals

- G1: A tpm release carries exactly the skills written against its subcommands, as a conforming Agent Plugins 1.0 package.
- G2: One command places the skills where Cursor, Claude Code, or GitHub Copilot reads them.
- G3: The installed skills never enter the repository; git ignores them, and the developer learns of drift locally.

<!-- req: R1 -->
## R1 - Embedding

<!-- req: R1.1 -->
### R1.1

The binary embeds the plugin package plugin/ at build time: its plugin.json and every skills/<name>/SKILL.md. Each SKILL.md conforms to the Agent Skills specification (front matter name equal to its directory name, lowercase letters, digits, and single hyphens, at most 64 characters; description of 1 to 1024 characters; no other keys than the specification's) and is at most 50 lines (design.md D10).

<!-- req: R1.2 -->
### R1.2

tpm install --list names every embedded skill with its description.

<!-- req: R1.3 -->
### R1.3

plugin/ conforms to Agent Plugins 1.0: plugin.json at its root is a JSON object with only the specification's fields, $schema https://agent-plugins.org/schemas/1.0.0/plugin.schema.json, and a name of 1 to 64 characters of a-z, 0-9, hyphens, and periods, alphanumeric at both ends, without doubled hyphens or periods; skills sit exactly one level under skills/; no path leaves the package. mage audit fails a package that does not conform.

<!-- req: R2 -->
## R2 - Targets

<!-- req: R2.1 -->
### R2.1

--agent names cursor, claude, or copilot, repeatable, or all. --dir is the repository to install into, by default the root of the git repository holding the working directory.

<!-- req: R2.2 -->
### R2.2

Each agent receives every skill as an Agent Skills directory with the skill's own SKILL.md: Cursor in .cursor/skills/<name>/, Claude Code in .claude/skills/<name>/, and GitHub Copilot in .github/skills/<name>/.

<!-- req: R3 -->
## R3 - Version stamp and updates

<!-- req: R3.1 -->
### R3.1

Every installed SKILL.md carries a comment naming the tpm version and the checksum of the text tpm wrote, less the comment itself, on the first line after its front matter, which must open the file.

<!-- req: R3.2 -->
### R3.2

A later install rewrites a file whose content still matches its stamp, keeps a file a person edited and reports it, and replaces an edited file only with --force.

<!-- req: R3.3 -->
### R3.3

A skill no longer embedded is removed only when its installed file is unedited; an edited one is reported and kept. A file tpm stamped in an earlier layout (.cursor/commands, .claude/commands, .github/prompts) is removed the same way.

<!-- req: R4 -->
## R4 - Check

<!-- req: R4.1 -->
### R4.1

tpm install --check writes nothing, names each installed file that is missing, outdated against the binary, edited, stale (a stamped file whose skill is no longer embedded), or unignored (a stamped file the exclude block does not list), and exits non-zero when any is.

<!-- req: R5 -->
## R5 - Common behavior

<!-- req: R5.1 -->
### R5.1

Honours --json and --dry-run and never prompts (srd010 R1).

<!-- req: R5.2 -->
### R5.2

Writes nothing outside --dir and the clone's git directory. Outside the agents' skills directories it writes only its block in .git/info/exclude, which git applies and never commits, so an install changes no tracked file.

<!-- req: R6 -->
## R6 - Ignored by git

<!-- req: R6.1 -->
### R6.1

tpm install keeps one block in the clone's .git/info/exclude, the path git rev-parse --git-path info/exclude names, so every linked worktree of the clone shares it. The block sits between the lines "# tpm install: skill commands from the local tpm binary" and "# end tpm install" and lists exactly the files tpm owns (those carrying its stamp), one anchored path per line, sorted. A file without the stamp is never listed, so a command the repository owns stays tracked. tpm install and --check refuse a --dir that is not a git repository.

<!-- req: R6.2 -->
### R6.2

Each install rewrites the block from the files tpm owns after the run, drops a removed stale skill, and leaves every line outside the block as it was. With no exclude file, it creates one holding only the block.

<!-- req: R6.3 -->
### R6.3

The installed files are untracked, so a new worktree of the clone has none; tpm issue pop installs the checkout's agents into each worktree it creates (srd013 R7.4), and the shared exclude block already lists their paths.

<!-- field: non_goals -->
## Non-goals

- Distributing the tpm binary itself; that is the release of tpm.
- Agents other than Cursor, Claude Code, and GitHub Copilot.
- Per-repository templating; repository-specific values come from tpm at run time.

<!-- field: acceptance_criteria -->
## Acceptance criteria

<!-- ac: AC1 -->
### AC1

plugin/ is embedded; every SKILL.md conforms to Agent Skills and is at most 50 lines; --list names each skill with its description; mage audit fails a manifest or skill that breaks Agent Plugins 1.0 or Agent Skills.

<!-- field: traces -->
#### Traces

- [R1.1](#r11)
- [R1.2](#r12)
- [R1.3](#r13)

<!-- ac: AC2 -->
### AC2

Installing for cursor, claude, and copilot writes each skill as an Agent Skills directory in .cursor/skills, .claude/skills, or .github/skills, into the repository root by default.

<!-- field: traces -->
#### Traces

- [R2.1](#r21)
- [R2.2](#r22)

<!-- ac: AC3 -->
### AC3

Every installed file carries the version and checksum stamp; a second install changes nothing; an edited file is kept and reported unless --force; a skill no longer embedded, or a file of the earlier commands layout, is removed only when unedited.

<!-- field: traces -->
#### Traces

- [R3.1](#r31)
- [R3.2](#r32)
- [R3.3](#r33)

<!-- ac: AC4 -->
### AC4

--check writes nothing and exits non-zero naming each missing, outdated, edited, stale, or unignored file, and passes after an install.

<!-- field: traces -->
#### Traces

- [R4.1](#r41)

<!-- ac: AC5 -->
### AC5

install honours --json and --dry-run, never prompts, and writes nothing outside --dir, the agents' skills directories, and its block in .git/info/exclude.

<!-- field: traces -->
#### Traces

- [R5.1](#r51)
- [R5.2](#r52)

<!-- ac: AC6 -->
### AC6

After an install, git ignores every installed file and nothing else: git status shows no change, in the main checkout and in a linked worktree; a command without the stamp in the same directory is not ignored; lines outside the block are unchanged; a directory that is not a git repository is refused; a worktree tpm issue pop creates receives the checkout's installed agents and stays clean.

<!-- field: traces -->
#### Traces

- [R6.1](#r61)
- [R6.2](#r62)
- [R6.3](#r63)
