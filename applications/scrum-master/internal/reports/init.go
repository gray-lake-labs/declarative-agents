// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package reports

import (
	"errors"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/corpus"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// InitResult is one file init considered.
type InitResult struct {
	Path   string `json:"path"`
	Action string `json:"action"` // created, kept
}

const deckTemplate = `# {title}

<!-- tpm reports deck renders one slide per outline section of an approved
summary, up to the slide limit (srd018 R4). -->
`

// Init scaffolds a reports repository in a local checkout (srd018 R1): the
// configuration, the current quarter's reports directory, the report
// schema, a summaries directory, and a deck template. It
// never overwrites a file that exists; a dry run writes nothing.
func Init(dir string, cfg Config, now time.Time, dryRun bool) ([]InitResult, error) {
	cfg.defaults()
	conf, err := corpus.FormatConfig("reports", "Reports configuration", cfg)
	if err != nil {
		return nil, err
	}
	example := Render("po-1", ISOWeek(now), []Entry{
		{"sprint_goal": "group/product-owners/po-1#4", "component": "grader", "status": "at-risk", "complete": "no", "problems": "the scoring API returns stale results"},
	}, []Entry{
		{"blocker": "stale results from the scoring API", "component": "grader", "since": now.Format("2006-01-02"), "waiting-on": "scoring-api"},
	})
	files := []struct{ path, content string }{
		{ConfigFile, string(conf)},
		{filepath.Join("reports", Quarter(now), ".gitkeep"), ""},
		{filepath.Join("schema", "report.md"), example + "\n<!-- The report format (config format report): one file per person per ISO week at\nreports/<YYYY-Qn>/<YYYY-Www>-<username>.md; tpm reports validate checks it. -->\n"},
		{filepath.Join("summaries", ".gitkeep"), ""},
		{filepath.Join("deck", "template.md"), deckTemplate},
	}
	var out []InitResult
	for _, f := range files {
		p := filepath.Join(dir, f.path)
		r := InitResult{Path: f.path, Action: "kept"}
		if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
			r.Action = "created"
			if !dryRun {
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					return out, err
				}
				if err := os.WriteFile(p, []byte(f.content), 0o644); err != nil {
					return out, err
				}
			}
		} else if err != nil {
			return out, err
		}
		out = append(out, r)
	}
	return out, nil
}
