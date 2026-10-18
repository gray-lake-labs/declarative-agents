// Package scrum-master holds the Agent Plugins 1.0 package of the skills the
// tpm binary embeds; tpm install writes them for the developer's agent
// (srd037).
package scrummaster

import "embed"

// Plugin is the package plugin/: plugin.json and skills/<name>/SKILL.md.
//
//go:embed plugin
var Plugin embed.FS

// SkillsDir is where the skills sit in Plugin.
const SkillsDir = "plugin/skills"
