package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Realtime scopes, ordered by rank: when events are coalesced the highest rank wins.
const (
	scopeParticipants = "participants"
	scopeShifts       = "shifts"
	scopeTypes        = "types"
	scopePlan         = "plan"
	scopeDeleted      = "deleted"
)

var scopeRank = map[string]int{
	scopeParticipants: 1,
	scopeShifts:       2,
	scopeTypes:        3,
	scopePlan:         4,
	scopeDeleted:      5,
}

const (
	realtimeCoalesce      = 300 * time.Millisecond
	realtimePingInterval  = 25 * time.Second
	realtimeMaxPerPlan    = 500
	realtimeMaxGlobal     = 2000
	realtimeEventPresence = "presence"
	realtimeEventChange   = "change"
)

var errRealtimeFull = errors.New("too many realtime connections")

// realtimeEvent is what a subscriber receives. Kind is "change" or "presence".
type realtimeEvent struct {
	Kind    string
	Rev     uint64
	Scope   string
	Viewers int
}

// realtimeSub is one subscription to a plan. Read events from C, call Close when done.
type realtimeSub struct {
	C       chan realtimeEvent // buffer 1, newest event replaces an unread one
	Done    chan struct{}      // closed when the hub shuts down
	Rev     uint64             // revision at subscribe time
	Viewers int                // subscriber count at subscribe time
	hub     *realtimeHub
	planID  int64
	once    sync.Once
}

// Close removes the subscription from the hub. Safe to call more than once.
func (sub *realtimeSub) Close() {
	sub.once.Do(func() { sub.hub.unsubscribe(sub) })
}

type realtimePlan struct {
	rev       uint64
	subs      map[*realtimeSub]struct{}
	scope     string // pending scope, "" if none
	presence  bool   // viewer count changed since last event
	timerOpen bool
}

// realtimeHub keeps an in-memory revision per plan and fans change events out to subscribers.
// It lives in the process (single server instance assumed).
type realtimeHub struct {
	mu       sync.Mutex
	instance string
	plans    map[int64]*realtimePlan
	total    int
	closed   bool
}

func newRealtimeHub() *realtimeHub {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return &realtimeHub{instance: hex.EncodeToString(b), plans: map[int64]*realtimePlan{}}
}

// plan returns the state of a plan, creating it if needed. Caller holds h.mu.
func (h *realtimeHub) plan(planID int64) *realtimePlan {
	p := h.plans[planID]
	if p == nil {
		p = &realtimePlan{subs: map[*realtimeSub]struct{}{}}
		h.plans[planID] = p
	}
	return p
}

// arm starts the coalescing timer of a plan if not running. Caller holds h.mu.
func (h *realtimeHub) arm(planID int64, p *realtimePlan) {
	if p.timerOpen {
		return
	}
	p.timerOpen = true
	time.AfterFunc(realtimeCoalesce, func() { h.flush(planID) })
}

// Revision returns the current revision of a plan (0 after server start).
func (h *realtimeHub) Revision(planID int64) uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.plan(planID).rev
}

// Publish bumps the revision of a plan and schedules a coalesced change event.
func (h *realtimeHub) Publish(planID int64, scope string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	p := h.plan(planID)
	p.rev++
	if scopeRank[scope] >= scopeRank[p.scope] {
		p.scope = scope
	}
	if len(p.subs) > 0 {
		h.arm(planID, p)
	}
}

// Subscribe registers a subscriber, or returns errRealtimeFull when a limit is hit.
func (h *realtimeHub) Subscribe(planID int64) (*realtimeSub, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	p := h.plan(planID)
	if h.closed || h.total >= realtimeMaxGlobal || len(p.subs) >= realtimeMaxPerPlan {
		return nil, errRealtimeFull
	}
	sub := &realtimeSub{
		C: make(chan realtimeEvent, 1), Done: make(chan struct{}),
		hub: h, planID: planID,
	}
	p.subs[sub] = struct{}{}
	h.total++
	sub.Rev, sub.Viewers = p.rev, len(p.subs)
	p.presence = true
	h.arm(planID, p)
	return sub, nil
}

