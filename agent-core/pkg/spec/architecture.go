// Copyright (c) 2026 Nokia
// SPDX-License-Identifier: BSD-3-Clause

package spec

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Architecture is the parsed ARCHITECTURE.md file, reduced to what the
// audit reads: the related_documents table, whose entries name files the
// corpus must hold (GH-2). The rest of the document is prose the checks
// do not interpret.
type Architecture struct {
	ID               string            `yaml:"id"`
	Title            string            `yaml:"title"`
	RelatedDocuments []RelatedDocument `yaml:"related_documents"`
}

// RelatedDocument is one row of ARCHITECTURE.md's related_documents table.
// Doc is a path relative to the corpus root.
type RelatedDocument struct {
	Doc     string `yaml:"doc"`
	Purpose string `yaml:"purpose"`
}

// loadArchitecture parses the architecture document. A corpus without one
// yields an empty model: the document is not required here, only read when
// present so its related_documents entries can be checked (GH-2). A present
// document that cannot load has its findings recorded by the format scan.
func loadArchitecture(scan *formatScan, file string) (Architecture, error) {
	path := scan.single(filepath.Join(scan.rootDir, file))
	if path == "" {
		return Architecture{}, nil
	}
	return ParseArchitecture(path)
}

// ParseArchitecture parses an architecture document from its file.
func ParseArchitecture(path string) (Architecture, error) {
	data, err := readDocument(path)
	if err != nil {
		return Architecture{}, fmt.Errorf("read architecture %s: %w", path, err)
	}
	return parseArchitectureBytes(data, path)
}

// parseArchitectureBytes decodes an architecture document from its YAML tree.
func parseArchitectureBytes(data []byte, path string) (Architecture, error) {
	var a Architecture
	if err := yaml.Unmarshal(data, &a); err != nil {
		return Architecture{}, fmt.Errorf("parse architecture %s: %w", path, err)
	}
	return a, nil
}
