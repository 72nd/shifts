package main

import "time"

type ShiftType struct {
	ID            string  `json:"id"`
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
