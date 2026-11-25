-- Coordination can be switched on per plan (prj/coordination-refinement-plan.md, section 5).
-- New plans start without it; plans that already hold coordination data keep it switched on.
ALTER TABLE shift_plans ADD COLUMN coordination_enabled BOOLEAN NOT NULL DEFAULT false;

UPDATE shift_plans SET coordination_enabled = true
WHERE experience_field_id IS NOT NULL
   OR attendance_mode <> 'off'
   OR default_target_minutes IS NOT NULL
   OR min_rest_minutes IS NOT NULL
   OR max_day_minutes IS NOT NULL
   OR id IN (SELECT plan_id FROM plan_fields)
   OR id IN (SELECT plan_id FROM plan_people)
   OR id IN (SELECT plan_id FROM shift_types
             WHERE load_factor <> 1.0 OR popularity <> 'auto' OR exp_min_count > 0)
   OR id IN (SELECT st.plan_id FROM shifts s JOIN shift_types st ON st.id = s.type_id
             WHERE s.load_factor IS NOT NULL OR s.popularity IS NOT NULL
                OR s.exp_min_count IS NOT NULL OR s.exp_min_level IS NOT NULL)
   OR id IN (SELECT st.plan_id FROM participants p JOIN shifts s ON s.id = p.shift_id
             JOIN shift_types st ON st.id = s.type_id
             WHERE p.status <> 'signed_up' OR p.added_by_admin);
