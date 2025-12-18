package web

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/72nd/shifts/internal/db"
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

// corsMiddleware handles CORS headers for cross-origin requests (e.g., from Swagger UI)
func (m *middlewareManager) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-Token, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

		// Handle preflight OPTIONS requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs all requests with their path, method and duration
func (m *middlewareManager) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		args := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start).String(),
		}
		if strings.HasPrefix(r.URL.Path, "/static/") {
			m.logger.Debug("request completed", args...)
		} else {
			m.logger.Info("request completed", args...)
		}
	})
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
