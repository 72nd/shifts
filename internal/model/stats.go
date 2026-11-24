package model

// =============================================================================
// Stats Response DTOs
// =============================================================================

// StatsResponse represents all statistics for a shift plan in API responses
// @Description Complete statistics data for a shift plan
type StatsResponse struct {
	// Overview statistics for the entire plan
	Overview StatsOverviewResponse `json:"overview"`
	// Statistics per helper/participant
	HelperStats []HelperStatsResponse `json:"helper_stats"`
	// Statistics per shift type
	TypeStats []TypeStatsResponse `json:"type_stats"`
	// Statistics per day
	DailyStats []DailyStatsResponse `json:"daily_stats"`
} // @name StatsResponse

// StatsOverviewResponse contains overview statistics
// @Description Overview statistics for a shift plan
type StatsOverviewResponse struct {
	// Total number of shifts in the plan
	TotalShifts int `json:"total_shifts" example:"100"`
	// Number of shifts that still have open positions
	OpenShifts int `json:"open_shifts" example:"15"`
	// Total number of required participants across all shifts
	TotalRequired int `json:"total_required" example:"300"`
	// Total number of filled positions
	TotalFilled int `json:"total_filled" example:"285"`
	// Total hours of all shifts (required positions * duration)
	TotalHours float64 `json:"total_hours" example:"600.5"`
	// Total filled hours (filled positions * duration)
	FilledHours float64 `json:"filled_hours" example:"570.5"`
	// Hours in past shifts
	PastHours float64 `json:"past_hours" example:"200.0"`
	// Hours in future shifts
	FutureHours float64 `json:"future_hours" example:"370.5"`
	// Number of unique helpers/participants
	UniqueHelpers int `json:"unique_helpers" example:"25"`
	// Average hours per helper
	AvgHoursPerHelper float64 `json:"avg_hours_per_helper" example:"22.82"`
} // @name StatsOverviewResponse

// HelperStatsResponse contains statistics for a single helper
// @Description Statistics for a single helper/participant
type HelperStatsResponse struct {
	// Name of the helper
	Name string `json:"name" example:"John Doe"`
	// Total number of shifts this helper is assigned to
	TotalShifts int `json:"total_shifts" example:"12"`
	// Total hours for this helper
	TotalHours float64 `json:"total_hours" example:"24.5"`
	// Hours in past shifts
	PastHours float64 `json:"past_hours" example:"10.0"`
	// Hours in future shifts
	FutureHours float64 `json:"future_hours" example:"14.5"`
	// Detailed breakdown of shifts for this helper
	ShiftDetails []ShiftDetailResponse `json:"shift_details"`
} // @name HelperStatsResponse

// ShiftDetailResponse contains details about a single shift in helper stats
// @Description Detailed information about a single shift for statistics
type ShiftDetailResponse struct {
	// Date of the shift
	Date Date `json:"date" swaggertype:"string" example:"2024-12-20"`
	// ID of the shift type
	TypeID int64 `json:"type_id" example:"1"`
	// Name of the shift type
	TypeName string `json:"type_name" example:"Kitchen Duty"`
	// Duration in hours
	Duration float64 `json:"duration" example:"2.0"`
	// Status of the shift (past, current, future)
	Status string `json:"status" example:"future"`
} // @name ShiftDetailResponse

// TypeStatsResponse contains statistics for a shift type
// @Description Statistics for a single shift type
type TypeStatsResponse struct {
	// Unique identifier of the shift type
	ID int64 `json:"id" example:"1"`
	// Title in German
	TitleDE string `json:"title_de" example:"Küchendienst"`
	// Title in English
	TitleEN string `json:"title_en" example:"Kitchen Duty"`
	// Total number of shifts of this type
	TotalShifts int `json:"total_shifts" example:"20"`
	// Total required participants for this type
	TotalRequired int `json:"total_required" example:"60"`
	// Total filled positions for this type
	TotalFilled int `json:"total_filled" example:"55"`
	// Total hours for this type (required * duration)
	TotalHours float64 `json:"total_hours" example:"120.0"`
	// Filled hours for this type
	FilledHours float64 `json:"filled_hours" example:"110.0"`
	// Number of open positions
	OpenPositions int `json:"open_positions" example:"5"`
	// Percentage filled
	PercentFilled int `json:"percent_filled" example:"92"`
} // @name TypeStatsResponse

// DailyStatsResponse contains statistics for a single day
// @Description Statistics for a single day
type DailyStatsResponse struct {
	// Date
	Date Date `json:"date" swaggertype:"string" example:"2024-12-20"`
	// Total number of shifts on this day
	TotalShifts int `json:"total_shifts" example:"10"`
	// Total required participants for this day
	TotalRequired int `json:"total_required" example:"30"`
	// Total filled positions for this day
	TotalFilled int `json:"total_filled" example:"28"`
	// Total hours for this day
	TotalHours float64 `json:"total_hours" example:"60.0"`
	// Filled hours for this day
	FilledHours float64 `json:"filled_hours" example:"56.0"`
	// Number of open positions
	OpenPositions int `json:"open_positions" example:"2"`
	// Percentage filled
	PercentFilled int `json:"percent_filled" example:"93"`
} // @name DailyStatsResponse