func (h *realtimeHub) unsubscribe(sub *realtimeSub) {
	h.mu.Lock()
	defer h.mu.Unlock()
	p := h.plans[sub.planID]
	if p == nil {
		return
	}
	if _, ok := p.subs[sub]; !ok {
		return
	}
	delete(p.subs, sub)
	h.total--
	if len(p.subs) > 0 {
		p.presence = true
		h.arm(sub.planID, p)
	} else {
		p.presence = false
	}
}

// flush sends the pending change (or presence) event to all subscribers of a plan.
func (h *realtimeHub) flush(planID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	p := h.plans[planID]
	if p == nil {
		return
	}
	p.timerOpen = false
	ev := realtimeEvent{Kind: realtimeEventPresence, Rev: p.rev, Viewers: len(p.subs)}
	if p.scope != "" {
		ev.Kind, ev.Scope = realtimeEventChange, p.scope
	} else if !p.presence {
		return
	}
	p.scope, p.presence = "", false
	for sub := range p.subs {
		if ev.Kind == realtimeEventChange {
			select { // replace an unread event
			case <-sub.C:
			default:
			}
		}
		select {
		case sub.C <- ev:
		default: // presence never replaces an unread event
		}
	}
}

// Close ends all subscriptions (used on shutdown).
func (h *realtimeHub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	for _, p := range h.plans {
		for sub := range p.subs {
			close(sub.Done)
		}
	}
}

// notifyPlanChanged tells realtime subscribers that a plan changed. Call after a successful write.
func (s *Server) notifyPlanChanged(planID int64, scope string) {
	s.hub.Publish(planID, scope)
}

// handlePlanEvents streams plan change events as Server-Sent Events.
// @Summary Stream plan changes
// @Description Server-Sent Events stream. Events: `hello` ({instance, rev, viewers}), `change` ({rev, scope, viewers}, scope is participants, shifts, types, plan or deleted) and `presence` ({viewers}). The client reloads the affected data on `change`. Returns 404 if realtime is disabled.
// @Tags Plans
// @Produce text/event-stream
// @Param token path string true "Plan token"
// @Success 200 {string} string "Event stream"
// @Failure 404 {string} string "Plan not found or realtime disabled"
// @Failure 503 {string} string "Too many connections"
// @Router /plans/{token}/events [get]
func (s *Server) handlePlanEvents(w http.ResponseWriter, r *http.Request) {
	if !s.realtimeEnabled {
		s.RespondWithError(w, NewAppError(ErrResourceNotFound, "Resource not found", http.StatusNotFound, nil))
		return
	}
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	sub, err := s.hub.Subscribe(plan.ID)
	if err != nil {
		w.Header().Set("Retry-After", "10")
		http.Error(w, "Too many realtime connections", http.StatusServiceUnavailable)
		return
	}
	defer sub.Close()
	realtimeConnections.Inc()
	defer realtimeConnections.Dec()

	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{}) // the global WriteTimeout must not end the stream
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	send := func(format string, args ...any) bool {
		if _, err := fmt.Fprintf(w, format, args...); err != nil {
			return false
		}
		return rc.Flush() == nil
	}
	hello, _ := json.Marshal(map[string]any{"instance": s.hub.instance, "rev": sub.Rev, "viewers": sub.Viewers})
	if !send("event: hello\ndata: %s\n\n", hello) {
		return
	}

	ping := time.NewTicker(realtimePingInterval)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-sub.Done:
			return
		case <-ping.C:
			if !send(": ping\n\n") {
				return
			}
		case ev := <-sub.C:
			var okSend bool
			if ev.Kind == realtimeEventChange {
				data, _ := json.Marshal(map[string]any{"rev": ev.Rev, "scope": ev.Scope, "viewers": ev.Viewers})
				okSend = send("id: %d\nevent: change\ndata: %s\n\n", ev.Rev, data)
			} else {
				okSend = send("event: presence\ndata: {\"viewers\":%d}\n\n", ev.Viewers)
			}
			if !okSend {
				return
			}
			realtimeEventsTotal.Inc()
		}
	}
}
