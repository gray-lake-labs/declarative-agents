package scaffold

import (
	"fmt"
	"strings"

	"github.com/gray-lake-labs/scrum-master"
	"github.com/gray-lake-labs/scrum-master/internal/cli"
	"github.com/gray-lake-labs/scrum-master/internal/help"
	"github.com/gray-lake-labs/scrum-master/internal/install"
)

// AgentsFile is the orientation file that tells an agent which tpm commands
// and skills work on the repository and the rules for changing it (srd043
// R1.6).
const AgentsFile = "AGENTS.md"

// skillLevels names the level a skill serves when it serves only one; every
// other skill serves both.
var skillLevels = map[string]string{
	"tasks":      "system",
	"sprint-end": "subsystem",
}

// agentsFile renders AGENTS.md for a repository at level from the binary's
// dispatch table, its help pages, and its embedded skills, so it names what
// this tpm runs and cannot drift from it.
func agentsFile(level string) ([]byte, error) {
	pages, err := help.Pages()
	if err != nil {
		return nil, err
	}
	summaries := map[string]string{}
	for _, p := range pages {
		summaries[p.Name] = p.Summary
	}
	skills, err := install.Skills(scrummaster.Plugin, scrummaster.SkillsDir)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString(BlockBegin + "\n# Agent instructions\n\n")
	fmt.Fprintf(&b, "This is a program's %s repository, run with tpm, the command-line tool of\n", level)
	b.WriteString("the AI-native scrum process. Read README.md first: it says what the\n" +
		"repository holds and how the planning hierarchy nests.\n\n")
	b.WriteString("## Rules\n\n" +
		"- Every change starts as an issue: file it with the specify skill or tpm issue file, take it with the implement skill or tpm issue pop, and deliver it with tpm issue merge. Never commit to main directly.\n" +
		"- A change under docs/ is a change of intent: the corpus gate must pass before it merges.\n" +
		"- Pass --control <system repository> to every tpm command that takes it, so the program's control/policy.md and control/labels.md apply instead of tpm's defaults.\n" +
		"- Fetch facts with tpm and its --json output; never read or edit tracker files by hand. tpm does the workflow, and the model writes only language (docs/constitutions/design.md D10).\n" +
		"- Every document follows docs/constitutions/design.md: Markdown with marker tags, in the format of the parent release docs/external-pins.yaml pins. After upgrading tpm, run tpm doctor to add missing files and repair the format.\n" +
		"- Every figure is Mermaid in a fenced code block inside the Markdown document that shows it. Put no generated image file, and no other diagram source, under docs/.\n\n")
	b.WriteString("## Skills\n\nInstall them for your agent with tpm install --agent <agent>.\n\n")
	for _, s := range skills {
		if l, ok := skillLevels[s.Name]; ok && l != level {
			continue
		}
		desc, _, _ := strings.Cut(s.Description, " Use when")
		fmt.Fprintf(&b, "- %s: %s\n", s.Name, desc)
	}
	b.WriteString("\n## tpm commands\n\nRun tpm help for the process and tpm help <group> for one group.\n\n")
	for _, g := range cli.Default.Groups() {
		summary := summaries[g]
		if g == "help" {
			summary = "the process, each group, and each concept, from the binary"
		}
		if summary == "" {
			if cs := cli.Default.Commands(g); len(cs) > 0 {
				summary = cs[0].Summary
			}
		}
		fmt.Fprintf(&b, "- tpm %s: %s\n", g, summary)
	}
	b.WriteString(BlockEnd + "\n")
	return []byte(b.String()), nil
}
