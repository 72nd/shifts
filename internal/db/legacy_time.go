package db

import (
	"fmt"
	"time"

	"github.com/72nd/shifts/internal/model"
)

// Legacy time parsers. They are used only by migration 0002_local_times to
// convert values written by v0.6.0 and older (RFC3339 instants). Regular code
// uses model.ParseDateTime and model.ParseDate, which reject offsets.

// parseLegacyDateTime reads a stored shift start. A value without offset is
// already in the new format and returned unchanged (changed == false). An
// RFC3339 value (offset, "Z", fractional seconds allowed) is converted to the
// wall clock in loc.
func parseLegacyDateTime(s string, loc *time.Location) (dt model.DateTime, changed bool, err error) {
	if v, perr := model.ParseDateTime(s); perr == nil {
		return v, false, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return model.DateTime{}, false, fmt.Errorf("neither local date-time nor RFC3339: %w", err)
	}
	return model.NewDateTime(t.In(loc)), true, nil
}

// parseLegacyDate reads a stored plan date. A plain YYYY-MM-DD value is
// returned unchanged. An RFC3339 value maps to the date of t.UTC()+12h, which
// yields the intended day for both UTC midnight and local midnight within
// +-12h.
func parseLegacyDate(s string) (d model.Date, changed bool, err error) {
	if v, perr := model.ParseDate(s); perr == nil {
		return v, false, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return model.Date{}, false, fmt.Errorf("neither date nor RFC3339: %w", err)
	}
	return model.NewDate(t.UTC().Add(12 * time.Hour)), true, nil
}
