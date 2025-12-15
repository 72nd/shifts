package web

import (
	"embed"
	"log/slog"
)

//go:embed docs/*.md
var docsFS embed.FS

// documentationManager manages user documentation (help pages)
type documentationManager struct {
	logger        *slog.Logger
	documentation map[string]DocumentContent
}

// newDocumentationManager creates a new documentation manager
func newDocumentationManager(logger *slog.Logger) *documentationManager {
	return &documentationManager{
		logger:        logger,
		documentation: make(map[string]DocumentContent),
	}
}

// parseDocumentation loads and parses all documentation files
func (dm *documentationManager) parseDocumentation() error {
	// Load English documentation
	enContent, err := docsFS.ReadFile("docs/help-en.md")
	if err != nil {
		return err
	}
	dm.documentation["en"] = renderMarkdownWithTOC(string(enContent))

	// Load German documentation
	deContent, err := docsFS.ReadFile("docs/help-de.md")
	if err != nil {
		return err
	}
	dm.documentation["de"] = renderMarkdownWithTOC(string(deContent))

	dm.logger.Info("documentation loaded", "languages", []string{"en", "de"})
	return nil
}

// getDocumentation retrieves documentation for a specific language
func (dm *documentationManager) getDocumentation(lang string) (DocumentContent, bool) {
	doc, ok := dm.documentation[lang]
	return doc, ok
}
