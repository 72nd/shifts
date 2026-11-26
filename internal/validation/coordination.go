package validation

import (
	"fmt"
	"math"
	"strconv"

	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/model"
)

// Limits of the coordination tools.
const (
	MaxFieldTextLength   = 200
	MaxApplyOps          = 2000
	MaxImportNames       = 500
	MaxMinutesPerDay     = 1440
	MaxTargetMinutes     = 100000
	MinLoadFactor        = 0.1
	MaxLoadFactor        = 5.0
	MaxExpMinCount       = 1000
	MinChoiceOptions     = 2
	MaxChoiceOptions     = 10
	MaxOptionLabelLength = 100
)

// Popularity values of shift types and shifts.
var popularityValues = map[string]bool{"auto": true, "popular": true, "very_popular": true, "normal": true}

// ValidPopularity reports whether p is auto, popular, very_popular or normal.
func ValidPopularity(p string) bool { return popularityValues[p] }

// ValidateLoadFactor checks 0.1 to 5.0 with at most one decimal place.
func ValidateLoadFactor(f float64, field string, r *Result) {
	if f < MinLoadFactor || f > MaxLoadFactor || math.Abs(f*10-math.Round(f*10)) > 1e-9 {
		r.AddInvalid(field, "Load factor must be between 0.1 and 5.0 with at most one decimal place")
	}
}

// FindField returns the field with the given ID, or nil.
func FindField(fields []db.PlanField, id int64) *db.PlanField {
	for i := range fields {
		if fields[i].ID == id {
			return &fields[i]
		}
	}
	return nil
}

// hasOption reports whether the choice field has an option with the given ID.
func hasOption(f *db.PlanField, id string) bool {
	for _, o := range f.Options {
		if o.ID == id {
			return true
		}
	}
	return false
}

// ValidateExperienceNeed checks exp_min_count and exp_min_level against the plan's
// experience field. A count above 0 needs a level that is an option of that field.
// required is the number of places of the shift (-1 or 0 = no upper bound check besides the maximum).
func ValidateExperienceNeed(count *int, level *string, required int, plan *db.ShiftPlan, r *Result) {
	if count != nil {
		max := MaxExpMinCount
		if required > 0 {
			max = required
		}
		if *count < 0 || *count > max {
			r.AddInvalid("exp_min_count", "Experience need must be between 0 and the number of required participants")
		}
	}
	if level != nil && *level != "" {
		var field *db.PlanField
		if plan.ExperienceFieldID != nil {
			field = FindField(plan.Fields, *plan.ExperienceFieldID)
		}
		if field == nil || !hasOption(field, *level) {
			r.AddInvalid("exp_min_level", "Experience level must be an option of the plan's experience field")
		}
	}
}

// ValidateShiftOverrides checks the coordination overrides of a shift.
func ValidateShiftOverrides(loadFactor *float64, popularity *string, count *int, level *string, required int, plan *db.ShiftPlan, r *Result) {
	if loadFactor != nil {
		ValidateLoadFactor(*loadFactor, "load_factor", r)
	}
	if popularity != nil && !ValidPopularity(*popularity) {
		r.AddInvalid("popularity", "Popularity must be auto, popular, very_popular or normal")
	}
	ValidateExperienceNeed(count, level, required, plan, r)
}

// ValidateCoordinationSettings checks the coordination settings of a plan update.
func ValidateCoordinationSettings(req *model.ShiftPlanUpdateRequest, fields []db.PlanField) *Result {
	r := NewResult()
	switch req.AttendanceMode {
	case "off", "participant", "admin":
	default:
		r.AddInvalid("attendance_mode", "Attendance mode must be off, participant or admin")
	}
	if req.ExperienceFieldID != nil {
		f := FindField(fields, *req.ExperienceFieldID)
		if f == nil || f.Kind != db.FieldChoice {
			r.AddInvalid("experience_field_id", "Experience field must be a choice field of this plan")
		}
	}
	checkMinutes := func(v *int, field string, max int) {
		if v != nil && (*v < 0 || *v > max) {
			r.AddInvalid(field, field+" must be between 0 and "+strconv.Itoa(max))
		}
	}
	checkMinutes(req.DefaultTargetMinutes, "default_target_minutes", MaxTargetMinutes)
	checkMinutes(req.MinRestMinutes, "min_rest_minutes", MaxMinutesPerDay)
	checkMinutes(req.MaxDayMinutes, "max_day_minutes", MaxMinutesPerDay)
	return r
}

