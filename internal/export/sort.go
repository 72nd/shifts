package export

import (
	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/model"
)

// getShiftDay calculates the logical day of a shift based on the boundary
// time ("HH:MM"). For example, with boundary "04:00" a shift at 02:00 belongs
// to the previous day.
func getShiftDay(shift db.Shift, boundaryTime string) (model.Date, error) {
	boundary, err := model.ParseBoundary(boundaryTime)
	if err != nil {
		return model.Date{}, err
	}
	return shift.StartTime.LogicalDay(boundary), nil
}
