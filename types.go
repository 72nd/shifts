package main

import "time"

type Event struct {
	ID            string    `json:"id"`        // UUID for the event
	Title         string    `json:"title"`     // Event title
	StartDate     time.Time `json:"startDate"` // Event start date
	EndDate       time.Time `json:"endDate"`   // Event end date
	AdminPassword string    `json:"-"`         // Admin password for this event
	CreatedAt     time.Time `json:"createdAt"` // When the event was created
}

type ShiftType struct {
	ID            string  `json:"id"`
	EventID       string  `json:"eventId"` // Reference to the event
	TitleDE       string  `json:"titleDE"`
	TitleEN       string  `json:"titleEN"`
	DescriptionDE string  `json:"descriptionDE"`
	DescriptionEN string  `json:"descriptionEN"`
	Location      string  `json:"location"`
	Password      string  `json:"password"`
	SortOrder     int     `json:"sortOrder"`
	Shifts        []Shift `json:"shifts"`
}

type Shift struct {
	ID           string    `json:"id"`
	StartTime    time.Time `json:"startTime"`
	Duration     int       `json:"duration"`
	Required     int       `json:"required"`
	Participants []string  `json:"participants"`
}

// Configuration for a specific event
type EventConfig struct {
	DayZero   string `json:"dayZero"`   // Reference date for relative day calculations
	DebugMode bool   `json:"debugMode"` // Debug mode setting per event
}
