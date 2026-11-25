package model

// =============================================================================
// Coordination DTOs (see prj/coordination-plan.md)
// =============================================================================

// CoordinationOp sets the status of one person in one shift
// @Description One operation of a coordination apply request
type CoordinationOp struct {
	// Shift the operation refers to
	ShiftID int64 `json:"shift_id" example:"42"`
	// Name of the person (matched by normalised name)
	Name string `json:"name" example:"Anna"`
	// New status: signed_up, assigned, declined, or remove (deletes the participation)
	Status string `json:"status" example:"assigned" enums:"signed_up,assigned,declined,remove"`
} // @name CoordinationOp

// CoordinationApplyRequest applies many operations atomically
// @Description Request body for applying coordination operations (max 2000) in one transaction
type CoordinationApplyRequest struct {
	// Operations in order
	Ops []CoordinationOp `json:"ops"`
	// Shifts to lock after the operations
	Lock []int64 `json:"lock"`
} // @name CoordinationApplyRequest

// CoordinationCreated reports a participation created by an operation
// @Description A participation that was created by an apply request
type CoordinationCreated struct {
	ShiftID       int64  `json:"shift_id" example:"42"`
	ParticipantID int64  `json:"participant_id" example:"7"`
	Name          string `json:"name" example:"Anna"`
} // @name CoordinationCreated

// CoordinationApplyResponse is the result of an apply request
// @Description Result of an apply request
type CoordinationApplyResponse struct {
	// Number of operations applied
	Applied int `json:"applied" example:"3"`
	// Participations that were created (for undo)
	Created []CoordinationCreated `json:"created"`
} // @name CoordinationApplyResponse

// =============================================================================
// Person fields
// =============================================================================

// FieldOption is one option of a choice field (the position is the level)
// @Description Option of a choice field. Omit the id for new options.
type FieldOption struct {
	ID string `json:"id,omitempty" example:"a1b2c3d4"`
	DE string `json:"de" example:"Erfahren"`
	EN string `json:"en" example:"Experienced"`
} // @name FieldOption

// PlanFieldRequest creates or updates a person field
// @Description Request body for creating or updating a person field. The kind is only used on create.
type PlanFieldRequest struct {
	LabelDE *string `json:"label_de" example:"Erfahrung"`
	LabelEN string  `json:"label_en" example:"Experience"`
	// choice, number, boolean or text
	Kind string `json:"kind" example:"choice" enums:"choice,number,boolean,text"`
	// Options of a choice field (2 to 10), lowest level first
	Options []FieldOption `json:"options"`
	// Who fills in the field: participant or admin
	EditableBy string `json:"editable_by" example:"participant" enums:"participant,admin"`
	// Required when signing up (only for participant fields)
	Required bool `json:"required"`
} // @name PlanFieldRequest

// PlanFieldResponse is a person field of a plan
// @Description Person field definition (without values)
type PlanFieldResponse struct {
	ID         int64         `json:"id" example:"1"`
	SortOrder  int           `json:"sort_order" example:"0"`
	LabelDE    *string       `json:"label_de" example:"Erfahrung"`
	LabelEN    string        `json:"label_en" example:"Experience"`
	Kind       string        `json:"kind" example:"choice"`
	Options    []FieldOption `json:"options"`
	EditableBy string        `json:"editable_by" example:"participant"`
	Required   bool          `json:"required"`
} // @name PlanFieldResponse

// FieldOrderRequest sets the order of the person fields
// @Description Request body for updating the display order of person fields
type FieldOrderRequest struct {
	FieldIDs []int64 `json:"field_ids"`
} // @name FieldOrderRequest

// =============================================================================
// People
// =============================================================================

// PersonResponse is the stored data of a person
// @Description Person data of a plan (admin only)
type PersonResponse struct {
	Name            string    `json:"name" example:"Anna"`
	AvailableFrom   *DateTime `json:"available_from" swaggertype:"string" example:"2024-12-20T16:00:00"`
	AvailableUntil  *DateTime `json:"available_until" swaggertype:"string" example:"2024-12-22T14:00:00"`
	TargetMinutes   *int      `json:"target_minutes" example:"720"`
	PrefersLongDays bool      `json:"prefers_long_days"`
	// Field values by field ID; a choice value is the option ID
	Values map[string]any `json:"values"`
} // @name PersonResponse

// PersonWriteRequest creates or completely replaces a person
// @Description Request body for creating or replacing a person (admin)
type PersonWriteRequest struct {
	Name            string         `json:"name" example:"Anna"`
	AvailableFrom   *DateTime      `json:"available_from" swaggertype:"string" example:"2024-12-20T16:00:00"`
	AvailableUntil  *DateTime      `json:"available_until" swaggertype:"string" example:"2024-12-22T14:00:00"`
	TargetMinutes   *int           `json:"target_minutes" example:"720"`
	PrefersLongDays bool           `json:"prefers_long_days"`
	Values          map[string]any `json:"values"`
} // @name PersonWriteRequest

// PersonInput is what a participant can send when signing up
// @Description Optional person details when signing up. Only fields editable by participants are allowed; a null value clears a field.
type PersonInput struct {
	AvailableFrom  *DateTime      `json:"available_from,omitempty" swaggertype:"string" example:"2024-12-20T16:00:00"`
	AvailableUntil *DateTime      `json:"available_until,omitempty" swaggertype:"string" example:"2024-12-22T14:00:00"`
	Values         map[string]any `json:"values,omitempty"`
} // @name PersonInput

// PeopleImportRequest creates many people by name
// @Description Request body for creating people from a list of names (max 500)
type PeopleImportRequest struct {
	Names []string `json:"names"`
} // @name PeopleImportRequest

// PeopleImportResponse counts the result of an import
// @Description Result of a people import
type PeopleImportResponse struct {
	Created int `json:"created" example:"12"`
	Skipped int `json:"skipped" example:"3"`
} // @name PeopleImportResponse

// PersonRenameRequest renames or merges a person
// @Description Request body for renaming a person in all participations of the plan
type PersonRenameRequest struct {
	From string `json:"from" example:"anna"`
	To   string `json:"to" example:"Anna M."`
	// Merge into an existing person with the target name
	Merge bool `json:"merge"`
} // @name PersonRenameRequest

// PersonLookupResponse is the public view of a person's participant fields
// @Description Details a person gave themselves (only fields editable by participants)
type PersonLookupResponse struct {
	Exists         bool           `json:"exists"`
	AvailableFrom  *DateTime      `json:"available_from" swaggertype:"string" example:"2024-12-20T16:00:00"`
	AvailableUntil *DateTime      `json:"available_until" swaggertype:"string" example:"2024-12-22T14:00:00"`
	Values         map[string]any `json:"values"`
} // @name PersonLookupResponse
