package web

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/export"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/72nd/shifts/docs" // Import generated swagger docs
)

//go:embed templates/* static/* docs/*.md docs/images/*
var content embed.FS

// Server represents the HTTP server
type Server struct {
	db            *db.DB
	logger        *slog.Logger
	templates     *templateManager
	docs          *documentationManager
	middleware    *middlewareManager
	router        *mux.Router
	exportService *export.Service
	defaultView   string // "list", "chrono" or "timetable" — fallback for shift plan page view mode
	defaultZone   string // IANA name of the default time zone for new plans

	hub             *realtimeHub
	realtimeEnabled bool
}

// NewServer creates a new server instance. defaultView must be "list", "chrono" or "timetable".
func NewServer(db *db.DB, logger *slog.Logger, defaultView, defaultZone string, realtimeEnabled bool) (*Server, error) {
	// Create export service
	exportService := export.NewService(db)

	// Register exporters
	exportService.RegisterExporter(export.NewJSONExporter())
	exportService.RegisterExporter(export.NewVacanciesExporter())
	exportService.RegisterExporter(export.NewTimetableExporter())
	exportService.RegisterExporter(export.NewReportExporter())
	exportService.RegisterExporter(export.NewPosterExporter())
	exportService.RegisterExporter(export.NewCSVExporter())

	s := &Server{
		db:            db,
		logger:        logger,
		templates:     newTemplateManager(logger),
		docs:          newDocumentationManager(logger),
		middleware:    newMiddlewareManager(logger, db, hashPassword),
		router:        mux.NewRouter(),
		exportService: exportService,
		defaultView:   defaultView,
		defaultZone:   defaultZone,

		hub:             newRealtimeHub(),
		realtimeEnabled: realtimeEnabled,
	}

	// Parse templates
	if err := s.templates.parseTemplates(); err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	// Parse documentation
	if err := s.docs.parseDocumentation(); err != nil {
		return nil, fmt.Errorf("failed to parse documentation: %w", err)
	}

	// Setup routes
	s.setupRoutes()

	return s, nil
}