// ValidatePlanField validates and sanitizes a field request in place. On update the
// caller sets f.Kind to the stored kind first (the kind cannot change).
func ValidatePlanField(f *model.PlanFieldRequest, isMultilingual, create bool) *Result {
	r := NewResult()
	f.LabelEN = ValidateRequiredMaxLength(f.LabelEN, "label_en", "English label", MaxTitleLength, r)
	if isMultilingual {
		de := ""
		if f.LabelDE != nil {
			de = *f.LabelDE
		}
		de = ValidateRequiredMaxLength(de, "label_de", "German label", MaxTitleLength, r)
		f.LabelDE = &de
	} else if f.LabelDE != nil {
		de := ValidateMaxLength(*f.LabelDE, "label_de", "German label", MaxTitleLength, r)
		f.LabelDE = &de
	}
	if create {
		switch f.Kind {
		case db.FieldChoice, db.FieldNumber, db.FieldBoolean, db.FieldText:
		default:
			r.AddInvalid("kind", "Kind must be choice, number, boolean or text")
		}
	}
	if f.EditableBy != db.EditableParticipant && f.EditableBy != db.EditableAdmin {
		r.AddInvalid("editable_by", "editable_by must be participant or admin")
	}
	if f.EditableBy == db.EditableAdmin {
		f.Required = false
	}
	if f.Kind == db.FieldChoice {
		if len(f.Options) < MinChoiceOptions || len(f.Options) > MaxChoiceOptions {
			r.AddInvalid("options", "A choice field needs between 2 and 10 options")
		}
		for i := range f.Options {
			prefix := "options[" + itoa(i) + "]."
			f.Options[i].EN = ValidateRequiredMaxLength(f.Options[i].EN, prefix+"en", "English option", MaxOptionLabelLength, r)
			if isMultilingual {
				f.Options[i].DE = ValidateRequiredMaxLength(f.Options[i].DE, prefix+"de", "German option", MaxOptionLabelLength, r)
			} else {
				f.Options[i].DE = ValidateMaxLength(f.Options[i].DE, prefix+"de", "German option", MaxOptionLabelLength, r)
			}
		}
	} else {
		f.Options = nil
	}
	return r
}

// ValidateFieldValues validates values by field ID against the plan's fields and returns
// the cleaned map. Only fields editable by editor ("participant" or "admin"; admin may edit all)
// are accepted. A nil value clears a field when allowNil is set.
func ValidateFieldValues(fields []db.PlanField, values map[string]any, editor string, allowNil bool, r *Result) map[string]any {
	out := map[string]any{}
	for key, v := range values {
		field := "values." + key
		id, err := strconv.ParseInt(key, 10, 64)
		var f *db.PlanField
		if err == nil {
			f = FindField(fields, id)
		}
		if f == nil {
			r.AddInvalid(field, "Unknown field")
			continue
		}
		if editor == db.EditableParticipant && f.EditableBy != db.EditableParticipant {
			r.AddInvalid(field, "This field can only be filled in by the coordination")
			continue
		}
		if v == nil {
			if allowNil {
				out[key] = nil
			}
			continue
		}
		switch f.Kind {
		case db.FieldChoice:
			s, ok := v.(string)
			if !ok || !hasOption(f, s) {
				r.AddInvalid(field, "Value must be one of the options")
				continue
			}
			out[key] = s
		case db.FieldNumber:
			n, ok := v.(float64)
			if !ok || math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 1e6 || math.Abs(n*10-math.Round(n*10)) > 1e-9 {
				r.AddInvalid(field, "Value must be a number with at most one decimal place")
				continue
			}
			out[key] = n
		case db.FieldBoolean:
			b, ok := v.(bool)
			if !ok {
				r.AddInvalid(field, "Value must be true or false")
				continue
			}
			out[key] = b
		case db.FieldText:
			s, ok := v.(string)
			if !ok {
				r.AddInvalid(field, "Value must be text")
				continue
			}
			s = ValidateMaxLength(s, field, "Text", MaxFieldTextLength, r)
			if s == "" {
				if allowNil {
					out[key] = nil
				}
				continue
			}
			out[key] = s
		}
	}
	return out
}

