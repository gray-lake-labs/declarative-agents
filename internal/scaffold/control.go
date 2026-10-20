package scaffold

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// labelAxes are the axes of the labels config format in its order, with the
// attributes its example gives each. state carries no prefix: its values are
// whole label names.
var labelAxes = []struct{ name, prefix, exclusive, required string }{
	{"plane", "plane::", "true", "all"},
	{"type", "type::", "true", "none"},
	{"kind", "kind::", "true", "data"},
	{"op", "op::", "true", "none"},
	{"state", "", "false", "none"},
}

// LabelsDocument renders labels as control/labels.md in the labels config
// format, so tpm issue labels reads back exactly labels.
func LabelsDocument(labels []tracker.Label) string {
	var b strings.Builder
	b.WriteString("---\nid: labels\ntitle: Label schema\n---\n\n")
	b.WriteString("<!-- field: version -->\n## Version\n\n1\n\n<!-- field: axes -->\n## Axes\n")
	axisOf := func(name string) string {
		for _, a := range labelAxes {
			if a.prefix != "" && strings.HasPrefix(name, a.prefix) {
				return a.name
			}
		}
		return "state"
	}
	for _, a := range labelAxes {
		fmt.Fprintf(&b, "\n<!-- field: %s -->\n### %s\n\n", a.name, strings.ToUpper(a.name[:1])+a.name[1:])
		prefix := a.prefix
		if prefix == "" {
			prefix = "None"
		}
		fmt.Fprintf(&b, "<!-- field: prefix -->\n#### Prefix\n\n%s\n\n", prefix)
		fmt.Fprintf(&b, "<!-- field: exclusive -->\n#### Exclusive\n\n%s\n\n", a.exclusive)
		fmt.Fprintf(&b, "<!-- field: required -->\n#### Required\n\n%s\n\n", a.required)
		b.WriteString("<!-- field: values -->\n#### Values\n\n| name | color | description |\n| --- | --- | --- |\n")
		for _, l := range labels {
			if axisOf(l.Name) == a.name {
				fmt.Fprintf(&b, "| %s | %s | %s |\n", strings.TrimPrefix(l.Name, a.prefix), l.Color, l.Description)
			}
		}
		if a.name == "kind" {
			b.WriteString("\n<!-- field: component -->\n### Component\n\n<!-- field: prefix -->\n#### Prefix\n\ncomponent::\n\n" +
				"<!-- field: exclusive -->\n#### Exclusive\n\ntrue\n\n<!-- field: required -->\n#### Required\n\nall\n\n" +
				"<!-- field: values_from -->\n#### Values from\n\nsubsystem SRDs (srd001 R21.1)\n\n<!-- field: color -->\n#### Color\n\n#7f8c8d\n")
		}
	}
	return b.String()
}

// PolicyDocument renders p as control/policy.md in the policy config format:
// the settings tpm's policy reader takes from the file.
func PolicyDocument(p program.Policy) string {
	var b strings.Builder
	b.WriteString("---\nid: policy\ntitle: Policy\n---\n\n<!-- field: version -->\n## Version\n\n1\n\n")
	b.WriteString("<!-- field: admission -->\n## Admission\n\n")
	fmt.Fprintf(&b, "<!-- field: require_milestone -->\n### Require milestone\n\n%t\n\n", p.RequireMilestone)
	fmt.Fprintf(&b, "<!-- field: require_acceptance_criteria -->\n### Require acceptance criteria\n\n%t\n\n", len(p.RequireAC) > 0)
	fmt.Fprintf(&b, "<!-- field: audit_command -->\n### Audit command\n\n%s\n\n", p.AuditCommand)
	b.WriteString("<!-- field: escalation -->\n## Escalation\n\n")
	for _, e := range []struct {
		field string
		value int
	}{
		{"blocker_age_days", p.Escalation.BlockerAgeDays},
		{"waiting_chain_length", p.Escalation.WaitingChainLength},
		{"slip_releases", p.Escalation.SlipReleases},
		{"evidence_age_days", p.Escalation.EvidenceAgeDays},
	} {
		fmt.Fprintf(&b, "<!-- field: %s -->\n### %s\n\n%d\n\n", e.field, heading(e.field), e.value)
	}
	b.WriteString("<!-- field: releases -->\n## Releases\n\n")
	fmt.Fprintf(&b, "<!-- field: scheme -->\n### Scheme\n\n%s\n\n", p.Releases.Scheme)
	fmt.Fprintf(&b, "<!-- field: breaking -->\n### Breaking\n\n%s\n\n", p.Releases.Breaking)
	fmt.Fprintf(&b, "<!-- field: tag_format -->\n### Tag format\n\n%s\n\n", p.Releases.TagFormat)
	b.WriteString("<!-- field: definition_of_done -->\n## Definition of done\n")
	artifacts := make([]string, 0, len(p.DefinitionOfDone))
	for a := range p.DefinitionOfDone {
		artifacts = append(artifacts, a)
	}
	sort.Strings(artifacts)
	for _, a := range artifacts {
		fmt.Fprintf(&b, "\n<!-- field: %s -->\n### %s\n\n| kind | environment | suite | command |\n| --- | --- | --- | --- |\n", a, heading(a))
		for _, c := range p.DefinitionOfDone[a] {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", c.Kind, c.Environment, c.Suite, c.Command)
		}
	}
	return b.String()
}

func heading(field string) string {
	s := strings.ReplaceAll(field, "_", " ")
	return strings.ToUpper(s[:1]) + s[1:]
}
