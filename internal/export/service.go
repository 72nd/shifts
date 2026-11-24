package export

import (
	"fmt"
	"sort"
	"time"

	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/model"
)

// Service handles the export functionality for shift plans
type Service struct {
	db        *db.DB
	exporters map[string]Exporter
}

// NewService creates a new export service
func NewService(db *db.DB) *Service {
	return &Service{
		db:        db,
		exporters: make(map[string]Exporter),
	}
}

// RegisterExporter registers an exporter for a specific format
func (s *Service) RegisterExporter(exporter Exporter) {
	s.exporters[exporter.Format()] = exporter
}

// GetExporter returns the exporter for the specified format
func (s *Service) GetExporter(format string) (Exporter, error) {
	exporter, ok := s.exporters[format]
	if !ok {
		return nil, NewError(ErrInvalidInput, fmt.Sprintf("unsupported export format: %s", format), nil)
	}
	return exporter, nil
}

// ShiftsByDay represents shifts grouped by their logical day
type ShiftsByDay struct {
	Date     model.Date `json:"date"`
	DayCount int        `json:"day_count"`
	Shifts   []db.Shift `json:"shifts"`
}

// ShiftTypeWithDays represents a shift type with its shifts grouped by days
type ShiftTypeWithDays struct {
	*db.ShiftType
	ShiftDays []ShiftsByDay `json:"shift_days"`
}

// ExportData represents the complete export structure for a shift plan
type ExportData struct {
	Plan          *db.ShiftPlan        `json:"plan"`
	ShiftTypes    []*ShiftTypeWithDays `json:"shift_types"`
	ExportedAt    time.Time            `json:"exported_at"`
	AppliedFilter *ExportFilter        `json:"applied_filter,omitempty"`
}

// NewExportData creates a new ExportData instance and validates the input
func NewExportData(plan *db.ShiftPlan, types []*db.ShiftType) (*ExportData, error) {
	// Validate boundary time
	if _, err := model.ParseBoundary(plan.DayBoundaryTime); err != nil {
		return nil, NewError(ErrInvalidInput, fmt.Sprintf("invalid day boundary time in plan: %v", err), err)
	}

	// Sort shift types by sort_order
	sort.Slice(types, func(i, j int) bool {
		return types[i].SortOrder < types[j].SortOrder
	})

	// Convert types to types with days
	typesWithDays := make([]*ShiftTypeWithDays, len(types))
	for i, t := range types {
		typesWithDays[i] = &ShiftTypeWithDays{
			ShiftType: t,
			ShiftDays: []ShiftsByDay{},
		}
	}

	return &ExportData{
		Plan:       plan,
		ShiftTypes: typesWithDays,
		ExportedAt: time.Now().In(plan.Location()),
	}, nil
}

// groupShiftsByDay groups shifts by their logical day
func groupShiftsByDay(shifts []db.Shift, boundaryTime string, startDate model.Date) ([]ShiftsByDay, error) {
	if len(shifts) == 0 {
		return []ShiftsByDay{}, nil
	}

	// Create a map of date -> shifts
	shiftMap := make(map[model.Date][]db.Shift)

	// Group shifts by day
	for _, shift := range shifts {
		day, err := getShiftDay(shift, boundaryTime)
		if err != nil {
			return nil, fmt.Errorf("failed to get shift day: %w", err)
		}
		shiftMap[day] = append(shiftMap[day], shift)
	}

	// Convert map to sorted slice
	var days []model.Date
	for day := range shiftMap {
		days = append(days, day)
	}
	sort.Slice(days, func(i, j int) bool {
		return days[i].Before(days[j].Time)
	})

	// Create sorted result
	result := make([]ShiftsByDay, len(days))
	for i, day := range days {
		// Sort shifts within the day by start time
		dayShifts := shiftMap[day]
		sort.Slice(dayShifts, func(i, j int) bool {
			return dayShifts[i].StartTime.Before(dayShifts[j].StartTime.Time)
		})

		result[i] = ShiftsByDay{
			Date:     day,
			DayCount: int(day.Sub(startDate.Time).Hours()/24+0.5) + 1,
			Shifts:   dayShifts,
		}
	}

	return result, nil
}

