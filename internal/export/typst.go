package export

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed static
var static embed.FS

// Semaphore to limit concurrent Typst executions to 10
var typstSemaphore = make(chan struct{}, 10)

// acquireSemaphore acquires a slot in the semaphore (blocking if at capacity)
func acquireSemaphore() {
	typstSemaphore <- struct{}{}
}

// releaseSemaphore releases a slot in the semaphore
func releaseSemaphore() {
	<-typstSemaphore
}

// TypstExporter is a base exporter for Typst-based PDF generation
type TypstExporter struct {
	template    string // Template filename (e.g., "timetable.typ")
	format      string // Format identifier (e.g., "timetable")
	displayName string // Display name for errors
}

// NewTypstExporter creates a new Typst exporter with the specified template and format
func NewTypstExporter(template, format, displayName string) *TypstExporter {
	return &TypstExporter{
		template:    template,
		format:      format,
		displayName: displayName,
	}
}

// Export converts ExportData to PDF using Typst
func (e *TypstExporter) Export(data *ExportData) ([]byte, error) {
	// Acquire semaphore slot (blocks if 10 are already running)
	acquireSemaphore()
	defer releaseSemaphore()

	// Create temporary directory for this export
	tempDir, err := os.MkdirTemp("", "typst-export-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write JSON data to temp file
	jsonPath := filepath.Join(tempDir, "data.json")
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal export data: %w", err)
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write JSON data: %w", err)
	}

	// Extract all embedded .typ files and assets to temp directory
	entries, err := static.ReadDir("static")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded static files: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := static.ReadFile(filepath.Join("static", entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded file %s: %w", entry.Name(), err)
		}
		targetPath := filepath.Join(tempDir, entry.Name())
		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", entry.Name(), err)
		}
	}

	// Prepare output path
	outputPath := filepath.Join(tempDir, "output.pdf")

	// Run Typst compile command
	// Use relative paths since we're running from tempDir
	cmd := exec.Command("typst", "compile", 
		"--input", "data=data.json",
		e.template,
		"output.pdf",
	)
	cmd.Dir = tempDir

	// Capture output for error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("typst compilation failed for %s: %w\nOutput: %s", e.displayName, err, string(output))
	}

	// Read generated PDF
	pdfData, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read generated PDF: %w", err)
	}

	return pdfData, nil
}

// ContentType returns the MIME type for PDF
func (e *TypstExporter) ContentType() string {
	return "application/pdf"
}

// FileExtension returns the file extension for PDF
func (e *TypstExporter) FileExtension() string {
	return "pdf"
}

// Format returns the format identifier
func (e *TypstExporter) Format() string {
	return e.format
}

// TimetableExporter exports shift plans as a timetable PDF
type TimetableExporter struct {
	*TypstExporter
}

// NewTimetableExporter creates a new timetable exporter
func NewTimetableExporter() *TimetableExporter {
	return &TimetableExporter{
		TypstExporter: NewTypstExporter("timetable.typ", "timetable", "Timetable"),
	}
}

// ReportExporter exports shift plans as a report PDF
type ReportExporter struct {
	*TypstExporter
}

// NewReportExporter creates a new report exporter
func NewReportExporter() *ReportExporter {
	return &ReportExporter{
		TypstExporter: NewTypstExporter("report.typ", "report", "Report"),
	}
}

// PosterExporter exports shift plans as a poster PDF
type PosterExporter struct {
	*TypstExporter
}

// NewPosterExporter creates a new poster exporter
func NewPosterExporter() *PosterExporter {
	return &PosterExporter{
		TypstExporter: NewTypstExporter("poster.typ", "poster", "Poster"),
	}
}
