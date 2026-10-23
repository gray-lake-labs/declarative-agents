package main

import (
	"path/filepath"

	"github.com/gray-lake-labs/scrum-master/internal/spectool"
)

// The corpus is Markdown in the format the pinned parent release declares
// (its eng10; srd040 R1). These targets run this repository's spec tool,
// built against that release's Markdown library, over the repository:
// convert migrates YAML documents, fmt writes the canonical form the audit's
// not-canonical check expects, and specCheck reports a document's findings
// with their fix hints.

// Convert migrates the tracked YAML corpus documents in scope to Markdown, in
// place: all, a document type, a directory, or a path from the repository root.
func Convert(scope string) error { return runSpecTool("convert", scope) }

// Fmt rewrites the Markdown corpus documents in scope in canonical form.
func Fmt(scope string) error { return runSpecTool("fmt", scope) }

// SpecCheck reports the format findings of the Markdown corpus documents in
// scope, and fails when there are any or a document is not canonical.
func SpecCheck(scope string) error { return runSpecTool("check", scope) }

func runSpecTool(command, scope string) error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	return specToolAt(root, command, scope)
}

// specCheckAt runs the spec tool's check over every corpus document of the
// repository at dir.
func specCheckAt(dir string) error { return specToolAt(dir, "check", "all") }

func specToolAt(dir, command, scope string) error {
	tag, err := pinnedRelease(dir)
	if err != nil {
		return err
	}
	rel, err := fetchRelease(tag)
	if err != nil {
		return err
	}
	tool, err := spectool.Ensure(rel.Dir, filepath.Join(rel.Src, "agent-core"), runStreaming)
	if err != nil {
		return err
	}
	return runStreaming(dir, tool, command, scope)
}
