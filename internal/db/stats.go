package db

import (
	"fmt"
	"sort"
	"time"
)

// Stats represents all statistics for a shift plan
type Stats struct {
	Overview    StatsOverview `json:"overview"`
	HelperStats []HelperStats `json:"helper_stats"`
	TypeStats   []TypeStats   `json:"type_stats"`
	DailyStats  []DailyStats  `json:"daily_stats"`
}

// StatsOverview contains the overview statistics
type StatsOverview struct {
	TotalShifts       int     `json:"total_shifts"`
	OpenShifts        int     `json:"open_shifts"`
	TotalRequired     int     `json:"total_required"`
	TotalFilled       int     `json:"total_filled"`
	TotalHours        float64 `json:"total_hours"`
	FilledHours       float64 `json:"filled_hours"`
	PastHours         float64 `json:"past_hours"`
	FutureHours       float64 `json:"future_hours"`
	UniqueHelpers     int     `json:"unique_helpers"`
	AvgHoursPerHelper float64 `json:"avg_hours_per_helper"`
}

// HelperStats contains statistics for a single helper
type HelperStats struct {
	Name         string        `json:"name"`
	TotalShifts  int           `json:"total_shifts"`
	TotalHours   float64       `json:"total_hours"`
	PastHours    float64       `json:"past_hours"`
	FutureHours  float64       `json:"future_hours"`
	ShiftDetails []ShiftDetail `json:"shift_details"`
}

// ShiftDetail contains details about a single shift
type ShiftDetail struct {
	Date     time.Time `json:"date"`
	TypeID   int64     `json:"type_id"`
	TypeName string    `json:"type_name"`
	Duration float64   `json:"duration"`
	Status   string    `json:"status"` // "past", "current", "future"
}

// TypeStats contains statistics for a shift type
type TypeStats struct {
	ID            int64   `json:"id"`
	TitleDE       string  `json:"title_de"`
	TitleEN       string  `json:"title_en"`
	TotalShifts   int     `json:"total_shifts"`
	TotalRequired int     `json:"total_required"`
	TotalFilled   int     `json:"total_filled"`
	TotalHours    float64 `json:"total_hours"`
	FilledHours   float64 `json:"filled_hours"`
	OpenPositions int     `json:"open_positions"`
	PercentFilled int     `json:"percent_filled"`
}

// DailyStats contains statistics for a single day
type DailyStats struct {
	Date          time.Time `json:"date"`
	TotalShifts   int       `json:"total_shifts"`
	TotalRequired int       `json:"total_required"`
	TotalFilled   int       `json:"total_filled"`
	TotalHours    float64   `json:"total_hours"`
	FilledHours   float64   `json:"filled_hours"`
	OpenPositions int       `json:"open_positions"`
	PercentFilled int       `json:"percent_filled"`
}

