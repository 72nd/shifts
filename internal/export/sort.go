package export

import (
	"fmt"
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