// CheckRequiredFields adds an error for every required participant field that has no value
// in the (merged) values.
func CheckRequiredFields(fields []db.PlanField, merged map[string]any, r *Result) {
	for _, f := range fields {
		if f.Required && f.EditableBy == db.EditableParticipant {
			if v, ok := merged[strconv.FormatInt(f.ID, 10)]; !ok || v == nil {
				r.AddRequired("values."+strconv.FormatInt(f.ID, 10), "Field "+f.LabelEN)
			}
		}
	}
}

// ValidateAttendance checks that from is not after until.
func ValidateAttendance(from, until *model.DateTime, r *Result) {
	if from != nil && until != nil && until.Before(from.Time) {
		r.AddInvalid("available_until", "Attendance end must not be before its start")
	}
}

// ValidatePersonWrite validates an admin person upsert in place.
func ValidatePersonWrite(req *model.PersonWriteRequest, fields []db.PlanField) *Result {
	r := NewResult()
	var name string
	name, nr := ValidateParticipantName(req.Name)
	r.Merge(nr)
	req.Name = name
	ValidateAttendance(req.AvailableFrom, req.AvailableUntil, r)
	if req.TargetMinutes != nil && (*req.TargetMinutes < 0 || *req.TargetMinutes > MaxTargetMinutes) {
		r.AddInvalid("target_minutes", "Target must be between 0 and "+strconv.Itoa(MaxTargetMinutes)+" minutes")
	}
	req.Values = ValidateFieldValues(fields, req.Values, db.EditableAdmin, false, r)
	return r
}

// ValidatePeopleNames sanitizes and de-duplicates the names of an import.
func ValidatePeopleNames(names []string, key func(string) string) ([]string, *Result) {
	r := NewResult()
	if len(names) == 0 {
		r.AddRequired("names", "Names")
		return nil, r
	}
	if len(names) > MaxImportNames {
		r.AddInvalid("names", fmt.Sprintf("At most %d names can be imported at once", MaxImportNames))
		return nil, r
	}
	seen := map[string]bool{}
	var out []string
	for i, n := range names {
		name, nr := ValidateParticipantName(n)
		if nr.HasErrors() {
			r.MergeWithPrefix(nr, "names["+itoa(i)+"].")
			continue
		}
		if k := key(name); !seen[k] {
			seen[k] = true
			out = append(out, name)
		}
	}
	return out, r
}

// ValidateCoordinationApply validates and sanitizes an apply request in place.
func ValidateCoordinationApply(req *model.CoordinationApplyRequest) *Result {
	r := NewResult()
	if len(req.Ops) > MaxApplyOps {
		r.AddInvalid("ops", fmt.Sprintf("At most %d operations can be applied at once", MaxApplyOps))
		return r
	}
	if len(req.Ops)+len(req.Lock) == 0 {
		r.AddRequired("ops", "Operations")
		return r
	}
	validateCoordinationOps(req.Ops, r)
	for i, id := range req.Lock {
		if id <= 0 {
			r.Add("lock["+itoa(i)+"]", CodeInvalidValue, "Shift ID must be a positive number")
		}
	}
	return r
}

// validateCoordinationOps checks and sanitizes the operations in place (shared by apply and drafts).
func validateCoordinationOps(ops []model.CoordinationOp, r *Result) {
	for i := range ops {
		op := &ops[i]
		prefix := "ops[" + itoa(i) + "]."
		if op.ShiftID <= 0 {
			r.Add(prefix+"shift_id", CodeInvalidValue, "Shift ID must be a positive number")
		}
		name, nr := ValidateParticipantName(op.Name)
		r.MergeWithPrefix(nr, prefix)
		op.Name = name
		switch op.Status {
		case db.StatusSignedUp, db.StatusAssigned, db.StatusDeclined, db.StatusRemove:
		default:
			r.Add(prefix+"status", CodeInvalidValue, "Status must be signed_up, assigned, declined or remove")
		}
	}
}
