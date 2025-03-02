package web

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/72nd/schipla/internal/db"
	"github.com/gorilla/mux"
)

//go:embed templates/* static/*
var content embed.FS

// Server represents the HTTP server
type Server struct {
	db         *db.DB
	logger     *slog.Logger
	templates  *templateManager
	middleware *middlewareManager
	router     *mux.Router
}

// New creates a new HTTP server
func New(db *db.DB, logger *slog.Logger) (*Server, error) {
	s := &Server{
		db:         db,
		logger:     logger,
		templates:  newTemplateManager(logger),
		middleware: newMiddlewareManager(logger, db, hashPassword),
		router:     mux.NewRouter(),
	}

	// Parse templates
	if err := s.templates.parseTemplates(); err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	// Setup routes
	s.setupRoutes()

	return s, nil
}

// setupRoutes configures all routes for the application
func (s *Server) setupRoutes() {
	// Static files
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("internal/web/static"))))

	// HTML routes
	s.router.HandleFunc("/", s.handleIndex).Methods("GET")
	s.router.HandleFunc("/plans/new", s.handleNewShiftPlan).Methods("GET")
	s.router.HandleFunc("/plans/success", s.handleShiftPlanSuccess).Methods("GET")
	s.router.HandleFunc("/plans/{token}", s.handleViewShiftPlan).Methods("GET")

	// API routes
	api := s.router.PathPrefix("/api").Subrouter()

	// Plan routes
	api.HandleFunc("/plans", s.handleCreateShiftPlan).Methods("POST")
	api.HandleFunc("/plans/{token}", s.handleGetShiftPlan).Methods("GET")
	api.HandleFunc("/plans/{token}", s.middleware.adminMiddleware(s.handleUpdateShiftPlan)).Methods("PUT")
	api.HandleFunc("/plans/{token}/verify-admin", s.handleVerifyAdmin).Methods("POST")

	// Type routes
	api.HandleFunc("/plans/{token}/types", s.handleGetShiftTypes).Methods("GET")
	api.HandleFunc("/plans/{token}/types", s.middleware.adminMiddleware(s.handleCreateShiftType)).Methods("POST")
	api.HandleFunc("/plans/{token}/types/order", s.middleware.adminMiddleware(s.handleUpdateShiftTypeOrder)).Methods("PUT")
	api.HandleFunc("/plans/{token}/types/{typeId}", s.middleware.adminMiddleware(s.handleUpdateShiftType)).Methods("PUT")
	api.HandleFunc("/plans/{token}/types/{typeId}", s.middleware.adminMiddleware(s.handleDeleteShiftType)).Methods("DELETE")

	// Shift routes
	api.HandleFunc("/plans/{token}/shifts", s.middleware.adminMiddleware(s.handleCreateShift)).Methods("POST")
	api.HandleFunc("/plans/{token}/shifts/{shiftId}", s.middleware.adminMiddleware(s.handleUpdateShift)).Methods("PUT")
	api.HandleFunc("/plans/{token}/shifts/{shiftId}", s.middleware.adminMiddleware(s.handleDeleteShift)).Methods("DELETE")

	// Participant routes
	api.HandleFunc("/plans/{token}/shifts/{shiftId}/participants", s.handleAddParticipant).Methods("POST")
	api.HandleFunc("/plans/{token}/shifts/{shiftId}/participants/{participantId}", s.handleRemoveParticipant).Methods("DELETE")

	// Add middleware
	s.router.Use(s.middleware.loggingMiddleware)
}

