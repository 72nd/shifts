package export

import (
	"fmt"
	"sort"
	"time"

	"github.com/72nd/shifts/internal/db"
)

// ShiftTime represents a parsed time with hours and minutes
type ShiftTime struct {
	Hours   int
	Minutes int
	Total   int // Total minutes since midnight
}

// ParseTime parses a time string in format "HH:MM". If the timeStr is empty,
// it returns a ShiftTime with 0 hours and 0 minutes.
func ParseTime(timeStr string) (ShiftTime, error) {
	if timeStr == "" {
		return ShiftTime{
			Hours:   0,
			Minutes: 0,
			Total:   0,
		}, nil
	}

	var hours, minutes int
	if _, err := fmt.Sscanf(timeStr, "%d:%d", &hours, &minutes); err != nil {
		return ShiftTime{}, fmt.Errorf("invalid time format (expected HH:MM): %w", err)
	}

	if hours < 0 || hours > 23 || minutes < 0 || minutes > 59 {
		return ShiftTime{}, fmt.Errorf("invalid time values: hours must be 0-23, minutes must be 0-59")
	}

	return ShiftTime{
		Hours:   hours,
		Minutes: minutes,
		Total:   hours*60 + minutes,
	}, nil
}

// getShiftDay calculates the logical day of a shift based on the boundary time
// For example, if boundary time is "04:00", a shift at 02:00 belongs to the previous day
func getShiftDay(shift db.Shift, boundaryTime string) (time.Time, error) {
	// Parse boundary time
	boundary, err := ParseTime(boundaryTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid boundary time: %w", err)
	}

	// Get shift time components
	shiftTime := ShiftTime{
		Hours:   shift.StartTime.Hour(),
		Minutes: shift.StartTime.Minute(),
		Total:   shift.StartTime.Hour()*60 + shift.StartTime.Minute(),
	}

	// Calculate logical day
	shiftDay := shift.StartTime.Truncate(24 * time.Hour)
	if shiftTime.Total < boundary.Total {
		// If shift is before boundary, it belongs to the previous day
		shiftDay = shiftDay.AddDate(0, 0, -1)
	}

	return shiftDay, nil
}

// sortShiftsByDay sorts shifts first by their logical day (considering boundary time)
// and then by their start time within each day
func sortShiftsByDay(shifts []db.Shift, boundaryTime string) ([]db.Shift, error) {
	// Validate boundary time format
	if _, err := ParseTime(boundaryTime); err != nil {
		return nil, fmt.Errorf("invalid boundary time format: %w", err)
	}

	sort.Slice(shifts, func(i, j int) bool {
		dayI, errI := getShiftDay(shifts[i], boundaryTime)
		dayJ, errJ := getShiftDay(shifts[j], boundaryTime)

		// If we encounter any errors, maintain original order
		if errI != nil || errJ != nil {
			return i < j
		}

		// First compare days
		if !dayI.Equal(dayJ) {
			return dayI.Before(dayJ)
		}

		// If same day, compare start times
		return shifts[i].StartTime.Before(shifts[j].StartTime)
	})

	return shifts, nil
}

// SortShiftsInType sorts all shifts within a shift type according to the plan's day boundary time
func SortShiftsInType(shiftType *db.ShiftType, boundaryTime string) error {
	if shiftType.Shifts == nil {
		return nil
	}

	sorted, err := sortShiftsByDay(shiftType.Shifts, boundaryTime)
	if err != nil {
		return fmt.Errorf("failed to sort shifts: %w", err)
	}

	shiftType.Shifts = sorted
	return nil
}
