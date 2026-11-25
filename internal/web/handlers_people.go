package web

import (
	"errors"
	"net/http"

	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/model"
	"github.com/72nd/shifts/internal/util"
	"github.com/72nd/shifts/internal/validation"
	"github.com/gorilla/mux"
)

// peopleError maps database errors of the people operations to a response.
func (s *Server) peopleError(w http.ResponseWriter, err error, message string) {
	switch {
	case errors.Is(err, db.ErrPersonNotFound):
		s.RespondWithError(w, NewAppError(ErrResourceNotFound, "Person not found", http.StatusNotFound, err))
	case errors.Is(err, db.ErrPersonExists):
		s.RespondWithError(w, NewAppError(ErrPersonExists, "A person with this name already exists", http.StatusConflict, err))
	default:
		s.RespondWithError(w, NewAppError(ErrDatabaseError, message, http.StatusInternalServerError, err))
	}
}

// handleListPeople returns the stored person rows
// @Summary List people
// @Description Returns all stored person data of the plan (attendance, target, person fields). People who only have participations and no stored data are not listed; clients add them from the participations. Requires admin authentication.
// @Tags People
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Success 200 {array} model.PersonResponse "Stored people"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/people [get]
func (s *Server) handleListPeople(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	people, err := s.db.ListPeople(plan.ID)
	if err != nil {
		s.peopleError(w, err, "Failed to list people")
		return
	}
	response := make([]model.PersonResponse, len(people))
	for i := range people {
		response[i] = PersonToResponse(&people[i])
	}
	s.respondJSON(w, response)
}

// handleUpsertPerson creates or replaces a person
// @Summary Create or update person
// @Description Creates or completely replaces the stored data of a person, identified by name. Requires admin authentication.
// @Tags People
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param person body model.PersonWriteRequest true "Person data"
// @Success 200 {object} model.PersonResponse "Stored person"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/people [put]
func (s *Server) handleUpsertPerson(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	var req model.PersonWriteRequest
	if !s.parseJSONTimes(w, r, &req) {
		return
	}
	if result := validation.ValidatePersonWrite(&req, plan.Fields); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}
	person, err := s.db.UpsertPerson(plan.ID, db.PlanPersonWrite{
		Name:            req.Name,
		AvailableFrom:   req.AvailableFrom,
		AvailableUntil:  req.AvailableUntil,
		TargetMinutes:   req.TargetMinutes,
		PrefersLongDays: req.PrefersLongDays,
		Values:          req.Values,
	})
	if err != nil {
		s.peopleError(w, err, "Failed to save person")
		return
	}
	s.notifyPlanChanged(plan.ID, scopeParticipants)
	s.respondJSON(w, PersonToResponse(person))
}

// handleImportPeople creates people from a list of names
// @Summary Import people
// @Description Creates people from a list of names (max 500). Names that already exist, as stored person or through participations, are skipped. Requires admin authentication.
// @Tags People
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param data body model.PeopleImportRequest true "Names"
// @Success 200 {object} model.PeopleImportResponse "Counts"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/people/import [post]
func (s *Server) handleImportPeople(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	var req model.PeopleImportRequest
	if !s.parseJSON(w, r, &req) {
		return
	}
	names, result := validation.ValidatePeopleNames(req.Names, util.PersonKey)
	if result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}
	created, skipped, err := s.db.ImportPeople(plan.ID, names)
	if err != nil {
		s.peopleError(w, err, "Failed to import people")
		return
	}
	if created > 0 {
		s.notifyPlanChanged(plan.ID, scopeParticipants)
	}
	s.respondJSON(w, model.PeopleImportResponse{Created: created, Skipped: skipped})
}

// handleRenamePerson renames or merges a person
// @Summary Rename person
// @Description Renames a person in all participations of the plan and in the stored data. If the target name belongs to another person and merge is false, 409 is returned; with merge the data are combined (the target wins, empty values are filled in) and duplicate participations in one shift become one (assigned beats signed_up beats declined). Requires admin authentication.
// @Tags People
// @Accept json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param data body model.PersonRenameRequest true "Old and new name"
// @Success 200 {string} string "Renamed"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or person not found"
// @Failure 409 {string} string "Target name is taken"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/people/rename [post]
func (s *Server) handleRenamePerson(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	var req model.PersonRenameRequest
	if !s.parseJSON(w, r, &req) {
		return
	}
	result := validation.NewResult()
	to, nr := validation.ValidateParticipantName(req.To)
	result.MergeWithPrefix(nr, "to.")
	from, nr := validation.ValidateParticipantName(req.From)
	result.MergeWithPrefix(nr, "from.")
	if result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}
	if err := s.db.RenamePerson(plan.ID, from, to, req.Merge); err != nil {
		s.peopleError(w, err, "Failed to rename person")
		return
	}
	s.notifyPlanChanged(plan.ID, scopeParticipants)
	s.respondOK(w)
}

// handleDeletePerson deletes the stored data of a person
// @Summary Delete person data
// @Description Deletes only the stored person data (attendance, target, person fields), never participations. The key is the normalised name (or any spelling of it). Requires admin authentication.
// @Tags People
// @Param token path string true "Plan token"
// @Param key path string true "Person name"
// @Param X-Admin-Token header string true "Admin token"
// @Success 204 "Deleted"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or person not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/people/{key} [delete]
func (s *Server) handleDeletePerson(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	if err := s.db.DeletePerson(plan.ID, mux.Vars(r)["key"]); err != nil {
		s.peopleError(w, err, "Failed to delete person")
		return
	}
	s.notifyPlanChanged(plan.ID, scopeParticipants)
	s.respondNoContent(w)
}

// handleLookupPerson returns what a person entered themselves
// @Summary Look up person details
// @Description Returns the details a person entered themselves (attendance and fields editable by participants), found by name. Anyone who knows a name can read these values; this matches the plan's trust model without login. Values of admin fields are never returned.
// @Tags People
// @Produce json
// @Param token path string true "Plan token"
// @Param name query string true "Person name"
// @Success 200 {object} model.PersonLookupResponse "Person details"
// @Failure 400 {string} string "Invalid input"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Router /plans/{token}/people/lookup [get]
func (s *Server) handleLookupPerson(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	name, result := validation.ValidateParticipantName(r.URL.Query().Get("name"))
	if result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}
	person, err := s.db.GetPersonByName(plan.ID, name)
	if err != nil {
		s.peopleError(w, err, "Failed to look up person")
		return
	}
	response := model.PersonLookupResponse{Values: map[string]any{}}
	if person != nil {
		response.Exists = true
		if plan.AttendanceMode == "participant" {
			response.AvailableFrom = person.AvailableFrom
			response.AvailableUntil = person.AvailableUntil
		}
		for _, f := range plan.Fields {
			if f.EditableBy != db.EditableParticipant {
				continue
			}
			key := itoa64(f.ID)
			if v, ok := person.Values[key]; ok {
				response.Values[key] = v
			}
		}
	}
	s.respondJSON(w, response)
}