// setupRoutes configures all routes for the application
func (s *Server) setupRoutes() {
	// Static files - use embedded filesystem
	staticFS, err := fs.Sub(content, "static")
	if err != nil {
		s.logger.Error("failed to setup static file server", "error", err)
	} else {
		s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	// HTML routes
	s.router.HandleFunc("/", s.handleIndex).Methods("GET")
	s.router.HandleFunc("/plans/new", s.handleNewShiftPlan).Methods("GET")
	s.router.HandleFunc("/plans/success", s.handleShiftPlanSuccess).Methods("GET")
	s.router.HandleFunc("/plans/{token}", s.handleViewShiftPlan).Methods("GET")
	s.router.HandleFunc("/plans/{token}/stats", s.handleShiftPlanStats).Methods("GET")
	s.router.HandleFunc("/plans/{token}/coordinate", s.handleCoordinationPage).Methods("GET")
	s.router.HandleFunc("/help", s.handleHelp).Methods("GET")
	s.router.HandleFunc("/images/{filename}", s.handleHelpImage).Methods("GET")
	s.router.HandleFunc("/plan-deleted", s.handlePlanDeleted()).Methods("GET")

	// API routes
	api := s.router.PathPrefix("/api").Subrouter()
	// Apply CORS middleware only to API routes (needed for Swagger UI and external API clients)
	api.Use(s.middleware.corsMiddleware)
	// gzip runs inside the logging/metrics middleware (added on the root router); skips /events
	api.Use(gzipMiddleware)
	// Handle OPTIONS preflight requests for all API routes (catch-all)
	api.PathPrefix("/").Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers are already set by the middleware
		w.WriteHeader(http.StatusOK)
	})

	// Plan routes
	api.HandleFunc("/plans", s.handleCreateShiftPlan).Methods("POST")
	api.HandleFunc("/plans/{token}", s.handleGetShiftPlan).Methods("GET")
	api.HandleFunc("/plans/{token}", s.middleware.adminMiddleware(s.handleUpdateShiftPlan)).Methods("PUT")
	api.HandleFunc("/plans/{token}", s.middleware.adminMiddleware(s.handleDeleteShiftPlan)).Methods("DELETE")
	api.HandleFunc("/plans/{token}/verify-admin", s.handleVerifyAdmin).Methods("POST")
	api.HandleFunc("/plans/{token}/stats", s.handleGetShiftPlanStats).Methods("GET")
	api.HandleFunc("/plans/{token}/export", s.middleware.adminMiddleware(s.handleExportPlan)).Methods("GET")

	api.HandleFunc("/plans/{token}/events", s.handlePlanEvents).Methods("GET")

	// Type routes
	api.HandleFunc("/plans/{token}/types", s.handleGetShiftTypes).Methods("GET")
	api.HandleFunc("/plans/{token}/types", s.middleware.adminMiddleware(s.handleCreateShiftType)).Methods("POST")
	api.HandleFunc("/plans/{token}/types/order", s.middleware.adminMiddleware(s.handleUpdateShiftTypeOrder)).Methods("PUT")
	api.HandleFunc("/plans/{token}/types/{typeId}", s.middleware.adminMiddleware(s.handleUpdateShiftType)).Methods("PUT")
	api.HandleFunc("/plans/{token}/types/{typeId}", s.middleware.adminMiddleware(s.handleDeleteShiftType)).Methods("DELETE")

	// Shift routes
	api.HandleFunc("/plans/{token}/shifts", s.middleware.adminMiddleware(s.handleCreateShift)).Methods("POST")

	// Batch shift operations (must be before {shiftId} routes to avoid routing conflicts)
	api.HandleFunc("/plans/{token}/shifts/batch", s.middleware.adminMiddleware(s.handleBatchCreateShifts)).Methods("POST")
	api.HandleFunc("/plans/{token}/shifts/batch", s.middleware.adminMiddleware(s.handleBatchUpdateShifts)).Methods("PUT")
	api.HandleFunc("/plans/{token}/shifts/batch", s.middleware.adminMiddleware(s.handleBatchDeleteShifts)).Methods("DELETE")

	// Single shift operations
	api.HandleFunc("/plans/{token}/shifts/{shiftId}", s.middleware.adminMiddleware(s.handleUpdateShift)).Methods("PUT")
	api.HandleFunc("/plans/{token}/shifts/{shiftId}", s.middleware.adminMiddleware(s.handleDeleteShift)).Methods("DELETE")
	api.HandleFunc("/plans/{token}/shifts/{shiftId}/lock", s.middleware.adminMiddleware(s.handleLockShift)).Methods("POST")
	api.HandleFunc("/plans/{token}/shifts/{shiftId}/unlock", s.middleware.adminMiddleware(s.handleUnlockShift)).Methods("POST")

	// Participant routes
	api.HandleFunc("/plans/{token}/shifts/{shiftId}/participants", s.handleAddParticipant).Methods("POST")
	api.HandleFunc("/plans/{token}/shifts/{shiftId}/participants/{participantId}", s.handleRemoveParticipant).Methods("DELETE")

	// Person field routes ("order" before {fieldId})
	api.HandleFunc("/plans/{token}/fields", s.middleware.adminMiddleware(s.handleCreateField)).Methods("POST")
	api.HandleFunc("/plans/{token}/fields/order", s.middleware.adminMiddleware(s.handleUpdateFieldOrder)).Methods("PUT")
	api.HandleFunc("/plans/{token}/fields/{fieldId}", s.middleware.adminMiddleware(s.handleUpdateField)).Methods("PUT")
	api.HandleFunc("/plans/{token}/fields/{fieldId}", s.middleware.adminMiddleware(s.handleDeleteField)).Methods("DELETE")

	// People routes (fixed paths before {key})
	api.HandleFunc("/plans/{token}/people", s.middleware.adminMiddleware(s.handleListPeople)).Methods("GET")
	api.HandleFunc("/plans/{token}/people", s.middleware.adminMiddleware(s.handleUpsertPerson)).Methods("PUT")
	api.HandleFunc("/plans/{token}/people/import", s.middleware.adminMiddleware(s.handleImportPeople)).Methods("POST")
	api.HandleFunc("/plans/{token}/people/rename", s.middleware.adminMiddleware(s.handleRenamePerson)).Methods("POST")
	api.HandleFunc("/plans/{token}/people/lookup", s.handleLookupPerson).Methods("GET")
	api.HandleFunc("/plans/{token}/people/{key:.+}", s.middleware.adminMiddleware(s.handleDeletePerson)).Methods("DELETE")

	// Coordination
	api.HandleFunc("/plans/{token}/coordination/apply", s.middleware.adminMiddleware(s.handleApplyCoordination)).Methods("POST")
	api.HandleFunc("/plans/{token}/coordination/drafts", s.middleware.adminMiddleware(s.handleListDrafts)).Methods("GET")
	api.HandleFunc("/plans/{token}/coordination/drafts", s.middleware.adminMiddleware(s.handleCreateDraft)).Methods("POST")
	api.HandleFunc("/plans/{token}/coordination/drafts/{id}", s.middleware.adminMiddleware(s.handleGetDraft)).Methods("GET")
	api.HandleFunc("/plans/{token}/coordination/drafts/{id}", s.middleware.adminMiddleware(s.handleUpdateDraft)).Methods("PUT")
	api.HandleFunc("/plans/{token}/coordination/drafts/{id}", s.middleware.adminMiddleware(s.handleDeleteDraft)).Methods("DELETE")

	// Swagger documentation - also needs CORS for cross-origin requests
	swaggerRouter := s.router.PathPrefix("/swagger/").Subrouter()
	swaggerRouter.Use(s.middleware.corsMiddleware)
	swaggerRouter.PathPrefix("/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Add logging middleware to all routes
	s.router.Use(s.middleware.loggingMiddleware)

	// Add metrics middleware to all routes
	s.router.Use(s.middleware.metricsMiddleware)

	// 404 Handler for HTML routes (must be added last)
	s.router.NotFoundHandler = s.handle404()
}

