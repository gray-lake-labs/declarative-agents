// Copyright (c) 2026 Nokia
// SPDX-License-Identifier: BSD-3-Clause

package spec

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validateIndexCorpus runs the checks that compare the corpus's indexes —
// SPECIFICATIONS.md's tables, the road map, and ARCHITECTURE.md's
// related_documents — against the documents and files they name.
func validateIndexCorpus(corpus *Corpus) []Finding {
	var findings []Finding
	findings = append(findings, checkUseCaseIndexRefs(corpus)...)
	findings = append(findings, checkTestSuiteIndexRefs(corpus)...)
	findings = append(findings, checkRoadmapUseCaseRefs(corpus)...)
	findings = append(findings, checkUseCaseTestSuiteReciprocity(corpus)...)
	findings = append(findings, checkTestCaseUseCaseRefs(corpus)...)
	findings = append(findings, checkSpecIndexPaths(corpus)...)
	findings = append(findings, checkArchitectureRelatedDocuments(corpus)...)
	findings = append(findings, checkRoadmapSummaryCoverage(corpus)...)
	findings = append(findings, checkIndexDocumentAgreement(corpus)...)
	return findings
}

func checkUseCaseIndexRefs(corpus *Corpus) []Finding {
	var findings []Finding
	for _, entry := range corpus.SpecIndex.UseCaseIndex {
		if _, ok := corpus.UseCases[entry.ID]; !ok {
			findings = append(findings, Finding{
				Check:   "index-missing-use-case",
				Level:   "error",
				Message: fmt.Sprintf("SPECIFICATIONS.md use_case_index references %q which does not exist", entry.ID),
			})
		}
	}
	return findings
}

// checkTestSuiteIndexRefs verifies that every test_suite_index entry
// references a test suite that exists in the corpus.

func checkTestSuiteIndexRefs(corpus *Corpus) []Finding {
	var findings []Finding
	for _, entry := range corpus.SpecIndex.TestSuiteIndex {
		if _, ok := corpus.TestSuites[entry.ID]; !ok {
			findings = append(findings, Finding{
				Check:   "index-missing-test-suite",
				Level:   "error",
				Message: fmt.Sprintf("SPECIFICATIONS.md test_suite_index references %q which does not exist", entry.ID),
			})
		}
	}
	return findings
}

// checkRoadmapUseCaseRefs verifies that use case IDs in roadmap releases
// reference use cases that exist in the corpus.

func checkRoadmapUseCaseRefs(corpus *Corpus) []Finding {
	var findings []Finding
	for _, rel := range corpus.Roadmap.Releases {
		for _, ucRef := range rel.UseCases {
			if _, ok := corpus.UseCases[ucRef.ID]; !ok {
				findings = append(findings, Finding{
					Check:   "roadmap-missing-use-case",
					Level:   "warning",
					Message: fmt.Sprintf("roadmap release %s references use case %q which does not exist", rel.Version, ucRef.ID),
				})
			}
		}
	}
	return findings
}

// checkUseCaseTestSuiteReciprocity verifies that use case test_suite
// fields reference test suites that exist, and that those test suites
// trace back to the use case.

func checkUseCaseTestSuiteReciprocity(corpus *Corpus) []Finding {
	var findings []Finding
	for _, ucID := range corpus.UCOrder {
		uc := corpus.UseCases[ucID]
		if uc.TestSuite == "" {
			continue
		}
		ts, ok := corpus.TestSuites[uc.TestSuite]
		if !ok {
			findings = append(findings, Finding{
				Check:   "use-case-missing-test-suite",
				Level:   "error",
				Message: fmt.Sprintf("use case %s references test_suite %q which does not exist", ucID, uc.TestSuite),
			})
			continue
		}
		tracesUC := false
		for _, trace := range ts.Traces {
			if trace == ucID {
				tracesUC = true
				break
			}
		}
		if !tracesUC {
			findings = append(findings, Finding{
				Check:   "test-suite-missing-uc-trace",
				Level:   "warning",
				Message: fmt.Sprintf("use case %s references test_suite %q but the suite does not trace back to it", ucID, uc.TestSuite),
			})
		}
	}
	return findings
}

// checkTestCaseUseCaseRefs verifies that test case use_case fields
// reference use cases that exist in the corpus.

func checkTestCaseUseCaseRefs(corpus *Corpus) []Finding {
	var findings []Finding
	for _, ts := range corpus.TestSuites {
		for _, tc := range ts.TestCases {
			if tc.UseCase == "" {
				continue
			}
			if _, ok := corpus.UseCases[tc.UseCase]; !ok {
				findings = append(findings, Finding{
					Check:   "test-case-missing-use-case",
					Level:   "warning",
					Message: fmt.Sprintf("test case %s:%s references use_case %q which does not exist", ts.ID, tc.Name, tc.UseCase),
				})
			}
		}
	}
	return findings
}

