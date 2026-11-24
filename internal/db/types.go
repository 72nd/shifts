package db

import (
	"fmt"
	"time"
)

// parseTimestamp parses a stored RFC3339 instant (created_at) and names the
// row in the error.
func parseTimestamp(table string, id int64, column, value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s id %d: invalid %s %q: %w", table, id, column, value, err)
	}
	return t, nil
}

// Language represents supported languages
type Language string

const (
	DE Language = "de"
	EN Language = "en"
)