// Start starts the HTTP server
func (s *Server) Start(host string, port int) error {
	// Create server with timeouts
	// WriteTimeout increased to 120s to accommodate PDF generation with Typst
	srv := &http.Server{
		Handler:      s.router,
		Addr:         fmt.Sprintf("%s:%d", host, port),
		WriteTimeout: 120 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	s.logger.Info("starting server", "addr", srv.Addr)
	return srv.ListenAndServe()
}

// StartMetricsServer starts a separate HTTP server for Prometheus metrics
// This server runs on a different port and only exposes /metrics endpoint
func (s *Server) StartMetricsServer(port int) error {
	metricsRouter := http.NewServeMux()
	metricsRouter.Handle("/metrics", promhttp.Handler())

	metricsServer := &http.Server{
		Addr:         fmt.Sprintf("localhost:%d", port),
		Handler:      metricsRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	s.logger.Info("starting metrics server", "addr", metricsServer.Addr)
	return metricsServer.ListenAndServe()
}

// handleVerifyAdmin verifies the admin password for a shift plan
// @Summary Verify admin password
// @Description Verifies the admin password for a shift plan and returns the token if valid
// @Tags Plans
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param data body VerifyAdminRequest true "Password to verify"
// @Success 200 {object} VerifyAdminResponse "Token if password is valid"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Invalid password"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Router /plans/{token}/verify-admin [post]
func (s *Server) handleVerifyAdmin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	var data struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid request body",
			http.StatusBadRequest,
			err,
		))
		return
	}

	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to get shift plan",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	if plan == nil {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Shift plan not found",
			http.StatusNotFound,
			nil,
		))
		return
	}

	// Hash the password before comparing
	hashedPassword := hashPassword(data.Password)

	// Verify password
	if hashedPassword != plan.AdminPassword {
		s.RespondWithError(w, NewAppError(
			ErrInvalidPassword,
			"Invalid password",
			http.StatusUnauthorized,
			nil,
		))
		return
	}

	// Return the password as token
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": data.Password,
	})
}

// hashPassword creates a simple hash of the password
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// RespondWithError sendet eine Fehlerantwort an den Client und loggt den Fehler
func (s *Server) RespondWithError(w http.ResponseWriter, err *AppError) {
	RespondWithError(w, s.logger, err)
}

func (s *Server) handleHelp(w http.ResponseWriter, r *http.Request) {
	// Get language from query parameter or default to English
	lang := r.URL.Query().Get("lang")
	if lang != "de" && lang != "en" {
		lang = "en"
	}

	// Get documentation for the specified language
	doc, ok := s.docs.getDocumentation(lang)
	if !ok {
		s.logger.Error("documentation not found", "lang", lang)
		http.Error(w, "Documentation not found", http.StatusInternalServerError)
		return
	}

	// Prepare data for template
	data := map[string]interface{}{
		"Lang":    lang,
		"Content": doc.HTML,
		"TOC":     doc.TOC,
	}

	s.templates.render(w, "help.html", data)
}

func (s *Server) handleHelpImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	filename := vars["filename"]

	// Construct the path to the image in the embedded filesystem
	imagePath := fmt.Sprintf("docs/images/%s", filename)

	// Read the image from the embedded filesystem
	data, err := content.ReadFile(imagePath)
	if err != nil {
		s.logger.Error("failed to read help image", "error", err, "path", imagePath)
		http.NotFound(w, r)
		return
	}

	// Determine content type based on file extension
	contentType := "application/octet-stream"
	if strings.HasSuffix(filename, ".png") {
		contentType = "image/png"
	} else if strings.HasSuffix(filename, ".jpg") || strings.HasSuffix(filename, ".jpeg") {
		contentType = "image/jpeg"
	} else if strings.HasSuffix(filename, ".gif") {
		contentType = "image/gif"
	} else if strings.HasSuffix(filename, ".svg") {
		contentType = "image/svg+xml"
	} else if strings.HasSuffix(filename, ".webp") {
		contentType = "image/webp"
	}

	// Set content type and cache headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year

	// Write the image data
	w.Write(data)
}

func (s *Server) handlePlanDeleted() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.templates.render(w, "plan-deleted.html", nil)
	}
}

func (s *Server) handle404() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only handle non-API routes
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			data := map[string]interface{}{
				"error": fmt.Sprintf("Die URL '%s' wurde nicht gefunden.", r.URL.Path),
			}
			s.templates.render(w, "404.html", data)
			return
		}

		// For API routes, return JSON error
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Resource not found",
			http.StatusNotFound,
			nil,
		))
	})
}
