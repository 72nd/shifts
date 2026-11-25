package util

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// PersonKey groups participations by person: the normalised name (Unicode NFC,
// trimmed, runs of whitespace collapsed to one space, lower case).
//
// The JavaScript counterpart is personKey in static/js/stats/compute.js
// (toLocaleLowerCase). strings.ToLower and toLocaleLowerCase agree for all
// practically relevant names; they differ for special cases such as the
// Turkish dotted capital İ.
func PersonKey(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(norm.NFC.String(name)), " "))
}
