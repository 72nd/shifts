package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// AppError repräsentiert einen anwendungsspezifischen Fehler
type AppError struct {
	Code       string // z.B. "shift_full", "invalid_password"
	Message    string // Benutzerfreundliche Nachricht
	StatusCode int    // HTTP-Statuscode
	Err        error  // Ursprünglicher Fehler (optional)
}

// Error implementiert die error-Schnittstelle
func (e *AppError) Error() string {
	return e.Message
}

// NewAppError erstellt einen neuen AppError
func NewAppError(code string, message string, statusCode int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
	}
}

// RespondWithError sendet eine Fehlerantwort an den Client und loggt den Fehler
func RespondWithError(w http.ResponseWriter, logger *slog.Logger, err *AppError) {
	// Strukturiertes Logging des Fehlers
	attrs := []any{
		"code", err.Code,
		"status", err.StatusCode,
		"message", err.Message,
	}

	// Füge den ursprünglichen Fehler hinzu, falls vorhanden
	if err.Err != nil {
		attrs = append(attrs, "error", err.Err)
	}

	logger.Error("application error", attrs...)

	msg := fmt.Sprintf("%s: %s", err.Code, err.Message)
	msg = strings.TrimSpace(msg)

	// Sende Fehlerantwort an den Client
	http.Error(w, msg, err.StatusCode)
}

// RespondWithErrorAndLogger sendet eine Fehlerantwort an den Client und loggt den Fehler mit einem Logger-Interface
func RespondWithErrorAndLogger(w http.ResponseWriter, logger Logger, err *AppError) {
	// Logge den Fehler mit dem Logger-Interface
	if err.Err != nil {
		logger.Error("application error",
			"code", err.Code,
			"status", err.StatusCode,
			"message", err.Message,
			"error", err.Err,
		)
	} else {
		logger.Error("application error",
			"code", err.Code,
			"status", err.StatusCode,
			"message", err.Message,
		)
	}

	// Sende Fehlerantwort an den Client
	http.Error(w, err.Message, err.StatusCode)
}

// Fehlercodes
const (
	// Authentifizierungsfehler
	ErrInvalidPassword = "invalid_password"
	ErrUnauthorized    = "unauthorized"

	// Ressourcenfehler
	ErrResourceNotFound = "not_found"
	ErrShiftFull        = "shift_full"
	ErrDuplicateName    = "duplicate_name"

	// Validierungsfehler
	ErrInvalidInput = "invalid_input"
	ErrMissingField = "missing_field"

	// Serverfehler
	ErrInternalError = "internal_error"
	ErrDatabaseError = "database_error"
)