// Start starts the HTTP server
func (s *Server) Start(host string, port int) error {
	// Create server with timeouts
	srv := &http.Server{
		Handler:      s.router,
		Addr:         fmt.Sprintf("%s:%d", host, port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	s.logger.Info("starting server", "addr", srv.Addr)
	return srv.ListenAndServe()
}

// handleVerifyAdmin verifies the admin password for a shift plan
func (s *Server) handleVerifyAdmin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	var data struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get plan to verify admin password
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}

	if plan == nil {
		http.NotFound(w, r)
		return
	}

	// Hash the password before comparing
	hashedPassword := hashPassword(data.Password)

	// Verify password
	if hashedPassword != plan.AdminPassword {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// hashPassword creates a simple hash of the password
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// logError logs errors with context and appropriate log levels
func (s *Server) logError(r *http.Request, operation string, err error, statusCode int, args ...any) {
	// Bestimme den Log-Level basierend auf dem Fehlertyp und Statuscode
	level := slog.LevelError

	// Für bestimmte HTTP-Statuscodes können wir einen anderen Level verwenden
	if statusCode >= 400 && statusCode < 500 {
		// Client-Fehler sind oft erwartbar, daher nur Warn
		level = slog.LevelWarn
	}

	// Sammle Kontext-Informationen aus der Anfrage
	requestInfo := map[string]interface{}{
		"method":      r.Method,
		"path":        r.URL.Path,
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.UserAgent(),
	}

	// Erstelle ein strukturiertes Log mit Kontext
	attrs := []any{
		"operation", operation,
		"error", err.Error(),
		"status_code", statusCode,
		"request", requestInfo,
		"timestamp", time.Now().Format(time.RFC3339),
	}

	// Füge zusätzliche Argumente hinzu, wenn vorhanden
	if len(args) > 0 {
		attrs = append(attrs, "details", args)
	}

	s.logger.Log(context.Background(), level, "web handler error", attrs...)
}

// logSuccess logs successful operations with context
func (s *Server) logSuccess(r *http.Request, operation string, statusCode int, args ...any) {
	// Sammle Kontext-Informationen aus der Anfrage
	requestInfo := map[string]interface{}{
		"method":      r.Method,
		"path":        r.URL.Path,
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.UserAgent(),
	}

	// Bestimme den Log-Level basierend auf dem Pfad und der Operation
	level := slog.LevelDebug // Standard: Debug für erfolgreiche Anfragen

	// Für wichtige Operationen (z.B. Erstellung, Aktualisierung, Löschung) verwenden wir Info
	if operation == "create_shift_plan" ||
		operation == "update_shift_plan" ||
		operation == "delete_shift_plan" ||
		operation == "create_shift_type" ||
		operation == "update_shift_type" ||
		operation == "delete_shift_type" ||
		operation == "create_shift" ||
		operation == "update_shift" ||
		operation == "delete_shift" {
		level = slog.LevelInfo
	}

	// Statische Ressourcen immer auf Debug-Level
	if len(r.URL.Path) >= 8 && r.URL.Path[:8] == "/static/" {
		level = slog.LevelDebug
	}

	// Erstelle ein strukturiertes Log mit Kontext
	attrs := []any{
		"operation", operation,
		"status_code", statusCode,
		"request", requestInfo,
		"timestamp", time.Now().Format(time.RFC3339),
	}

	// Füge zusätzliche Argumente hinzu, wenn vorhanden
	if len(args) > 0 {
		attrs = append(attrs, "details", args)
	}

	s.logger.Log(context.Background(), level, "web handler success", attrs...)
}

// logRequest logs incoming requests with their parameters
func (s *Server) logRequest(r *http.Request, operation string, args ...any) {
	// Sammle Kontext-Informationen aus der Anfrage
	requestInfo := map[string]interface{}{
		"method":       r.Method,
		"path":         r.URL.Path,
		"remote_addr":  r.RemoteAddr,
		"user_agent":   r.UserAgent(),
		"query_params": r.URL.Query(),
	}

	// Sammle Header-Informationen (ohne sensible Daten)
	headers := make(map[string]string)
	for key, values := range r.Header {
		// Sensible Header ausschließen oder maskieren
		if key == "Authorization" || key == "X-Admin-Token" {
			headers[key] = "[REDACTED]"
		} else if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	requestInfo["headers"] = headers

	// Erstelle ein strukturiertes Log mit Kontext
	attrs := []any{
		"operation", operation,
		"request", requestInfo,
		"timestamp", time.Now().Format(time.RFC3339),
	}

	// Füge zusätzliche Argumente hinzu, wenn vorhanden
	if len(args) > 0 {
		attrs = append(attrs, "details", args)
	}

	// Verwende Debug-Level für Anfrage-Logs, um die Menge zu reduzieren
	s.logger.Log(context.Background(), slog.LevelDebug, "web handler request", attrs...)
}