// GetShiftPlanStats calculates all statistics for a shift plan
func (db *DB) GetShiftPlanStats(planID int64) (*Stats, error) {
	// Get all shift types with their shifts and participants
	types, err := db.GetShiftTypesByPlan(planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shift types: %w", err)
	}

	// Get the plan to access the day boundary time
	var plan ShiftPlan
	err = db.QueryRow(`
		SELECT day_boundary_time
		FROM shift_plans
		WHERE id = ?
	`, planID).Scan(&plan.DayBoundaryTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	// Parse the day boundary time
	boundaryHour, boundaryMinute := 0, 0
	if plan.DayBoundaryTime != "" {
		_, err := fmt.Sscanf(plan.DayBoundaryTime, "%d:%d", &boundaryHour, &boundaryMinute)
		if err != nil {
			return nil, fmt.Errorf("invalid day boundary time format: %w", err)
		}
	}

	now := time.Now()
	stats := &Stats{
		Overview:    StatsOverview{},
		HelperStats: make([]HelperStats, 0),
		TypeStats:   make([]TypeStats, 0),
		DailyStats:  make([]DailyStats, 0),
	}

	// Maps for aggregating data
	helperMap := make(map[string]*HelperStats)
	dailyMap := make(map[string]*DailyStats)
	uniqueHelperNames := make(map[string]bool)

	// Process each type and its shifts
	for _, t := range types {
		typeStats := TypeStats{
			ID:      t.ID,
			TitleDE: t.TitleDE,
			TitleEN: t.TitleEN,
		}

		for _, shift := range t.Shifts {
			startTime := shift.StartTime
			endTime := startTime.Add(time.Duration(shift.Duration) * time.Minute)
			shiftHours := float64(shift.Duration) / 60
			filledPositions := len(shift.Participants)

			// Adjust the day based on the boundary time
			shiftDay := startTime
			if startTime.Hour() < boundaryHour || (startTime.Hour() == boundaryHour && startTime.Minute() < boundaryMinute) {
				shiftDay = startTime.AddDate(0, 0, -1)
			}
			dayKey := shiftDay.Format("2006-01-02")

			// Update overview stats
			stats.Overview.TotalShifts++
			stats.Overview.TotalRequired += shift.Required
			stats.Overview.TotalFilled += filledPositions
			stats.Overview.TotalHours += shiftHours * float64(shift.Required)
			stats.Overview.FilledHours += shiftHours * float64(filledPositions)

			if filledPositions < shift.Required {
				stats.Overview.OpenShifts++
			}

			// Calculate past and future hours
			var status string
			if endTime.Before(now) {
				status = "past"
				stats.Overview.PastHours += shiftHours * float64(filledPositions)
			} else if startTime.After(now) {
				status = "future"
				stats.Overview.FutureHours += shiftHours * float64(filledPositions)
			} else {
				status = "current"
			}

			// Update type stats
			typeStats.TotalShifts++
			typeStats.TotalRequired += shift.Required
			typeStats.TotalFilled += filledPositions
			typeStats.TotalHours += shiftHours * float64(shift.Required)
			typeStats.FilledHours += shiftHours * float64(filledPositions)
			typeStats.OpenPositions += shift.Required - filledPositions

			// Update daily stats
			dailyStats, exists := dailyMap[dayKey]
			if !exists {
				dailyStats = &DailyStats{
					Date: shiftDay,
				}
				dailyMap[dayKey] = dailyStats
			}
			dailyStats.TotalShifts++
			dailyStats.TotalRequired += shift.Required
			dailyStats.TotalFilled += filledPositions
			dailyStats.TotalHours += shiftHours * float64(shift.Required)
			dailyStats.FilledHours += shiftHours * float64(filledPositions)
			dailyStats.OpenPositions += shift.Required - filledPositions

			// Update helper stats
			for _, p := range shift.Participants {
				uniqueHelperNames[p.Name] = true
				helperStats, exists := helperMap[p.Name]
				if !exists {
					helperStats = &HelperStats{
						Name:         p.Name,
						ShiftDetails: make([]ShiftDetail, 0),
					}
					helperMap[p.Name] = helperStats
				}
				helperStats.TotalShifts++
				helperStats.TotalHours += shiftHours
				if status == "past" {
					helperStats.PastHours += shiftHours
				} else if status == "future" {
					helperStats.FutureHours += shiftHours
				}
				helperStats.ShiftDetails = append(helperStats.ShiftDetails, ShiftDetail{
					Date:     shiftDay,
					TypeID:   t.ID,
					TypeName: t.TitleEN,
					Duration: shiftHours,
					Status:   status,
				})
			}
		}

		// Calculate percentage filled for type stats
		if typeStats.TotalRequired > 0 {
			typeStats.PercentFilled = int(float64(typeStats.TotalFilled) / float64(typeStats.TotalRequired) * 100)
		}
		stats.TypeStats = append(stats.TypeStats, typeStats)
	}

	// Calculate daily percentages and sort by date
	for _, dailyStats := range dailyMap {
		if dailyStats.TotalRequired > 0 {
			dailyStats.PercentFilled = int(float64(dailyStats.TotalFilled) / float64(dailyStats.TotalRequired) * 100)
		}
		stats.DailyStats = append(stats.DailyStats, *dailyStats)
	}
	sort.Slice(stats.DailyStats, func(i, j int) bool {
		return stats.DailyStats[i].Date.Before(stats.DailyStats[j].Date)
	})

	// Convert helper map to slice and sort by total hours
	stats.Overview.UniqueHelpers = len(uniqueHelperNames)
	if stats.Overview.UniqueHelpers > 0 {
		stats.Overview.AvgHoursPerHelper = stats.Overview.FilledHours / float64(stats.Overview.UniqueHelpers)
	}

	for _, helperStats := range helperMap {
		stats.HelperStats = append(stats.HelperStats, *helperStats)
	}
	sort.Slice(stats.HelperStats, func(i, j int) bool {
		return stats.HelperStats[i].TotalHours > stats.HelperStats[j].TotalHours
	})

	return stats, nil
}
