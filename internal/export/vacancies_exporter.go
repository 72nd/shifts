package export

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/model"
)

// VacanciesExporter implements the Exporter interface for text format showing vacant positions
type VacanciesExporter struct{}

// NewVacanciesExporter creates a new vacancies exporter
func NewVacanciesExporter() *VacanciesExporter {
	return &VacanciesExporter{}
}

// Export converts ExportData to plain text format showing only shifts with vacancies
func (e *VacanciesExporter) Export(data *ExportData) ([]byte, error) {
	var output strings.Builder

	// Collect all vacant shifts organized by day
	type VacantShift struct {
		Title     string
		StartTime time.Time
		EndTime   time.Time
		Needed    int
	}

	// Map of date -> list of vacant shifts
	vacanciesByDay := make(map[model.Date][]VacantShift)

	// Iterate through all shift types and their days
	for _, shiftType := range data.ShiftTypes {
		for _, shiftDay := range shiftType.ShiftDays {
			for _, shift := range shiftDay.Shifts {
				// Only include shifts with vacancies (and Required > 0 to skip unlimited shifts)
				occupied := len(db.ActiveParticipants(shift.Participants))
				if shift.Required > 0 && occupied < shift.Required {
					needed := shift.Required - occupied
					endTime := shift.StartTime.Time.Add(time.Duration(shift.Duration) * time.Minute)

					vacantShift := VacantShift{
						Title:     shiftType.TitleEN,
						StartTime: shift.StartTime.Time,
						EndTime:   endTime,
						Needed:    needed,
					}

					vacanciesByDay[shiftDay.Date] = append(vacanciesByDay[shiftDay.Date], vacantShift)
				}
			}
		}
	}

	// Sort days
	var days []model.Date
	for day := range vacanciesByDay {
		days = append(days, day)
	}
	// Sort days chronologically
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j].Time) })

	// If no vacancies, return a message
	if len(days) == 0 {
		return []byte("No vacant shifts found.\n"), nil
	}

	// Format output
	for dayIdx, day := range days {
		// Format day header: "MO, 02.12.2025"
		weekday := day.Weekday().String()[:2] // Get first 2 letters (MO, TU, etc.)
		weekday = strings.ToUpper(weekday)
		output.WriteString(fmt.Sprintf("%s, %s\n\n", weekday, day.Format("02.01.2006")))

		// Get shifts for this day and sort by start time
		shifts := vacanciesByDay[day]
		sort.Slice(shifts, func(i, j int) bool { return shifts[i].StartTime.Before(shifts[j].StartTime) })

		// Format each shift
		for _, shift := range shifts {
			startStr := shift.StartTime.Format("15:04")
			endStr := shift.EndTime.Format("15:04")
			personStr := "person"
			if shift.Needed > 1 {
				personStr = "persons"
			}
			output.WriteString(fmt.Sprintf("%s, %s–%s: %d %s needed\n",
				shift.Title, startStr, endStr, shift.Needed, personStr))
		}

		// Add empty line between days (except after the last day)
		if dayIdx < len(days)-1 {
			output.WriteString("\n")
		}
	}

	return []byte(output.String()), nil
}

// ContentType returns the MIME type for plain text
func (e *VacanciesExporter) ContentType() string {
	return "text/plain; charset=utf-8"
}

// FileExtension returns the file extension for text files
func (e *VacanciesExporter) FileExtension() string {
	return "txt"
}

// Format returns the format identifier
func (e *VacanciesExporter) Format() string {
	return "vacancies"
}