// checkSpecIndexPaths verifies that path fields in spec index entries
// point to existing files. Every index table SPECIFICATIONS.md carries is
// covered, the document indexes included: a foundation document deleted or
// never written used to pass because its index section was not parsed, and
// the formatter quietly unlinks an entry whose target is missing (GH-2).

func checkSpecIndexPaths(corpus *Corpus) []Finding {
	if corpus.RootDir == "" {
		return nil
	}
	var findings []Finding
	for _, index := range corpus.SpecIndex.pathIndexes() {
		findings = append(findings, documentIndexBrokenPaths(corpus, index.name, index.entries)...)
	}
	return findings
}

// pathIndex is one of SPECIFICATIONS.md's index tables reduced to what the
// path check reads: the table's field name and its id/path entries.
type pathIndex struct {
	name    string
	entries []DocumentIndexEntry
}

// pathIndexes lists every index table whose entries carry a path, each
// reduced to id/path pairs for the broken-path check.
func (si *SpecIndex) pathIndexes() []pathIndex {
	srds := make([]DocumentIndexEntry, len(si.SRDIndex))
	for i, e := range si.SRDIndex {
		srds[i] = DocumentIndexEntry{ID: e.ID, Path: e.Path}
	}
	ucs := make([]DocumentIndexEntry, len(si.UseCaseIndex))
	for i, e := range si.UseCaseIndex {
		ucs[i] = DocumentIndexEntry{ID: e.ID, Path: e.Path}
	}
	tss := make([]DocumentIndexEntry, len(si.TestSuiteIndex))
	for i, e := range si.TestSuiteIndex {
		tss[i] = DocumentIndexEntry{ID: e.ID, Path: e.Path}
	}
	return []pathIndex{
		{"foundation_document_index", si.FoundationIndex},
		{"srd_index", srds},
		{"config_format_index", si.ConfigFormatIndex},
		{"semantic_model_index", si.SemanticModelIndex},
		{"use_case_index", ucs},
		{"test_suite_index", tss},
	}
}

// documentIndexBrokenPaths reports the entries of one document index table
// whose path does not resolve, whether or not the entry is written as a link.
func documentIndexBrokenPaths(corpus *Corpus, index string, entries []DocumentIndexEntry) []Finding {
	var findings []Finding
	for _, entry := range entries {
		if entry.Path == "" || fileExists(filepath.Join(corpus.RootDir, entry.Path)) {
			continue
		}
		findings = append(findings, Finding{
			Check:   "index-broken-path",
			Level:   "error",
			Message: fmt.Sprintf("%s entry %s path %q does not exist", index, entry.ID, entry.Path),
		})
	}
	return findings
}

// checkArchitectureRelatedDocuments verifies that every related_documents
// entry in ARCHITECTURE.md names a file that exists. The doc column is a
// path relative to the corpus root; an entry for a deleted or never-written
// document raised no finding before, because nothing parsed the table (GH-2).
func checkArchitectureRelatedDocuments(corpus *Corpus) []Finding {
	if corpus.RootDir == "" {
		return nil
	}
	var findings []Finding
	for _, ref := range corpus.Architecture.RelatedDocuments {
		if ref.Doc == "" || fileExists(filepath.Join(corpus.RootDir, ref.Doc)) {
			continue
		}
		findings = append(findings, Finding{
			Check:   "architecture-broken-related-document",
			Level:   "error",
			Message: fmt.Sprintf("ARCHITECTURE.md related_documents entry %q does not exist", ref.Doc),
		})
	}
	return findings
}

// checkDocSpecRequirementsSources verifies that canonical requirement
// source paths in doc specs point to existing files.

func checkDocSpecRequirementsSources(corpus *Corpus) []Finding {
	if corpus.RootDir == "" {
		return nil
	}
	var findings []Finding
	for id, ds := range corpus.DocSpecs {
		for _, path := range ds.RequirementsSource.Canonical {
			if !fileExists(filepath.Join(corpus.RootDir, path)) {
				findings = append(findings, Finding{
					Check:   "docspec-broken-requirement-source",
					Level:   "error",
					Message: fmt.Sprintf("doc spec %s canonical requirements_source %q does not exist", id, path),
				})
			}
		}
	}
	return findings
}

// checkDocSpecRelatedDocuments verifies that related_documents IDs
// resolve to known SRDs or doc specs.

func checkDocSpecRelatedDocuments(corpus *Corpus) []Finding {
	knownIDs := make(map[string]bool)
	for srdID := range corpus.SRDs {
		knownIDs[srdID] = true
		parts := strings.SplitN(srdID, "-", 2)
		if len(parts) > 0 {
			knownIDs[parts[0]] = true
		}
	}
	for dsID := range corpus.DocSpecs {
		knownIDs[dsID] = true
	}

	var findings []Finding
	for id, ds := range corpus.DocSpecs {
		for _, ref := range ds.RelatedDocuments {
			if !knownIDs[ref] {
				findings = append(findings, Finding{
					Check:   "docspec-broken-related-document",
					Level:   "warning",
					Message: fmt.Sprintf("doc spec %s related_documents references %q which is not a known SRD or spec", id, ref),
				})
			}
		}
	}
	return findings
}

