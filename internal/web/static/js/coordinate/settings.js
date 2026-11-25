/**
 * Fields & rules tab: person fields, experience field, attendance and working time rules.
 *
 * The settings are saved with PUT /api/plans/{token}, which needs all plan fields, so the form
 * sends the plan's existing values together with the changed coordination settings. The nested
 * components get the page scope as `page` ($data) and do not use the parent through `this`.
 */

/** "1:30" or "1,5" (hours) → minutes; null for empty; NaN for something else. */
function parseHours(text) {
    const s = String(text ?? '').trim();
    if (s === '') return null;
    const clock = /^(\d{1,3}):([0-5]\d)$/.exec(s);
    if (clock) return Number(clock[1]) * 60 + Number(clock[2]);
    const n = Number(s.replace(',', '.'));
    return Number.isFinite(n) && n >= 0 ? Math.round(n * 60) : NaN;
}

document.addEventListener('alpine:init', () => {
    Alpine.data('coordSettings', () => ({
        form: { experienceFieldId: '', attendanceMode: 'off', target: '', rest: '', maxDay: '' },
        dirty: false,
        errors: {},

        /** Copies the plan's settings into the form (the page calls it when the plan is loaded). */
        sync(plan, page) {
            const hours = m => (m == null ? '' : m % 60 === 0 ? String(m / 60) : page.duration(m));
            this.form = {
                experienceFieldId: plan.experience_field_id != null ? String(plan.experience_field_id) : '',
                attendanceMode: plan.attendance_mode || 'off',
                target: hours(plan.default_target_minutes),
                rest: plan.min_rest_minutes == null ? '' : page.duration(plan.min_rest_minutes),
                maxDay: hours(plan.max_day_minutes),
            };
            this.dirty = false;
            this.errors = {};
        },

        choiceFields(plan) {
            return plan.fields.filter(f => f.kind === 'choice');
        },

        levelsText(page) {
            return page.$t('coord.levels_order', { levels: page.model.experience.options.map(o => page.t(o)).join(' < ') });
        },

        kindLabel(page, f) {
            return page.$t(`coord.field_kind_${f.kind}`);
        },

        editField(id) {
            this.$dispatch('coord-field', { id });
        },

        /** Moves a field one position up or down and saves the order. */
        async move(page, id, delta) {
            const ids = page.plan.fields.map(f => f.id);
            const i = ids.indexOf(id);
            const j = i + delta;
            if (i < 0 || j < 0 || j >= ids.length || page.busy) return;
            [ids[i], ids[j]] = [ids[j], ids[i]];
            page.busy = true;
            page.clearErrors();
            try {
                await page.api('PUT', '/fields/order', { field_ids: ids });
                await page.load(true);
            } catch (e) {
                page.fail(e, false);
            } finally {
                page.busy = false;
            }
        },

        /** Saves the coordination settings together with the unchanged plan data. */
        async save(page) {
            if (page.busy) return;
            const target = parseHours(this.form.target);
            const rest = parseHours(this.form.rest);
            const maxDay = parseHours(this.form.maxDay);
            this.errors = {};
            if (Number.isNaN(target)) this.errors.target = true;
            if (Number.isNaN(rest) || rest > 1440) this.errors.rest = true;
            if (Number.isNaN(maxDay) || maxDay > 1440) this.errors.maxDay = true;
            if (Object.keys(this.errors).length) return;

            const p = page.plan;
            const body = {
                title_de: p.title_de,
                title_en: p.title_en,
                description_de: p.description_de,
                description_en: p.description_en,
                start_date: p.start_date,
                end_date: p.end_date,
                is_multilingual: p.is_multilingual,
                day_boundary_time: p.day_boundary_time,
                collaborative_mode: p.collaborative_mode,
                show_declined: p.show_declined,
                experience_field_id: this.form.experienceFieldId ? Number(this.form.experienceFieldId) : null,
                attendance_mode: this.form.attendanceMode,
                default_target_minutes: target,
                min_rest_minutes: rest,
                max_day_minutes: maxDay,
            };
            page.busy = true;
            page.clearErrors();
            try {
                await page.api('PUT', '', body);
                this.dirty = false;
                page.showToast(page.$t('coord.saved'), false);
                await page.load(true);
            } catch (e) {
                page.fail(e, false);
            } finally {
                page.busy = false;
            }
        },
    }));

    /** Areas table: factor, popularity and experience need per shift type, saved per row. */
    Alpine.data('coordAreas', () => ({
        rows: {},

        /** Copies the types into the rows; rows with unsaved input keep their values. */
        sync(types, page) {
            const rows = {};
            for (const t of types) {
                const old = this.rows[t.id];
                rows[t.id] = old && (old.dirty || old.saving) ? old : {
                    load: String(t.load_factor ?? 1).replace('.', ','),
                    popularity: t.popularity || 'auto',
                    count: t.exp_min_count || 0,
                    level: t.exp_min_level || '',
                    dirty: false,
                    saving: false,
                    error: '',
                };
            }
            this.rows = rows;
        },

        /** Saves one area with its full type record (title, description and location stay as they are). */
        async save(page, type) {
            const row = this.rows[type.id];
            if (!row || row.saving) return;
            const load = Number(String(row.load).trim().replace(',', '.'));
            if (!Number.isFinite(load) || load < 0.1 || load > 5) {
                row.error = page.$t('coord.error_load_factor');
                return;
            }
            row.saving = true;
            row.error = '';
            try {
                await page.api('PUT', `/types/${type.id}`, {
                    title_de: type.title_de,
                    title_en: type.title_en,
                    description_de: type.description_de,
                    description_en: type.description_en,
                    location_de: type.location_de,
                    location_en: type.location_en,
                    load_factor: load,
                    popularity: row.popularity,
                    exp_min_count: Number(row.count) || 0,
                    exp_min_level: row.level || '',
                });
                row.dirty = false;
                await page.load(true);
            } catch (e) {
                row.error = e.message || page.$t('coord.error_action');
            } finally {
                row.saving = false;
            }
        },
    }));

    Alpine.data('coordFieldModal', () => ({
        id: null,
        form: { kind: 'choice', label_de: '', label_en: '', options: [], editable_by: 'participant', required: false },
        confirmDelete: false,

        open(detail, page, dialog) {
            page.clearErrors();
            this.confirmDelete = false;
            const f = detail.id ? page.plan.fields.find(x => x.id === detail.id) : null;
            this.id = f ? f.id : null;
            this.form = f
                ? {
                    kind: f.kind,
                    label_de: f.label_de || '',
                    label_en: f.label_en,
                    options: f.options.map(o => ({ ...o })),
                    editable_by: f.editable_by,
                    required: f.required,
                }
                : {
                    kind: 'choice',
                    label_de: '',
                    label_en: '',
                    options: [{ de: '', en: '' }, { de: '', en: '' }],
                    editable_by: 'participant',
                    required: false,
                };
            dialog.showModal();
        },

        close(page, dialog) {
            page.clearErrors();
            dialog.close();
        },

        addOption() {
            if (this.form.options.length < 10) this.form.options.push({ de: '', en: '' });
        },

        removeOption(i) {
            if (this.form.options.length > 2) this.form.options.splice(i, 1);
        },

        moveOption(i, delta) {
            const j = i + delta;
            if (j < 0 || j >= this.form.options.length) return;
            const o = this.form.options;
            [o[i], o[j]] = [o[j], o[i]];
        },

        async save(page, dialog) {
            if (page.busy) return;
            const f = this.form;
            const body = {
                kind: f.kind,
                label_en: f.label_en.trim(),
                label_de: page.plan.is_multilingual ? f.label_de.trim() : (f.label_de.trim() || null),
                options: f.kind === 'choice' ? f.options.map(o => ({ ...(o.id ? { id: o.id } : {}), de: o.de.trim(), en: o.en.trim() })) : [],
                editable_by: f.editable_by,
                required: f.editable_by === 'participant' && f.required,
            };
            page.busy = true;
            page.clearErrors();
            try {
                if (this.id) await page.api('PUT', `/fields/${this.id}`, body);
                else await page.api('POST', '/fields', body);
                dialog.close();
                await page.load(true);
            } catch (e) {
                page.fail(e, true);
            } finally {
                page.busy = false;
            }
        },

        async remove(page, dialog) {
            if (page.busy || !this.id) return;
            page.busy = true;
            page.clearErrors();
            try {
                await page.api('DELETE', `/fields/${this.id}`);
                dialog.close();
                await page.load(true);
            } catch (e) {
                page.fail(e, true);
            } finally {
                page.busy = false;
                this.confirmDelete = false;
            }
        },
    }));
});
