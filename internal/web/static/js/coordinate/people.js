/**
 * People tab: list, person dialog and name import (prj/coordination-plan.md, 9.3).
 *
 * The list only filters and formats what the model provides. The dialogs call the people API
 * through the page ($data of the page scope is passed in as `page`), so the nested components do
 * not rely on `this` reaching the parent scope.
 */
import { toInputValue, fromInputValue, serializeDateTime } from '../datetime.js';

/** Lower case, no diacritics: "Ánna" is found with "anna". */
function searchKey(str) {
    return String(str || '').normalize('NFD').replace(/\p{M}/gu, '').normalize('NFC').toLocaleLowerCase().trim();
}

document.addEventListener('alpine:init', () => {
    Alpine.data('coordPeople', () => ({
        search: '',
        onlyWarnings: false,

        /** People matching the search and the warnings filter, sorted by name. */
        rows(personList) {
            const q = searchKey(this.search);
            return personList
                .filter(p => (!q || searchKey(p.name).includes(q)) && (!this.onlyWarnings || p.warnings.length > 0))
                .sort((a, b) => a.name.localeCompare(b.name));
        },

        edit(key) {
            this.$dispatch('coord-person', { key });
        },

        add() {
            this.$dispatch('coord-person', { key: null });
        },

        paste() {
            this.$dispatch('coord-paste');
        },

        /** Experience level label of a person ("" if none). */
        levelLabel(page, p) {
            const o = page.model.experience.options[p.level];
            return o ? page.t(o) : '';
        },

        /** "{Field} fehlt" for the first missing field, "+n" for further ones. */
        missingText(page, p) {
            const [first] = p.missingFields;
            if (!first) return '';
            const text = page.$t('coord.field_missing', { field: page.fieldLabel(first) });
            return p.missingFields.length > 1 ? `${text} +${p.missingFields.length - 1}` : text;
        },

        /** Attendance as "Fr 18.07. 16:00 bis So 14:00" with open ends left out. */
        attendanceLabel(page, p) {
            const { from, until } = p.attendance;
            const fmt = d => `${page.dayLabel(d)} ${page.time(d)}`;
            if (from && until) return page.$t('coord.attendance_range', { from: fmt(from), until: fmt(until) });
            if (from) return page.$t('coord.attendance_from', { from: fmt(from) });
            if (until) return page.$t('coord.attendance_until', { until: fmt(until) });
            return '';
        },

        /** Target in hours; the plan default is marked with an asterisk. */
        targetLabel(page, p) {
            if (p.targetMin === null) return '';
            return page.number(p.targetMin / 60) + (p.row?.target_minutes == null ? '*' : '');
        },

        /** Shown value of a custom field for a person. */
        fieldValue(page, p, f) {
            const v = p.row?.values?.[f.id];
            if (v === undefined || v === null || v === '') return '';
            if (f.kind === 'choice') {
                const o = f.options.find(x => x.id === v);
                return o ? page.t(o) : '';
            }
            if (f.kind === 'boolean') return v ? page.$t('coord.yes') : '';
            if (f.kind === 'number') return page.number(v);
            return String(v);
        },
    }));

    Alpine.data('coordPersonModal', () => ({
        isNew: true,
        originalName: '',
        hasRow: false,
        form: { name: '', from: '', until: '', targetHours: '', prefersLongDays: false, values: {} },
        mergeAsk: false,
        confirmDelete: false,

        /** Opens the dialog for a person key, or empty for a new person. */
        open(detail, page, dialog) {
            page.clearErrors();
            this.mergeAsk = false;
            this.confirmDelete = false;
            const p = detail.key ? page.model.persons.get(detail.key) : null;
            this.isNew = !p;
            this.originalName = p ? p.name : '';
            this.hasRow = !!p?.row;
            this.form = {
                name: p ? p.name : '',
                from: p?.attendance.from ? toInputValue(p.attendance.from) : '',
                until: p?.attendance.until ? toInputValue(p.attendance.until) : '',
                targetHours: p?.row?.target_minutes != null ? String(p.row.target_minutes / 60) : '',
                prefersLongDays: !!p?.prefersLongDays,
                values: { ...(p?.row?.values || {}) },
            };
            dialog.showModal();
        },

        close(page, dialog) {
            page.clearErrors();
            dialog.close();
        },

        /** Request body of the person upsert. */
        payload(plan) {
            const f = this.form;
            const values = {};
            for (const field of plan.fields) {
                const v = f.values[field.id];
                if (v === undefined || v === null || v === '' || v === false) continue;
                values[field.id] = field.kind === 'number' ? Number(v) : v;
            }
            const hours = parseFloat(String(f.targetHours).replace(',', '.'));
            return {
                name: f.name.trim(),
                available_from: f.from ? serializeDateTime(fromInputValue(f.from)) : null,
                available_until: f.until ? serializeDateTime(fromInputValue(f.until)) : null,
                target_minutes: Number.isFinite(hours) ? Math.round(hours * 60) : null,
                prefers_long_days: f.prefersLongDays,
                values,
            };
        },

        /**
         * Saves the person. A changed name renames all participations first; if the new name belongs
         * to somebody else the server answers 409 and the dialog asks whether to merge.
         */
        async save(page, dialog, merge = false) {
            if (page.busy) return;
            const body = this.payload(page.plan);
            if (!body.name) return;
            page.busy = true;
            page.clearErrors();
            try {
                if (!this.isNew && body.name !== this.originalName) {
                    await page.api('POST', '/people/rename', { from: this.originalName, to: body.name, merge });
                }
                await page.api('PUT', '/people', body);
                dialog.close();
                await page.load(true);
            } catch (e) {
                if (e.status === 409 && !merge) this.mergeAsk = true;
                else page.fail(e, true);
            } finally {
                page.busy = false;
            }
        },

        /** Deletes the stored details of the person (participations stay). */
        async remove(page, dialog) {
            if (page.busy) return;
            page.busy = true;
            page.clearErrors();
            try {
                await page.api('DELETE', `/people/${encodeURIComponent(this.originalName)}`);
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

    Alpine.data('coordPasteModal', () => ({
        text: '',
        result: null,

        open(dialog) {
            this.text = '';
            this.result = null;
            dialog.showModal();
        },

        names() {
            return this.text.split('\n').map(s => s.trim()).filter(Boolean);
        },

        async submit(page) {
            if (page.busy || !this.names().length) return;
            page.busy = true;
            page.clearErrors();
            try {
                this.result = await page.api('POST', '/people/import', { names: this.names() });
                await page.load(true);
            } catch (e) {
                page.fail(e, true);
            } finally {
                page.busy = false;
            }
        },
    }));
});
