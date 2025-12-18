package web

import (
	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/model"
)

// =============================================================================
// Shift Plan Mappers
// =============================================================================

// ShiftPlanToResponse converts a db.ShiftPlan to a model.ShiftPlanResponse
func ShiftPlanToResponse(plan *db.ShiftPlan) model.ShiftPlanResponse {
	return model.ShiftPlanResponse{
		ID:                plan.ID,
		Token:             plan.Token,
		StartDate:         plan.StartDate,
		EndDate:           plan.EndDate,
		CreatedAt:         plan.CreatedAt,
		IsMultilingual:    plan.IsMultilingual,
		TitleDE:           plan.TitleDE,
		TitleEN:           plan.TitleEN,
		DescriptionDE:     plan.DescriptionDE,
		DescriptionEN:     plan.DescriptionEN,
		DescriptionDEHTML: plan.DescriptionDEHTML,
		DescriptionENHTML: plan.DescriptionENHTML,
		DayBoundaryTime:   plan.DayBoundaryTime,
		CollaborativeMode: plan.CollaborativeMode,
	}
}

// CreateShiftPlanRequestToNew converts a model.CreateShiftPlanRequest to a db.NewShiftPlan
func CreateShiftPlanRequestToNew(req *model.CreateShiftPlanRequest) db.NewShiftPlan {
	return db.NewShiftPlan{
		TitleDE:           req.TitleDE,
		TitleEN:           req.TitleEN,
		DescriptionDE:     req.DescriptionDE,
		DescriptionEN:     req.DescriptionEN,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		IsMultilingual:    req.IsMultilingual,
		Password:          req.Password,
		DayBoundaryTime:   req.DayBoundaryTime,
		CollaborativeMode: req.CollaborativeMode,
	}
}

// UpdateShiftPlanRequestToNew converts a model.ShiftPlanUpdateRequest to a db.NewShiftPlan
func UpdateShiftPlanRequestToNew(req *model.ShiftPlanUpdateRequest) db.NewShiftPlan {
	return db.NewShiftPlan{
		TitleDE:           req.TitleDE,
		TitleEN:           req.TitleEN,
		DescriptionDE:     req.DescriptionDE,
		DescriptionEN:     req.DescriptionEN,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		IsMultilingual:    req.IsMultilingual,
		DayBoundaryTime:   req.DayBoundaryTime,
		CollaborativeMode: req.CollaborativeMode,
		// Note: Password is not included in updates
	}
}

// =============================================================================
// Shift Type Mappers
// =============================================================================

// ShiftTypeToResponse converts a db.ShiftType to a model.ShiftTypeResponse
func ShiftTypeToResponse(shiftType *db.ShiftType) model.ShiftTypeResponse {
	response := model.ShiftTypeResponse{
		ID:                shiftType.ID,
		PlanID:            shiftType.PlanID,
		SortOrder:         shiftType.SortOrder,
		TitleDE:           shiftType.TitleDE,
		TitleEN:           shiftType.TitleEN,
		DescriptionDE:     shiftType.DescriptionDE,
		DescriptionEN:     shiftType.DescriptionEN,
		DescriptionDEHTML: shiftType.DescriptionDEHTML,
		DescriptionENHTML: shiftType.DescriptionENHTML,
		LocationDE:        shiftType.LocationDE,
		LocationEN:        shiftType.LocationEN,
	}

	// Convert nested shifts if present
	if len(shiftType.Shifts) > 0 {
		response.Shifts = ShiftsToResponse(shiftType.Shifts)
	}

	return response
}

// ShiftTypesToResponse converts a slice of db.ShiftType to a slice of model.ShiftTypeResponse
func ShiftTypesToResponse(shiftTypes []db.ShiftType) []model.ShiftTypeResponse {
	responses := make([]model.ShiftTypeResponse, len(shiftTypes))
	for i, shiftType := range shiftTypes {
		responses[i] = ShiftTypeToResponse(&shiftType)
	}
	return responses
}

// CreateShiftTypeRequestToNew converts a model.CreateShiftTypeRequest to a db.NewShiftType
func CreateShiftTypeRequestToNew(req *model.CreateShiftTypeRequest, planID int64) db.NewShiftType {
	return db.NewShiftType{
		PlanID:        planID,
		Password:      req.Password,
		TitleDE:       req.TitleDE,
		TitleEN:       req.TitleEN,
		DescriptionDE: req.DescriptionDE,
		DescriptionEN: req.DescriptionEN,
		LocationDE:    req.LocationDE,
		LocationEN:    req.LocationEN,
	}
}

// =============================================================================
// Shift Mappers
// =============================================================================

// ShiftToResponse converts a db.Shift to a model.ShiftResponse
func ShiftToResponse(shift *db.Shift) model.ShiftResponse {
	response := model.ShiftResponse{
		ID:            shift.ID,
		TypeID:        shift.TypeID,
		StartTime:     shift.StartTime,
		Duration:      shift.Duration,
		Required:      shift.Required,
		CommentDE:     shift.CommentDE,
		CommentEN:     shift.CommentEN,
		CommentDEHTML: shift.CommentDEHTML,
		CommentENHTML: shift.CommentENHTML,
		IsLocked:      shift.IsLocked,
	}

	// Convert nested participants if present
	if len(shift.Participants) > 0 {
		response.Participants = ParticipantsToResponse(shift.Participants)
	}

	return response
}

