package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"crypto/sha256"
	"encoding/hex"

	"github.com/72nd/schipla/internal/db"
	"github.com/gorilla/mux"
)

//go:embed templates/* static/*
var content embed.FS

// Server represents the HTTP server
type Server struct {
	db        *db.DB
	logger    *slog.Logger
	templates map[string]*template.Template
	router    *mux.Router
}

// New creates a new HTTP server
func New(db *db.DB, logger *slog.Logger) (*Server, error) {
	s := &Server{
		db:        db,
		logger:    logger,
		templates: make(map[string]*template.Template),
		router:    mux.NewRouter(),
	}

	// Parse templates
	if err := s.parseTemplates(); err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	// Setup routes
	s.setupRoutes()

	return s, nil
}

// parseTemplates parses all templates and stores them in the templates map
func (s *Server) parseTemplates() error {
	// Get all template files
	pages, err := fs.Glob(content, "templates/*.html")
	if err != nil {
		return fmt.Errorf("failed to get template files: %w", err)
	}

	// Get all partials
	partials, err := fs.Glob(content, "templates/partials/*.html")
	if err != nil {
		return fmt.Errorf("failed to get partial files: %w", err)
	}

	// Parse each template
	for _, page := range pages {
		name := filepath.Base(page)

		// Create a new template with functions
		tmpl := template.New(name)

		// Parse base layout and all partials first
		patterns := append([]string{"templates/layouts/base.html"}, partials...)
		if _, err := tmpl.ParseFS(content, patterns...); err != nil {
			return fmt.Errorf("failed to parse base and partials for %s: %w", name, err)
		}

		// Then parse the page template
		if _, err := tmpl.ParseFS(content, page); err != nil {
			return fmt.Errorf("failed to parse template %s: %w", name, err)
		}

		s.templates[name] = tmpl
	}

	return nil
}

// render renders a template with the given name and data
func (s *Server) render(w http.ResponseWriter, name string, data interface{}) {
	// Get template from cache
	ts, ok := s.templates[name]
	if !ok {
		s.logger.Error("template not found", "name", name)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Execute template
	if err := ts.Execute(w, data); err != nil {
		s.logger.Error("failed to execute template", "error", err, "name", name)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// setupRoutes configures all routes for the application
func (s *Server) setupRoutes() {
	// Static file server
	staticFS, err := fs.Sub(content, "static")
	if err != nil {
		s.logger.Error("failed to setup static file server", "error", err)
		return
	}
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Web routes
	s.router.HandleFunc("/", s.handleIndex).Methods("GET")
	s.router.HandleFunc("/shift-plans/new", s.handleNewShiftPlan).Methods("GET")
	s.router.HandleFunc("/shift-plans/success", s.handleShiftPlanSuccess).Methods("GET")

	// API routes
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/shift-plans", s.handleCreateShiftPlan).Methods("POST")
	api.HandleFunc("/shift-plans/{token}", s.handleGetShiftPlan).Methods("GET")
	api.HandleFunc("/shift-plans/{token}/types", s.handleGetShiftTypes).Methods("GET")
	api.HandleFunc("/shift-plans/{token}/types", s.adminMiddleware(s.handleCreateShiftType)).Methods("POST")
	api.HandleFunc("/shift-plans/{token}/types/{id}", s.adminMiddleware(s.handleUpdateShiftType)).Methods("PUT")
	api.HandleFunc("/shift-plans/{token}/types/{id}", s.adminMiddleware(s.handleDeleteShiftType)).Methods("DELETE")
	api.HandleFunc("/shift-plans/{token}/verify-admin", s.handleVerifyAdmin).Methods("POST")
	api.HandleFunc("/shift-plans/{token}/shifts/{id}/participants", s.handleAddParticipant).Methods("POST")
	api.HandleFunc("/shift-plans/{token}/shifts/{id}/participants/{participantId}", s.handleRemoveParticipant).Methods("DELETE")

	// Plan routes
	plans := s.router.PathPrefix("/shift-plans/{token}").Subrouter()
	plans.HandleFunc("", s.handleViewShiftPlan).Methods("GET")
	plans.HandleFunc("/admin", s.handleAdminShiftPlan).Methods("GET")

	// Add middleware
	s.router.Use(s.loggingMiddleware)
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

// Middleware
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
		)
	})
}

// Route handlers
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.render(w, "index.html", nil)
}

func (s *Server) handleNewShiftPlan(w http.ResponseWriter, r *http.Request) {
	s.render(w, "new.html", nil)
}

func (s *Server) handleShiftPlanSuccess(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	password := r.URL.Query().Get("password")

	if url == "" || password == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := map[string]string{
		"url":      url,
		"password": password,
	}

	s.render(w, "success.html", data)
}

func (s *Server) handleCreateShiftPlan(w http.ResponseWriter, r *http.Request) {
	var plan db.NewShiftPlan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		s.logger.Error("failed to decode request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create plan
	createdPlan, err := s.db.CreateShiftPlan(plan)
	if err != nil {
		s.logger.Error("failed to create plan", "error", err)
		http.Error(w, "Failed to create plan", http.StatusInternalServerError)
		return
	}

	// Return plan data
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":    createdPlan.Token,
		"url":      fmt.Sprintf("/shift-plans/%s", createdPlan.Token),
		"password": createdPlan.AdminPassword,
	})
}

