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

	// Parse each template
	for _, page := range pages {
		name := filepath.Base(page)

		// Create a new template with functions
		ts, err := template.New(name).ParseFS(content, "templates/layouts/base.html", page)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", name, err)
		}

		s.templates[name] = ts
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
	api.HandleFunc("/shifts/{id}/participants", s.handleAddParticipant).Methods("POST")

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
	shiftID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid shift ID", http.StatusBadRequest)
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