// SortShifts sorts all shifts in all shift types according to the plan's day boundary time
func (e *ExportData) SortShifts() error {
	for _, t := range e.ShiftTypes {
		if t.Shifts == nil {
			continue
		}

		// Group shifts by day
		shiftDays, err := groupShiftsByDay(t.Shifts, e.Plan.DayBoundaryTime, e.Plan.StartDate)
		if err != nil {
			return fmt.Errorf("failed to group shifts for type %d: %w", t.ID, err)
		}

		// Update the shift days
		t.ShiftDays = shiftDays
		t.Shifts = nil // Clear the original shifts array to avoid duplicate data
	}
	return nil
}

// ExportPlanWithFilter exports a shift plan with all its data and applies filters if provided
func (s *Service) ExportPlanWithFilter(token string, adminToken string, filter *ExportFilter) (*ExportData, error) {
	// Get plan
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		return nil, NewError(ErrLoadingPlan, "Failed to load shift plan", err)
	}
	if plan == nil {
		return nil, NewError(ErrInvalidToken, "Plan not found", nil)
	}

	// Verify admin token
	if ok, err := s.db.ValidateShiftPlanPassword(plan.ID, adminToken); !ok || err != nil {
		return nil, NewError(ErrUnauthorized, "Invalid admin token", err)
	}

	// Load all shift types
	types, err := s.loadShiftTypes(plan.ID)
	if err != nil {
		return nil, err
	}

	// Create export data
	exportData, err := NewExportData(plan, types)
	if err != nil {
		return nil, NewError(ErrInvalidInput, "Failed to create export data", err)
	}

	// Sort shifts
	if err := exportData.SortShifts(); err != nil {
		return nil, NewError(ErrInvalidInput, "Failed to sort shifts", err)
	}

	// Apply filters if provided and not empty
	if filter != nil && !filter.IsEmpty() {
		exportData = filter.ApplyFilters(exportData)
		// Store the applied filter in the export data for documentation
		exportData.AppliedFilter = filter
	}

	return exportData, nil
}

// Export exports a shift plan in the specified format with optional filtering
func (s *Service) Export(token string, adminToken string, format string, filter *ExportFilter) ([]byte, string, string, error) {
	// Get the exporter for the specified format
	exporter, err := s.GetExporter(format)
	if err != nil {
		return nil, "", "", err
	}

	// Export the plan with filter
	exportData, err := s.ExportPlanWithFilter(token, adminToken, filter)
	if err != nil {
		return nil, "", "", err
	}

	// Use the exporter to convert data
	data, err := exporter.Export(exportData)
	if err != nil {
		return nil, "", "", NewError(ErrInvalidInput, "Failed to export data", err)
	}

	// Generate filename
	filename := fmt.Sprintf("shift-plan-%s-%s.%s",
		exportData.Plan.Token,
		exportData.ExportedAt.Format("2006-01-02"),
		exporter.FileExtension())

	return data, exporter.ContentType(), filename, nil
}

// loadShiftTypes loads all shift types with their shifts and participants
func (s *Service) loadShiftTypes(planID int64) ([]*db.ShiftType, error) {
	types, err := s.db.GetShiftTypesByPlan(planID)
	if err != nil {
		return nil, NewError(ErrLoadingShiftTypes, "Failed to load shift types", err)
	}

	// Convert types slice to pointer slice
	typesPtrs := make([]*db.ShiftType, len(types))
	for i := range types {
		typesPtrs[i] = &types[i]
		if err := s.loadShiftsForType(typesPtrs[i]); err != nil {
			return nil, err
		}
	}

	return typesPtrs, nil
}

// loadShiftsForType loads all shifts with their participants for a shift type
func (s *Service) loadShiftsForType(shiftType *db.ShiftType) error {
	shifts, err := s.db.GetShiftsByType(shiftType.ID)
	if err != nil {
		return NewError(ErrLoadingShifts, "Failed to load shifts", err)
	}

	// Load participants for each shift
	for i := range shifts {
		participants, err := s.db.GetParticipantsByShift(shifts[i].ID)
		if err != nil {
			return NewError(ErrLoadingParticipants, "Failed to load participants", err)
		}
		shifts[i].Participants = participants
	}

	shiftType.Shifts = shifts
	return nil
}
