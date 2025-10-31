package export

import (
	"encoding/json"
)

// JSONExporter implements the Exporter interface for JSON format
type JSONExporter struct{}

// NewJSONExporter creates a new JSON exporter
func NewJSONExporter() *JSONExporter {
	return &JSONExporter{}
}

// Export converts ExportData to JSON format with pretty printing
func (e *JSONExporter) Export(data *ExportData) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

// ContentType returns the MIME type for JSON
func (e *JSONExporter) ContentType() string {
	return "application/json"
}

// FileExtension returns the file extension for JSON
func (e *JSONExporter) FileExtension() string {
	return "json"
}

// Format returns the format identifier
func (e *JSONExporter) Format() string {
	return "json"
}
