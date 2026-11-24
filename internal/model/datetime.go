package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	// DateTimeLayout is the storage and JSON layout of a DateTime.
	DateTimeLayout = "2006-01-02T15:04:05"
	// DateLayout is the storage and JSON layout of a Date.
	DateLayout = "2006-01-02"
)

// DateTime is a wall-clock time at the event location without a zone
// ("floating" local time). The embedded time.Time ALWAYS has Location
// time.UTC; its fields are the wall clock. Never call .In(), .Local() or
// compare it with time.Now(); use NowIn instead.
type DateTime struct{ time.Time }

// Date is a calendar day; the embedded time is midnight UTC.
type Date struct{ time.Time }

var (
	defaultLocMu sync.RWMutex
	defaultLoc   *time.Location
)

// SetDefaultLocation sets the default time zone for new plans and for the
// migration of existing plans. Called once in main.
func SetDefaultLocation(loc *time.Location) {
	defaultLocMu.Lock()
	defer defaultLocMu.Unlock()
	defaultLoc = loc
}

// DefaultLocation returns the default zone (Europe/Berlin if never set).
func DefaultLocation() *time.Location {
	defaultLocMu.RLock()
	loc := defaultLoc
	defaultLocMu.RUnlock()
	if loc != nil {
		return loc
	}
	if l, err := time.LoadLocation("Europe/Berlin"); err == nil {
		return l
	}
	return time.UTC
}

// NewDateTime takes t's wall-clock fields and drops the zone.
func NewDateTime(t time.Time) DateTime {
	return DateTime{time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC)}
}

// NewDate takes t's calendar day and drops the zone.
func NewDate(t time.Time) Date {
	return Date{time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)}
}

// NowIn returns the current wall clock in the given zone.
func NowIn(loc *time.Location) DateTime {
	return NewDateTime(time.Now().In(loc))
}

// ParseDateTime parses "2006-01-02T15:04" or "2006-01-02T15:04:05" (also with
// a space instead of 'T'). Offsets, "Z" and fractional seconds are rejected.
func ParseDateTime(s string) (DateTime, error) {
	norm := s
	if len(norm) > 10 && norm[10] == ' ' {
		norm = norm[:10] + "T" + norm[11:]
	}
	layout := "2006-01-02T15:04"
	switch len(norm) {
	case len("2006-01-02T15:04"):
	case len("2006-01-02T15:04:05"):
		layout = DateTimeLayout
	default:
		return DateTime{}, fmt.Errorf("invalid date-time %q: want YYYY-MM-DDTHH:MM[:SS] without offset", s)
	}
	t, err := time.Parse(layout, norm)
	if err != nil {
		return DateTime{}, fmt.Errorf("invalid date-time %q: want YYYY-MM-DDTHH:MM[:SS] without offset", s)
	}
	return DateTime{t}, nil
}

// ParseDate parses "2006-01-02" only.
func ParseDate(s string) (Date, error) {
	if len(s) != len(DateLayout) {
		return Date{}, fmt.Errorf("invalid date %q: want YYYY-MM-DD", s)
	}
	t, err := time.Parse(DateLayout, s)
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q: want YYYY-MM-DD", s)
	}
	return Date{t}, nil
}

// ParseBoundary converts "HH:MM" to minutes since midnight; "" yields 0.
func ParseBoundary(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	if len(s) != 5 || s[2] != ':' {
		return 0, fmt.Errorf("invalid time %q: want HH:MM", s)
	}
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, fmt.Errorf("invalid time %q: want HH:MM", s)
	}
	return t.Hour()*60 + t.Minute(), nil
}

// String returns the DateTimeLayout representation.
func (d DateTime) String() string { return d.Time.Format(DateTimeLayout) }

// AddMinutes returns d shifted by m minutes on the wall clock.
func (d DateTime) AddMinutes(m int) DateTime {
	return DateTime{d.Time.Add(time.Duration(m) * time.Minute)}
}

// LogicalDay returns the day d belongs to; times before the boundary
// (minutes since midnight) belong to the previous day.
func (d DateTime) LogicalDay(boundaryMinutes int) Date {
	day := NewDate(d.Time)
	if d.Hour()*60+d.Minute() < boundaryMinutes {
		return day.AddDays(-1)
	}
	return day
}

// String returns the DateLayout representation.
func (d Date) String() string { return d.Time.Format(DateLayout) }

// AddDays returns d shifted by n calendar days.
func (d Date) AddDays(n int) Date { return Date{d.Time.AddDate(0, 0, n)} }

// At returns the wall-clock time hour:minute on day d.
func (d Date) At(hour, minute int) DateTime {
	return DateTime{time.Date(d.Year(), d.Month(), d.Day(), hour, minute, 0, 0, time.UTC)}
}

// MarshalJSON writes the layout string; the zero value becomes null.
func (d DateTime) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.String())
}

// UnmarshalJSON parses with ParseDateTime; null or "" yields the zero value.
func (d *DateTime) UnmarshalJSON(b []byte) error {
	s, err := jsonString(b)
	if err != nil {
		return err
	}
	if s == "" {
		*d = DateTime{}
		return nil
	}
	v, err := ParseDateTime(s)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// MarshalJSON writes the layout string; the zero value becomes null.
func (d Date) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.String())
}

// UnmarshalJSON parses with ParseDate; null or "" yields the zero value.
func (d *Date) UnmarshalJSON(b []byte) error {
	s, err := jsonString(b)
	if err != nil {
		return err
	}
	if s == "" {
		*d = Date{}
		return nil
	}
	v, err := ParseDate(s)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

func jsonString(b []byte) (string, error) {
	if strings.TrimSpace(string(b)) == "null" {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return "", fmt.Errorf("expected a string: %w", err)
	}
	return s, nil
}

// Value implements driver.Valuer.
func (d DateTime) Value() (driver.Value, error) { return d.String(), nil }

// Value implements driver.Valuer.
func (d Date) Value() (driver.Value, error) { return d.String(), nil }

// Scan implements sql.Scanner. It accepts string, []byte and time.Time and
// returns an error for anything else (including NULL).
func (d *DateTime) Scan(src any) error {
	switch v := src.(type) {
	case string:
		p, err := ParseDateTime(v)
		if err != nil {
			return err
		}
		*d = p
	case []byte:
		p, err := ParseDateTime(string(v))
		if err != nil {
			return err
		}
		*d = p
	case time.Time:
		*d = NewDateTime(v)
	default:
		return fmt.Errorf("cannot scan %T into DateTime", src)
	}
	return nil
}

// Scan implements sql.Scanner. It accepts string, []byte and time.Time and
// returns an error for anything else (including NULL).
func (d *Date) Scan(src any) error {
	switch v := src.(type) {
	case string:
		p, err := ParseDate(v)
		if err != nil {
			return err
		}
		*d = p
	case []byte:
		p, err := ParseDate(string(v))
		if err != nil {
			return err
		}
		*d = p
	case time.Time:
		*d = NewDate(v)
	default:
		return fmt.Errorf("cannot scan %T into Date", src)
	}
	return nil
}
