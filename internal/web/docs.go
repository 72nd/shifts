package web

import (
	"fmt"
	"io/fs"
)

// documentationManager handles markdown documentation rendering and caching
type documentationManager struct {
	docs   map[string]DocumentContent
	logger Logger
}

// newDocumentationManager creates a new documentation manager
func newDocumentationManager(logger Logger) *documentationManager {
	return &documentationManager{
		docs:   make(map[string]DocumentContent),
		logger: logger,
	}
}

// parseDocumentation parses all markdown documentation files and caches them
func (dm *documentationManager) parseDocumentation() error {
	// Define the documentation files to parse
	docFiles := []string{
		"docs/help-en.md",
		"docs/help-de.md",
	}

	for _, docPath := range docFiles {
		// Read markdown file from embedded filesystem
		data, err := fs.ReadFile(content, docPath)
		if err != nil {
			return fmt.Errorf("failed to read documentation file %s: %w", docPath, err)
		}

		// Render markdown with TOC
		rendered := renderMarkdownWithTOC(string(data))

		// Extract language from filename (help-en.md -> en)
		lang := docPath[len(docPath)-5 : len(docPath)-3]

		// Cache the rendered documentation
		dm.docs[lang] = rendered
	}

	return nil
}

// getDocumentation returns the cached documentation for the given language
func (dm *documentationManager) getDocumentation(lang string) (DocumentContent, bool) {
	doc, ok := dm.docs[lang]
	return doc, ok
}