func (s *Server) handleGetShiftPlan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plan)
}

func (s *Server) handleViewShiftPlan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

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

	s.render(w, "view.html", plan)
}

func (s *Server) handleAdminShiftPlan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

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

	if err := s.templates["admin.html"].Execute(w, plan); err != nil {
		s.logger.Error("failed to render template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (s *Server) handleGetShiftTypes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	// Get plan first to verify it exists and get its ID
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

	// Get shift types with their shifts
	types, err := s.db.GetShiftTypesByPlan(plan.ID)
	if err != nil {
		s.logger.Error("failed to get shift types", "error", err)
		http.Error(w, "Failed to get shift types", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types)
}

func (s *Server) handleAddParticipant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	shiftID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid shift ID", http.StatusBadRequest)
		return
	}

	// Verify the shift belongs to the plan
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to verify shift plan", http.StatusInternalServerError)
		return
	}
	if plan == nil {
		http.NotFound(w, r)
		return
	}

	// Verify the shift belongs to this plan
	belongs, err := s.db.VerifyShiftBelongsToPlan(shiftID, plan.ID)
	if err != nil {
		s.logger.Error("failed to verify shift ownership", "error", err)
		http.Error(w, "Failed to verify shift ownership", http.StatusInternalServerError)
		return
	}
	if !belongs {
		http.Error(w, "Shift not found", http.StatusNotFound)
		return
	}

	var data struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	participant, err := s.db.AddParticipant(shiftID, data.Name)
	if err != nil {
		s.logger.Error("failed to add participant", "error", err)
		http.Error(w, "Failed to add participant", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(participant)
}

func (s *Server) handleRemoveParticipant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	shiftID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid shift ID", http.StatusBadRequest)
		return
	}

	// Verify the shift belongs to the plan
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to verify shift plan", http.StatusInternalServerError)
		return
	}
	if plan == nil {
		http.NotFound(w, r)
		return
	}

	// Verify the shift belongs to this plan
	belongs, err := s.db.VerifyShiftBelongsToPlan(shiftID, plan.ID)
	if err != nil {
		s.logger.Error("failed to verify shift ownership", "error", err)
		http.Error(w, "Failed to verify shift ownership", http.StatusInternalServerError)
		return
	}
	if !belongs {
		http.Error(w, "Shift not found", http.StatusNotFound)
		return
	}

	// Get the participant from the database first to get their name
	participants, err := s.db.GetParticipantsByShift(shiftID)
	if err != nil {
		s.logger.Error("failed to get participants", "error", err)
		http.Error(w, "Failed to get participants", http.StatusInternalServerError)
		return
	}

	participantID, err := strconv.ParseInt(vars["participantId"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid participant ID", http.StatusBadRequest)
		return
	}

	// Find the participant with the matching ID
	var participantName string
	for _, p := range participants {
		if p.ID == participantID {
			participantName = p.Name
			break
		}
	}

	if participantName == "" {
		http.Error(w, "Participant not found", http.StatusNotFound)
		return
	}

	// Remove the participant using their name
	err = s.db.RemoveParticipant(shiftID, participantName)
	if err != nil {
		s.logger.Error("failed to remove participant", "error", err)
		http.Error(w, "Failed to remove participant", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// adminMiddleware verifies the admin token for protected routes
func (s *Server) adminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		token := vars["token"]
		adminToken := r.Header.Get("X-Admin-Token")

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

		// Hash the admin token before comparing
		hashedToken := hashPassword(adminToken)

		// Verify admin token
		if hashedToken != plan.AdminPassword {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
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

// handleCreateShiftType handles the creation of a new shift type
func (s *Server) handleCreateShiftType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	// Get plan first to verify it exists and get its ID
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

	// Parse request body
	var shiftType db.NewShiftType
	if err := json.NewDecoder(r.Body).Decode(&shiftType); err != nil {
		s.logger.Error("failed to decode request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set plan ID
	shiftType.PlanID = plan.ID

	// Create shift type
	createdType, err := s.db.CreateShiftType(shiftType)
	if err != nil {
		s.logger.Error("failed to create shift type", "error", err)
		http.Error(w, "Failed to create shift type", http.StatusInternalServerError)
		return
	}

	// Return created shift type
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdType)
}

// handleUpdateShiftType handles updating an existing shift type
func (s *Server) handleUpdateShiftType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	typeID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid shift type ID", http.StatusBadRequest)
		return
	}

	// Get plan first to verify it exists
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

	// Parse request body
	var shiftType db.ShiftType
	if err := json.NewDecoder(r.Body).Decode(&shiftType); err != nil {
		s.logger.Error("failed to decode request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Ensure the ID matches
	shiftType.ID = typeID
	shiftType.PlanID = plan.ID

	// Update shift type
	if err := s.db.UpdateShiftType(typeID, shiftType); err != nil {
		s.logger.Error("failed to update shift type", "error", err)
		http.Error(w, "Failed to update shift type", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleDeleteShiftType handles deleting an existing shift type
func (s *Server) handleDeleteShiftType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	typeID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid shift type ID", http.StatusBadRequest)
		return
	}

	// Get plan first to verify it exists
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

	// Delete the shift type
	if err := s.db.DeleteShiftType(plan.ID, typeID); err != nil {
		s.logger.Error("failed to delete shift type", "error", err)
		http.Error(w, "failed to delete shift type", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
