package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/gorilla/mux"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
)

var (
	templates = template.Must(template.New("").Funcs(template.FuncMap{
		"markdown": func(s string) template.HTML {
			return template.HTML(blackfriday.Run([]byte(s)))
		},
	}).ParseGlob("templates/*.html"))
	config    Config
	sanitizer *bluemonday.Policy
)

func init() {
	config = LoadConfig()
	initDB()

	// Initialize HTML sanitizer with UGC policy (User Generated Content)
	sanitizer = bluemonday.UGCPolicy()
}

func main() {
	r := mux.NewRouter()

	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Routes
	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/admin/verify", verifyAdminHandler).Methods("POST")
	r.HandleFunc("/shifttypes", getShiftTypesHandler).Methods("GET")
	r.HandleFunc("/shifttypes", adminMiddleware(createShiftTypeHandler)).Methods("POST")
	r.HandleFunc("/shifttypes/{id}", adminMiddleware(updateShiftTypeHandler)).Methods("PUT")
	r.HandleFunc("/shifttypes/{id}", adminMiddleware(deleteShiftTypeHandler)).Methods("DELETE")
	r.HandleFunc("/shifttypes/{typeId}/shifts", adminMiddleware(createShiftHandler)).Methods("POST")
	r.HandleFunc("/shifttypes/{typeId}/shifts/{shiftId}", adminMiddleware(deleteShiftHandler)).Methods("DELETE")
	r.HandleFunc("/shifttypes/{typeId}/shifts/{shiftId}/join", joinShiftHandler).Methods("POST")
	r.HandleFunc("/shifttypes/{typeId}/shifts/{shiftId}/leave", leaveShiftHandler).Methods("POST")
	r.HandleFunc("/shifttypes/{typeId}/shifts/{shiftId}/participants/{name}", leaveShiftHandler).Methods("DELETE")
	r.HandleFunc("/shifttypes/{id}/sort", adminMiddleware(updateShiftTypeSort))

	addr := fmt.Sprintf(":%s", config.Port)
	log.Printf("Server starting on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func adminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		adminToken := r.Header.Get("X-Admin-Token")
		if adminToken != config.AdminPassword {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func verifyAdminHandler(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if data.Password != config.AdminPassword {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	shiftTypes, err := getShiftTypes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Config     Config
		ShiftTypes map[string]ShiftType
	}{
		Config:     config,
		ShiftTypes: shiftTypes,
	}

	templates.ExecuteTemplate(w, "index.html", data)
}

func getShiftTypesHandler(w http.ResponseWriter, r *http.Request) {
	shiftTypes, err := getShiftTypes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(shiftTypes)
}

func createShiftTypeHandler(w http.ResponseWriter, r *http.Request) {
	var shiftType ShiftType
	if err := json.NewDecoder(r.Body).Decode(&shiftType); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := validateAndSanitizeShiftType(&shiftType); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := createShiftType(shiftType); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(shiftType)
}

func updateShiftTypeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var shiftType ShiftType
	if err := json.NewDecoder(r.Body).Decode(&shiftType); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := validateAndSanitizeShiftType(&shiftType); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := updateShiftType(id, shiftType); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(shiftType)
}

func deleteShiftTypeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := deleteShiftType(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func createShiftHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	typeId := vars["typeId"]

	var shift Shift
	if err := json.NewDecoder(r.Body).Decode(&shift); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := validateShift(&shift); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := createShift(typeId, shift); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(shift)
}

func deleteShiftHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	typeId := vars["typeId"]
	shiftId := vars["shiftId"]

	if err := deleteShift(typeId, shiftId); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func joinShiftHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	typeId := vars["typeId"]
	shiftId := vars["shiftId"]

	var data struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sanitizedName, err := validateAndSanitizeParticipant(data.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if shift type requires password
	shiftTypes, err := getShiftTypes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	shiftType, exists := shiftTypes[typeId]
	if !exists {
		http.Error(w, "Shift type not found", http.StatusNotFound)
		return
	}

	if shiftType.Password != "" && shiftType.Password != data.Password {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	if err := joinShift(typeId, shiftId, sanitizedName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func leaveShiftHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	typeId := vars["typeId"]
	shiftId := vars["shiftId"]

	var data struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sanitizedName, err := validateAndSanitizeParticipant(data.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if shift type requires password
	shiftTypes, err := getShiftTypes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	shiftType, exists := shiftTypes[typeId]
	if !exists {
		http.Error(w, "Shift type not found", http.StatusNotFound)
		return
	}

	if shiftType.Password != "" && shiftType.Password != data.Password {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	if err := leaveShift(typeId, shiftId, sanitizedName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Input validation and sanitization functions
func validateAndSanitizeShiftType(st *ShiftType) error {
	// Validate titles (required, length between 2-100 chars, no special chars except common punctuation)
	titleRegex := regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,!?()]+$`)

	// Validate German title
	if len(st.TitleDE) < 2 || len(st.TitleDE) > 100 || !titleRegex.MatchString(st.TitleDE) {
		return fmt.Errorf("invalid German title")
	}

	// Validate English title
	if len(st.TitleEN) < 2 || len(st.TitleEN) > 100 || !titleRegex.MatchString(st.TitleEN) {
		return fmt.Errorf("invalid English title")
	}

	// Validate location (optional, but if present must be 2-100 chars)
	if st.Location != "" {
		if len(st.Location) < 2 || len(st.Location) > 100 || !titleRegex.MatchString(st.Location) {
			return fmt.Errorf("invalid location")
		}
	}

	// Sanitize all string fields
	st.TitleDE = sanitizer.Sanitize(st.TitleDE)
	st.TitleEN = sanitizer.Sanitize(st.TitleEN)
	st.DescriptionDE = sanitizer.Sanitize(st.DescriptionDE)
	st.DescriptionEN = sanitizer.Sanitize(st.DescriptionEN)
	st.Location = sanitizer.Sanitize(st.Location)

	return nil
}

func validateShift(s *Shift) error {
	// Validate start time (must be in the future)
	if s.StartTime.Before(time.Now()) {
		return fmt.Errorf("start time must be in the future")
	}

	// Validate duration (greater than 1 minute)
	if s.Duration < 1 {
		return fmt.Errorf("duration must be greater than 1 minute")
	}

	// Validate required participants (more than one)
	if s.Required < 1 {
		return fmt.Errorf("required participants must be more than one")
	}

	return nil
}

func validateAndSanitizeParticipant(name string) (string, error) {
	// Validate participant name (2-50 chars, letters, numbers, spaces, and common punctuation)
	nameRegex := regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,]+$`)
	if len(name) < 2 || len(name) > 50 || !nameRegex.MatchString(name) {
		return "", fmt.Errorf("invalid participant name")
	}

	// Sanitize and return the name
	return sanitizer.Sanitize(name), nil
}

type sortOrderRequest struct {
	SortOrder int `json:"sortOrder"`
}

func updateShiftTypeSort(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req sortOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := updateShiftTypeOrder(id, req.SortOrder); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
