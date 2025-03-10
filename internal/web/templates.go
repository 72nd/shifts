package web

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
)

// templateManager handles all template-related functionality
type templateManager struct {
	templates map[string]*template.Template
	logger    Logger
}

// newTemplateManager creates a new template manager
func newTemplateManager(logger Logger) *templateManager {
	return &templateManager{
		templates: make(map[string]*template.Template),
		logger:    logger,
	}
}

// parseTemplates parses all templates and stores them in the templates map
func (tm *templateManager) parseTemplates() error {
	// Get all template files
	pages, err := fs.Glob(content, "templates/*.html")
	if err != nil {
		return fmt.Errorf("failed to get template files: %w", err)
	}

	// Get all partials
	partials, err := fs.Glob(content, "templates/partials/*.html")
	if err != nil {
		return fmt.Errorf("failed to get partial files: %w", err)
	}

	// Get all help partials
	helpPartials, err := fs.Glob(content, "templates/help-partials/*.html")
	if err != nil {
		return fmt.Errorf("failed to get help partial files: %w", err)
	}

	// Parse each template
	for _, page := range pages {
		name := filepath.Base(page)

		// Create a new template with functions
		tmpl := template.New(name)

		// Parse base layout and all partials first
		patterns := append([]string{"templates/layouts/base.html"}, partials...)
		patterns = append(patterns, helpPartials...)
		if _, err := tmpl.ParseFS(content, patterns...); err != nil {
			return fmt.Errorf("failed to parse base, partials, and help partials for %s: %w", name, err)
		}

		// Then parse the page template
		if _, err := tmpl.ParseFS(content, page); err != nil {
			return fmt.Errorf("failed to parse template %s: %w", name, err)
		}

		tm.templates[name] = tmpl
	}

	return nil
}

// render renders a template with the given name and data
func (tm *templateManager) render(w http.ResponseWriter, name string, data interface{}) {
	// Get template from cache
	ts, ok := tm.templates[name]
	if !ok {
		tm.logger.Error("template not found", "name", name)
		RespondWithErrorAndLogger(w, tm.logger, NewAppError(
			ErrInternalError,
			"Internal Server Error",
			http.StatusInternalServerError,
			fmt.Errorf("template %s not found", name),
		))
		return
	}

	// Execute template
	if err := ts.Execute(w, data); err != nil {
		tm.logger.Error("failed to execute template", "error", err, "name", name)
		RespondWithErrorAndLogger(w, tm.logger, NewAppError(
			ErrInternalError,
			"Internal Server Error",
			http.StatusInternalServerError,
			err,
		))
	}
}

// Logger interface for template manager
type Logger interface {
	Error(msg string, args ...interface{})
}