// checkDocSpecImplementationPaths verifies that implementation file
// paths in doc specs exist on disk.

func checkDocSpecImplementationPaths(corpus *Corpus) []Finding {
	if corpus.RootDir == "" {
		return nil
	}
	var findings []Finding
	for id, ds := range corpus.DocSpecs {
		for _, path := range ds.Implementation.Paths {
			if !fileExists(filepath.Join(corpus.RootDir, path)) {
				findings = append(findings, Finding{
					Check:   "docspec-broken-implementation-path",
					Level:   "error",
					Message: fmt.Sprintf("doc spec %s implementation path %q does not exist", id, path),
				})
			}
		}
	}
	return findings
}

// checkDocSpecExamplePaths verifies that example file paths in doc specs
// exist on disk.

func checkDocSpecExamplePaths(corpus *Corpus) []Finding {
	if corpus.RootDir == "" {
		return nil
	}
	var findings []Finding
	for id, ds := range corpus.DocSpecs {
		for _, ex := range ds.Examples {
			if ex.File == "" {
				continue
			}
			if !fileExists(filepath.Join(corpus.RootDir, ex.File)) {
				findings = append(findings, Finding{
					Check:   "docspec-broken-example-path",
					Level:   "warning",
					Message: fmt.Sprintf("doc spec %s example file %q does not exist", id, ex.File),
				})
			}
		}
	}
	return findings
}

// checkMachineDiagnostics runs core.DiagnoseMachineSpec on each loaded
// machine and surfaces its diagnostics as findings.

// checkRoadmapSummaryCoverage verifies that every release in road-map.md has
// a row in SPECIFICATIONS.md's roadmap_summary. The two are maintained by
// hand, and a release that reaches the road map without reaching the summary
// leaves the index quietly describing a smaller project than the one that
// exists (GH-2310).
func checkRoadmapSummaryCoverage(corpus *Corpus) []Finding {
	if len(corpus.Roadmap.Releases) == 0 {
		return nil
	}
	summarized := make(map[string]bool, len(corpus.SpecIndex.RoadmapSummary))
	for _, entry := range corpus.SpecIndex.RoadmapSummary {
		summarized[entry.Version] = true
	}
	var findings []Finding
	for _, release := range corpus.Roadmap.Releases {
		if release.Version == "" || summarized[release.Version] {
			continue
		}
		findings = append(findings, Finding{
			Check: "roadmap-summary-missing-release",
			Level: "error",
			Message: fmt.Sprintf(
				"road-map.md release %s has no roadmap_summary row in SPECIFICATIONS.md",
				release.Version),
		})
	}
	return findings
}

// checkIndexDocumentAgreement verifies that what SPECIFICATIONS.md says
// about a use case or test suite matches what that document says about
// itself. An index is only useful while it agrees with what it indexes, and
// nothing else compares the two (GH-2310).
func checkIndexDocumentAgreement(corpus *Corpus) []Finding {
	var findings []Finding
	for _, entry := range corpus.SpecIndex.UseCaseIndex {
		document, ok := corpus.UseCases[entry.ID]
		if !ok {
			continue // index-missing-use-case reports this
		}
		findings = append(findings,
			indexFieldMismatch("use_case_index", entry.ID, "title", entry.Title, document.Title)...)
		findings = append(findings,
			indexFieldMismatch("use_case_index", entry.ID, "status", entry.Status, document.Status)...)
	}
	for _, entry := range corpus.SpecIndex.TestSuiteIndex {
		document, ok := corpus.TestSuites[entry.ID]
		if !ok {
			continue // index-missing-test-suite reports this
		}
		findings = append(findings,
			indexFieldMismatch("test_suite_index", entry.ID, "title", entry.Title, document.Title)...)
		findings = append(findings,
			indexFieldMismatch("test_suite_index", entry.ID, "release", entry.Release, document.Release)...)
	}
	return findings
}

// indexFieldMismatch reports one disagreeing field. Silence on either side is
// not disagreement: an index may summarize less than the document states, and
// a document that omits a field is not contradicting the index that carries
// one. Only two present and different values are a mismatch; the checks that
// require an entry to exist at all are separate.
func indexFieldMismatch(index, id, field, indexed, documented string) []Finding {
	indexed, documented = strings.TrimSpace(indexed), strings.TrimSpace(documented)
	if indexed == "" || documented == "" || indexed == documented {
		return nil
	}
	return []Finding{{
		Check: "index-document-mismatch",
		Level: "error",
		Message: fmt.Sprintf(
			"SPECIFICATIONS.md %s entry %s states %s %q but the document states %q",
			index, id, field, indexed, documented),
	}}
}
