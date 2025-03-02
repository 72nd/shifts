package web

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/72nd/schipla/internal/db"
	"github.com/gorilla/mux"
)

// middlewareManager handles all middleware functionality
type middlewareManager struct {
	logger       *slog.Logger
	db           *db.DB
	hashPassword func(string) string
}

// newMiddlewareManager creates a new middleware manager
func newMiddlewareManager(logger *slog.Logger, db *db.DB, hashPassword func(string) string) *middlewareManager {
	return &middlewareManager{
		logger:       logger,
		db:           db,
		hashPassword: hashPassword,
	}
}

// loggingMiddleware logs all requests with their path, method and duration
func (m *middlewareManager) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Erstelle einen ResponseWriter, der den Status-Code erfasst
		rw := newStatusRecorder(w)

		// Anfrage protokollieren
		requestID := generateRequestID()
		m.logger.Debug("request started",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)

		// Anfrage verarbeiten
		next.ServeHTTP(rw, r)

		// Dauer berechnen
		duration := time.Since(start)

		// Log-Level basierend auf Status-Code, Pfad und Dauer bestimmen
		level := slog.LevelDebug // Standard für erfolgreiche Anfragen

		// Fehler und Warnungen immer entsprechend loggen
		if rw.status >= 500 {
			level = slog.LevelError
		} else if rw.status >= 400 {
			level = slog.LevelWarn
		} else if duration > 500*time.Millisecond {
			// Langsame Anfragen als Warnung protokollieren
			level = slog.LevelWarn
		} else {
			// Erfolgreiche Anfragen: API-Anfragen auf INFO, statische Ressourcen auf DEBUG
			if len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
				level = slog.LevelInfo // API-Anfragen sind wichtiger
			} else if len(r.URL.Path) >= 8 && r.URL.Path[:8] == "/static/" {
				level = slog.LevelDebug // Statische Ressourcen auf DEBUG
			} else if rw.status == http.StatusOK {
				// Erfolgreiche HTML-Seiten auf INFO
				level = slog.LevelInfo
			}
		}

		// Anfrage abgeschlossen protokollieren
		m.logger.Log(context.Background(), level, "request completed",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", duration.Milliseconds(),
			"size", rw.size,
			"timestamp", time.Now().Format(time.RFC3339),
		)
	})
}

// statusRecorder ist ein ResponseWriter, der den Status-Code und die Größe der Antwort erfasst
type statusRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

// newStatusRecorder erstellt einen neuen statusRecorder
func newStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{
		ResponseWriter: w,
		status:         http.StatusOK, // Standard ist 200 OK
	}
}

// WriteHeader überschreibt die WriteHeader-Methode, um den Status-Code zu erfassen
func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Write überschreibt die Write-Methode, um die Größe der Antwort zu erfassen
func (r *statusRecorder) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.size += size
	return size, err
}

// generateRequestID generiert eine eindeutige ID für jede Anfrage
func generateRequestID() string {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}

// adminMiddleware verifies that the request has a valid admin token
func (m *middlewareManager) adminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		token := vars["token"]
		adminToken := r.Header.Get("X-Admin-Token")

		// Get plan to verify admin password
		plan, err := m.db.GetShiftPlanByToken(token)
		if err != nil {
			m.logger.Error("failed to get shift plan", "error", err)
			http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
			return
		}

		if plan == nil {
			http.NotFound(w, r)
			return
		}

		// Hash the admin token before comparing
		hashedToken := m.hashPassword(adminToken)

		// Verify admin token
		if hashedToken != plan.AdminPassword {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
