package main

// The corpus is Markdown in the format the pinned parent release declares
// (its eng10; srd040 R1). These targets run that release's spec tool, a
// binary it publishes, over this repository: convert migrates YAML documents, fmt writes
// the canonical form the audit's not-canonical check expects, and specCheck
// reports a document's findings with their fix hints.

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
	tag, err := pinnedRelease(root)
	if err != nil {
		return err
	}
	rel, err := fetchRelease(tag)
	if err != nil {
		return err
	}
	return runStreaming(root, rel.SpecTool, command, scope)
}

// specCheckAt runs the pinned release's spec tool check over every corpus
// document of the repository at dir.
func specCheckAt(dir string) error {
	tag, err := pinnedRelease(dir)
	if err != nil {
		return err
	}
	rel, err := fetchRelease(tag)
	if err != nil {
		return err
	}
	return runStreaming(dir, rel.SpecTool, "check", "all")
}
