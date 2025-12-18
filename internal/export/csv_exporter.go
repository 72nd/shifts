package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"
)

// CSVExporter implements the Exporter interface for CSV format
type CSVExporter struct{}

// NewCSVExporter creates a new CSV exporter
func NewCSVExporter() *CSVExporter {
	return &CSVExporter{}
}

// Export converts ExportData to CSV format
func (e *CSVExporter) Export(data *ExportData) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write CSV header
	header := []string{
		"id",
		"type_id",
		"type_title_de",
		"type_title_en",
		"start_time",
		"duration",
		"required",
		"comment_de",
		"comment_en",
		"is_locked",
		"participants_count",
		"participants",
	}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Iterate through all shift types and their days
	for _, shiftType := range data.ShiftTypes {
		for _, shiftDay := range shiftType.ShiftDays {
			for _, shift := range shiftDay.Shifts {
				// Collect participant names
				participantNames := make([]string, len(shift.Participants))
				for i, p := range shift.Participants {
					participantNames[i] = p.Name
				}
				participantsStr := ""
				if len(participantNames) > 0 {
					// Join participant names with semicolon to avoid conflicts with CSV commas
					participantsStr = joinWithSemicolon(participantNames)
				}

				// Convert boolean to string
				isLockedStr := "false"
				if shift.IsLocked {
					isLockedStr = "true"
				}

				// Format start time as RFC3339
				startTimeStr := shift.StartTime.Format(time.RFC3339)

				// Write CSV row
				row := []string{
					strconv.FormatInt(shift.ID, 10),
					strconv.FormatInt(shift.TypeID, 10),
					shiftType.TitleDE,
					shiftType.TitleEN,
					startTimeStr,
					strconv.Itoa(shift.Duration),
					strconv.Itoa(shift.Required),
					shift.CommentDE,
					shift.CommentEN,
					isLockedStr,
					strconv.Itoa(len(shift.Participants)),
					participantsStr,
				}
				if err := writer.Write(row); err != nil {
					return nil, fmt.Errorf("failed to write CSV row: %w", err)
				}
			}
		}
	}

	// Flush the writer to ensure all data is written
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("failed to flush CSV writer: %w", err)
	}

	return buf.Bytes(), nil
}

// joinWithSemicolon joins strings with semicolon separator
// This avoids conflicts with CSV comma separators
func joinWithSemicolon(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += ";" + strs[i]
	}
	return result
}

// ContentType returns the MIME type for CSV
func (e *CSVExporter) ContentType() string {
	return "text/csv; charset=utf-8"
}

// FileExtension returns the file extension for CSV files
func (e *CSVExporter) FileExtension() string {
	return "csv"
}

// Format returns the format identifier
func (e *CSVExporter) Format() string {
	return "csv"
}
