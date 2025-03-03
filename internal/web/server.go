package web

import (
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
	s.router.HandleFunc("/plans/{token}/stats", s.handleShiftPlanStats).Methods("GET")

	// API routes
	api := s.router.PathPrefix("/api").Subrouter()

	// Plan routes
	api.HandleFunc("/plans", s.handleCreateShiftPlan).Methods("POST")
	api.HandleFunc("/plans/{token}", s.handleGetShiftPlan).Methods("GET")
	api.HandleFunc("/plans/{token}", s.middleware.adminMiddleware(s.handleUpdateShiftPlan)).Methods("PUT")
	api.HandleFunc("/plans/{token}/verify-admin", s.handleVerifyAdmin).Methods("POST")
	api.HandleFunc("/plans/{token}/stats", s.handleGetShiftPlanStats).Methods("GET")

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
