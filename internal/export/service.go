package export

import (
	"fmt"
	"sort"
	"time"

	"github.com/72nd/shifts/internal/db"
)

// Service handles the export functionality for shift plans
type Service struct {
	db *db.DB
}

// NewService creates a new export service
func NewService(db *db.DB) *Service {
	return &Service{db: db}
}

// ShiftsByDay represents shifts grouped by their logical day
type ShiftsByDay struct {
	Date   time.Time  `json:"date"`
	Shifts []db.Shift `json:"shifts"`
}

// ShiftTypeWithDays represents a shift type with its shifts grouped by days
type ShiftTypeWithDays struct {
	*db.ShiftType
	ShiftDays []ShiftsByDay `json:"shift_days"`
}

// ExportData represents the complete export structure for a shift plan
type ExportData struct {
	Plan       *db.ShiftPlan        `json:"plan"`
	ShiftTypes []*ShiftTypeWithDays `json:"shift_types"`
	ExportedAt time.Time            `json:"exported_at"`
}

// NewExportData creates a new ExportData instance and validates the input
func NewExportData(plan *db.ShiftPlan, types []*db.ShiftType) (*ExportData, error) {
	// Validate boundary time
	if _, err := ParseTime(plan.DayBoundaryTime); err != nil {
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
		ExportedAt: time.Now(),
	}, nil
}

// groupShiftsByDay groups shifts by their logical day
func groupShiftsByDay(shifts []db.Shift, boundaryTime string) ([]ShiftsByDay, error) {
	if len(shifts) == 0 {
		return []ShiftsByDay{}, nil
	}

	// Create a map of date -> shifts
	shiftMap := make(map[time.Time][]db.Shift)

	// Group shifts by day
	for _, shift := range shifts {
		day, err := getShiftDay(shift, boundaryTime)
		if err != nil {
			return nil, fmt.Errorf("failed to get shift day: %w", err)
		}
		shiftMap[day] = append(shiftMap[day], shift)
	}

	// Convert map to sorted slice
	var days []time.Time
	for day := range shiftMap {
		days = append(days, day)
	}
	sort.Slice(days, func(i, j int) bool {
		return days[i].Before(days[j])
	})

	// Create sorted result
	result := make([]ShiftsByDay, len(days))
	for i, day := range days {
		// Sort shifts within the day by start time
		dayShifts := shiftMap[day]
		sort.Slice(dayShifts, func(i, j int) bool {
			return dayShifts[i].StartTime.Before(dayShifts[j].StartTime)
		})

		result[i] = ShiftsByDay{
			Date:   day,
			Shifts: dayShifts,
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
		shiftDays, err := groupShiftsByDay(t.Shifts, e.Plan.DayBoundaryTime)
		if err != nil {
			return fmt.Errorf("failed to group shifts for type %d: %w", t.ID, err)
		}

		// Update the shift days
		t.ShiftDays = shiftDays
		t.Shifts = nil // Clear the original shifts array to avoid duplicate data
	}
	return nil
}

// GenerateFilename generates a filename for the export based on the plan token
func (e *ExportData) GenerateFilename() string {
	return fmt.Sprintf("shift-plan-%s-%s.json", e.Plan.Token, e.ExportedAt.Format("2006-01-02"))
}

// ExportPlan exports a shift plan with all its data
func (s *Service) ExportPlan(token string, adminToken string) (*ExportData, error) {
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

	return exportData, nil
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
