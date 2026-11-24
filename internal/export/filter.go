package export

import (
	"strings"

	"github.com/72nd/shifts/internal/db"
)

// ExportFilter defines the filter criteria for exports
type ExportFilter struct {
	TypeIDs     []int64 // List of shift type IDs to include (empty = all)
	Day         string  // ISO date for day filter (empty = all days)
	Status      string  // Status filter: "all", "empty", "critical", "not_full", "full", "overbooked"
	SearchQuery string  // Search query for participant names (empty = no search)
}

// ApplyFilters applies the filter criteria to the export data
// Returns a new ExportData with filtered shift types and shifts
func (f *ExportFilter) ApplyFilters(data *ExportData) *ExportData {
	if data == nil {
		return nil
	}

	// Create a copy of the export data
	filtered := &ExportData{
		Plan:       data.Plan,
		ShiftTypes: make([]*ShiftTypeWithDays, 0),
		ExportedAt: data.ExportedAt,
	}

	// Filter shift types by TypeIDs (if specified)
	for _, shiftType := range data.ShiftTypes {
		if f.shouldIncludeType(shiftType.ID) {
			filteredType := &ShiftTypeWithDays{
				ShiftType: shiftType.ShiftType,
				ShiftDays: f.filterShiftDays(shiftType.ShiftDays),
			}

			// Only include type if it has shifts after filtering
			if len(filteredType.ShiftDays) > 0 {
				filtered.ShiftTypes = append(filtered.ShiftTypes, filteredType)
			}
		}
	}

	return filtered
}

// shouldIncludeType checks if a type ID should be included based on TypeIDs filter
func (f *ExportFilter) shouldIncludeType(typeID int64) bool {
	// If no type filter is set, include all types
	if len(f.TypeIDs) == 0 {
		return true
	}

	// Check if typeID is in the filter list
	for _, id := range f.TypeIDs {
		if id == typeID {
			return true
		}
	}
	return false
}

// filterShiftDays filters the shift days based on the filter criteria
func (f *ExportFilter) filterShiftDays(shiftDays []ShiftsByDay) []ShiftsByDay {
	if len(shiftDays) == 0 {
		return []ShiftsByDay{}
	}

	filtered := make([]ShiftsByDay, 0)

	for _, day := range shiftDays {
		// Apply day filter
		if f.Day != "" {
			dayStr := day.Date.String()
			if dayStr != f.Day {
				continue
			}
		}

		// Filter shifts within this day
		filteredShifts := f.filterShifts(day.Shifts)
		if len(filteredShifts) > 0 {
			filtered = append(filtered, ShiftsByDay{
				Date:     day.Date,
				DayCount: day.DayCount,
				Shifts:   filteredShifts,
			})
		}
	}

	return filtered
}

// filterShifts filters shifts based on status and search query
func (f *ExportFilter) filterShifts(shifts []db.Shift) []db.Shift {
	if len(shifts) == 0 {
		return []db.Shift{}
	}

	filtered := make([]db.Shift, 0)

	for _, shift := range shifts {
		if f.shouldIncludeShift(shift) {
			filtered = append(filtered, shift)
		}
	}

	return filtered
}

// shouldIncludeShift checks if a shift should be included based on status and search filters
func (f *ExportFilter) shouldIncludeShift(shift db.Shift) bool {
	// Apply status filter
	if !f.matchesStatusFilter(shift) {
		return false
	}

	// Apply search query filter
	if !f.matchesSearchQuery(shift) {
		return false
	}

	return true
}

// matchesStatusFilter checks if a shift matches the status filter
func (f *ExportFilter) matchesStatusFilter(shift db.Shift) bool {
	if f.Status == "" || f.Status == "all" {
		return true
	}

	participantCount := len(shift.Participants)
	percentage := 0.0
	if shift.Required > 0 {
		percentage = (float64(participantCount) / float64(shift.Required)) * 100
	}

	switch f.Status {
	case "empty":
		return participantCount == 0
	case "critical":
		return participantCount > 0 && percentage < 50 && participantCount < shift.Required
	case "not_full":
		return participantCount < shift.Required
	case "full":
		return participantCount == shift.Required
	case "overbooked":
		// Overbooked: more participants than required slots
		// Exclude unlimited shifts (Required <= 0)
		return shift.Required > 0 && participantCount > shift.Required
	default:
		return true
	}
}

// matchesSearchQuery checks if a shift matches the search query
// Returns true if search query is empty or if any participant name contains the query
func (f *ExportFilter) matchesSearchQuery(shift db.Shift) bool {
	query := strings.TrimSpace(f.SearchQuery)
	if query == "" {
		return true
	}

	queryLower := strings.ToLower(query)

	// Search in participant names
	for _, participant := range shift.Participants {
		if strings.Contains(strings.ToLower(participant.Name), queryLower) {
			return true
		}
	}

	return false
}

// IsEmpty returns true if no filters are set
func (f *ExportFilter) IsEmpty() bool {
	return len(f.TypeIDs) == 0 &&
		f.Day == "" &&
		(f.Status == "" || f.Status == "all") &&
		strings.TrimSpace(f.SearchQuery) == ""
}
