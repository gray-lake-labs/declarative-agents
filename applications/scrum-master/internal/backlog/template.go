// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0

package backlog

import (
	"embed"
	"flag"
	"fmt"
	"strings"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/cli"
)

// templates are one document of each hierarchy level, in the layout the
// gate reads, chained theme to story so together they pass it (GH-20).
//
//go:embed templates/*.md.tmpl
var templates embed.FS

// Template is a document of kind in the layout tpm backlog check accepts:
// its frontmatter, its context tree, and every field the level declares,
// with example values to replace.
func Template(kind string) (string, error) {
	b, err := templates.ReadFile("templates/" + kind + ".md.tmpl")
	if err != nil {
		return "", fmt.Errorf("no template for kind %q; --kind is one of %s", kind, strings.Join(kinds(), ", "))
	}
	return string(b), nil
}

func init() {
	var kind string
	cli.Default.Register(&cli.Command{
		Group: "backlog", Name: "template", Summary: "a document of one hierarchy level in the layout the gate reads, to copy and fill",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&kind, "kind", "", "the level: "+strings.Join(kinds(), ", ")+" (required)")
		},
		Run: func(ctx *cli.Context) error {
			if kind == "" {
				return fmt.Errorf("%w: template needs --kind, one of %s", cli.ErrUsage, strings.Join(kinds(), ", "))
			}
			t, err := Template(kind)
			if err != nil {
				return err
			}
			return ctx.Emit(struct {
				Kind     string `json:"kind"`
				Document string `json:"document"`
			}{kind, t}, t)
		},
	})
}