// ShiftsToResponse converts a slice of db.Shift to a slice of model.ShiftResponse
func ShiftsToResponse(shifts []db.Shift) []model.ShiftResponse {
	responses := make([]model.ShiftResponse, len(shifts))
	for i, shift := range shifts {
		responses[i] = ShiftToResponse(&shift)
	}
	return responses
}

// =============================================================================
// Participant Mappers
// =============================================================================

// ParticipantToResponse converts a db.Participant to a model.ParticipantResponse
func ParticipantToResponse(participant *db.Participant) model.ParticipantResponse {
	return model.ParticipantResponse{
		ID:   participant.ID,
		Name: participant.Name,
	}
}

// ParticipantsToResponse converts a slice of db.Participant to a slice of model.ParticipantResponse
func ParticipantsToResponse(participants []db.Participant) []model.ParticipantResponse {
	responses := make([]model.ParticipantResponse, len(participants))
	for i, participant := range participants {
		responses[i] = ParticipantToResponse(&participant)
	}
	return responses
}

// =============================================================================
// Stats Mappers
// =============================================================================

// StatsToResponse converts a db.Stats to a model.StatsResponse
func StatsToResponse(stats *db.Stats) model.StatsResponse {
	return model.StatsResponse{
		Overview:    StatsOverviewToResponse(&stats.Overview),
		HelperStats: HelperStatsToResponse(stats.HelperStats),
		TypeStats:   TypeStatsToResponse(stats.TypeStats),
		DailyStats:  DailyStatsToResponse(stats.DailyStats),
	}
}

// StatsOverviewToResponse converts a db.StatsOverview to a model.StatsOverviewResponse
func StatsOverviewToResponse(overview *db.StatsOverview) model.StatsOverviewResponse {
	return model.StatsOverviewResponse{
		TotalShifts:       overview.TotalShifts,
		OpenShifts:        overview.OpenShifts,
		TotalRequired:     overview.TotalRequired,
		TotalFilled:       overview.TotalFilled,
		TotalHours:        overview.TotalHours,
		FilledHours:       overview.FilledHours,
		PastHours:         overview.PastHours,
		FutureHours:       overview.FutureHours,
		UniqueHelpers:     overview.UniqueHelpers,
		AvgHoursPerHelper: overview.AvgHoursPerHelper,
	}
}

// HelperStatsToResponse converts a slice of db.HelperStats to a slice of model.HelperStatsResponse
func HelperStatsToResponse(helperStats []db.HelperStats) []model.HelperStatsResponse {
	responses := make([]model.HelperStatsResponse, len(helperStats))
	for i, stats := range helperStats {
		responses[i] = model.HelperStatsResponse{
			Name:         stats.Name,
			TotalShifts:  stats.TotalShifts,
			TotalHours:   stats.TotalHours,
			PastHours:    stats.PastHours,
			FutureHours:  stats.FutureHours,
			ShiftDetails: ShiftDetailsToResponse(stats.ShiftDetails),
		}
	}
	return responses
}

// ShiftDetailsToResponse converts a slice of db.ShiftDetail to a slice of model.ShiftDetailResponse
func ShiftDetailsToResponse(shiftDetails []db.ShiftDetail) []model.ShiftDetailResponse {
	responses := make([]model.ShiftDetailResponse, len(shiftDetails))
	for i, detail := range shiftDetails {
		responses[i] = model.ShiftDetailResponse{
			Date:     detail.Date,
			TypeID:   detail.TypeID,
			TypeName: detail.TypeName,
			Duration: detail.Duration,
			Status:   detail.Status,
		}
	}
	return responses
}

// TypeStatsToResponse converts a slice of db.TypeStats to a slice of model.TypeStatsResponse
func TypeStatsToResponse(typeStats []db.TypeStats) []model.TypeStatsResponse {
	responses := make([]model.TypeStatsResponse, len(typeStats))
	for i, stats := range typeStats {
		responses[i] = model.TypeStatsResponse{
			ID:            stats.ID,
			TitleDE:       stats.TitleDE,
			TitleEN:       stats.TitleEN,
			TotalShifts:   stats.TotalShifts,
			TotalRequired: stats.TotalRequired,
			TotalFilled:   stats.TotalFilled,
			TotalHours:    stats.TotalHours,
			FilledHours:   stats.FilledHours,
			OpenPositions: stats.OpenPositions,
			PercentFilled: stats.PercentFilled,
		}
	}
	return responses
}

// DailyStatsToResponse converts a slice of db.DailyStats to a slice of model.DailyStatsResponse
func DailyStatsToResponse(dailyStats []db.DailyStats) []model.DailyStatsResponse {
	responses := make([]model.DailyStatsResponse, len(dailyStats))
	for i, stats := range dailyStats {
		responses[i] = model.DailyStatsResponse{
			Date:          stats.Date,
			TotalShifts:   stats.TotalShifts,
			TotalRequired: stats.TotalRequired,
			TotalFilled:   stats.TotalFilled,
			TotalHours:    stats.TotalHours,
			FilledHours:   stats.FilledHours,
			OpenPositions: stats.OpenPositions,
			PercentFilled: stats.PercentFilled,
		}
	}
	return responses
}
