package web

import (
	"fmt"
	"net/http"
	"strings"
)

// notModified sets Cache-Control and the plan ETag on a GET response and returns true
// (after writing 304) if the client's If-None-Match matches. Call it right after the
// plan lookup, before any further DB query. The ETag is "<instance>-<rev>-<admin?1:0>";
// the admin flag only reflects whether an X-Admin-Token header was sent (not validated,
// no DB query), which is enough to keep differently-privileged responses apart.
// Without realtime the revision is never bumped, so no ETag is used then.
// Only use it for responses that depend solely on plan data changed via notifyPlanChanged
// (no time-dependent fields).
func (s *Server) notModified(w http.ResponseWriter, r *http.Request, planID int64) bool {
	if !s.realtimeEnabled {
		return false
	}
	admin := 0
	if r.Header.Get("X-Admin-Token") != "" {
		admin = 1
	}
	etag := fmt.Sprintf(`"%s-%d-%d"`, s.hub.instance, s.hub.Revision(planID), admin)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Add("Vary", "X-Admin-Token")
	for _, candidate := range strings.Split(r.Header.Get("If-None-Match"), ",") {
		c := strings.TrimPrefix(strings.TrimSpace(candidate), "W/")
		if c == etag {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}
	return false
}
