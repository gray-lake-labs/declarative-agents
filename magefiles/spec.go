package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The corpus is Markdown in the format the pinned Nokia declarative-agents
// release declares (its eng10; srd040 R1). These targets run that release's Markdown
// library over this repository: convert migrates YAML documents, fmt writes
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
	cache, err := parentCheckout(root)
	if err != nil {
		return err
	}
	tool, err := ensureSpecTool(cache, tag, runStreaming)
	if err != nil {
		return err
	}
	return runStreaming(root, tool, command, scope)
}

// releaseModule is the Go module of the pinned release that holds its
// Markdown library; the tool is compiled against the extracted source, so
// this repository's go.mod gains no dependency.
const releaseModule = parentRepository + "/agent-core"

// ensureSpecTool builds the spec tool against the pinned release once per tag.
func ensureSpecTool(cache, tag string, run commandRunner) (string, error) {
	dir, err := releaseDir(tag)
	if err != nil {
		return "", err
	}
	binary := filepath.Join(dir, "spectool")
	if _, err := os.Stat(binary); err == nil {
		return binary, nil
	}
	src, err := extractRelease(cache, tag, dir, run)
	if err != nil {
		return "", err
	}
	build := filepath.Join(dir, "spectool-src")
	if err := writeSpecToolSource(build, filepath.Join(src, "agent-core")); err != nil {
		return "", err
	}
	if err := run(build, "go", "build", "-mod=mod", "-o", binary, "."); err != nil {
		return "", fmt.Errorf("build the spec tool at %s: %w", tag, err)
	}
	return binary, nil
}

// writeSpecToolSource writes the tool's module: its main and a go.mod that
// resolves the release module to the extracted source, with the release's
// own checksums.
func writeSpecToolSource(dir, releaseSource string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	mod := fmt.Sprintf("module scrum-master/spectool\n\ngo 1.26\n\nrequire %s v0.0.0\n\nreplace %s => %s\n",
		releaseModule, releaseModule, releaseSource)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		return err
	}
	if sum, err := os.ReadFile(filepath.Join(releaseSource, "go.sum")); err == nil {
		if err := os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0o644); err != nil {
			return err
		}
	}
	main := strings.ReplaceAll(specToolMain, "RELEASE_MODULE", releaseModule)
	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(main), 0o644)
}

// specToolMain is the tool: it lists the tracked documents with git, picks
// each one's declaration from the release's built-in declarations, indexes
// the corpus so references link across documents, and converts, formats, or
// diagnoses each one.
const specToolMain = `package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"RELEASE_MODULE/pkg/spec/markdown"
)

// objectivePath names an objective document, scrum-master's type written as an
// SRD in the release's format; the release declares no location for it, so
// the tool reads it with the SRD declaration (design.md objective).
var objectivePath = regexp.MustCompile("(^|/)objectives/([a-z]+/)?obj[0-9]{3}-[a-z0-9-]+\\.(yaml|md)$")

// declarationFor picks a document's declaration and the type its scope
// names: an objective document's is the SRD's, under the scope objective.
func declarationFor(declarations markdown.Declarations, path string, source bool) (*markdown.Declaration, string, bool) {
	if objectivePath.MatchString(path) {
		d, ok := declarations["srd"]
		return d, "objective", ok
	}
	var d *markdown.Declaration
	var ok bool
	if source {
		d, ok = declarations.ForSource(path)
	} else {
		d, ok = declarations.ForMarkdown(path)
	}
	if !ok {
		return nil, "", false
	}
	return d, d.Type(), true
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: spectool convert|fmt|check <scope>")
		os.Exit(2)
	}
	command, scope := os.Args[1], os.Args[2]
	declarations, err := markdown.Default()
	if err != nil {
		fail(err)
	}
	out, err := exec.Command("git", "ls-files", "*.yaml", "*.md").Output()
	if err != nil {
		fail(fmt.Errorf("list the tracked documents: %w", err))
	}
	paths := strings.Fields(string(out))
	links := markdown.LinkIndexes{}
	for _, path := range paths {
		if _, _, ok := declarationFor(declarations, path, false); ok && strings.HasSuffix(path, ".md") {
			if data, err := os.ReadFile(path); err == nil {
				links.Add(path, data)
			}
		}
	}
	matched, problems := 0, 0
	for _, path := range paths {
		switch {
		case command == "convert" && strings.HasSuffix(path, ".yaml"):
			d, docType, ok := declarationFor(declarations, path, true)
			if !ok || !inScope(path, docType, scope) {
				continue
			}
			matched++
			data, err := os.ReadFile(path)
			if err != nil {
				fail(err)
			}
			md, report, err := d.ConvertYAML(data, path)
			if err != nil {
				fmt.Println("FAIL " + err.Error())
				problems++
				continue
			}
			if err := os.WriteFile(report.Target, md, 0o644); err != nil {
				fail(err)
			}
			if err := os.Remove(path); err != nil {
				fail(err)
			}
			fmt.Println(strings.Join(report.Lines(), "\n"))
		case (command == "fmt" || command == "check") && strings.HasSuffix(path, ".md"):
			d, docType, ok := declarationFor(declarations, path, false)
			if !ok || !inScope(path, docType, scope) {
				continue
			}
			matched++
			data, err := os.ReadFile(path)
			if err != nil {
				fail(err)
			}
			formatted, findings, err := d.FormatLinked(data, path, links.For(path))
			if err != nil {
				fail(err)
			}
			if command == "check" {
				for _, f := range d.Diagnose(data, path) {
					fmt.Println(f.String())
					problems++
				}
				if formatted != nil && !bytes.Equal(formatted, data) {
					fmt.Println(path + ": not-canonical: run mage fmt")
					problems++
				}
				continue
			}
			if formatted == nil {
				for _, f := range findings {
					fmt.Println(f.String())
				}
				problems++
				continue
			}
			if !bytes.Equal(formatted, data) {
				if err := os.WriteFile(path, formatted, 0o644); err != nil {
					fail(err)
				}
				fmt.Println("formatted " + path)
			}
		}
	}
	if matched == 0 {
		fail(fmt.Errorf("no corpus document matches %q", scope))
	}
	if problems > 0 {
		fail(fmt.Errorf("%d problem(s) in %d document(s)", problems, matched))
	}
}

// inScope reads a scope as all, a document type, a directory, or a path.
func inScope(path, documentType, scope string) bool {
	scope = strings.TrimSuffix(scope, "/")
	return scope == "all" || scope == documentType || path == scope || strings.HasPrefix(path, scope+"/")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "spectool: "+err.Error())
	os.Exit(1)
}
`
