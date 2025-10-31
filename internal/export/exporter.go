package export

// Exporter defines the interface for all export formats
type Exporter interface {
	// Export converts ExportData into the target format
	Export(data *ExportData) ([]byte, error)

	// ContentType returns the MIME type for HTTP headers
	ContentType() string

	// FileExtension returns the file extension (e.g., "json", "pdf", "ics")
	FileExtension() string

	// Format returns the format identifier (e.g., "json", "pdf", "ical")
	Format() string
}
