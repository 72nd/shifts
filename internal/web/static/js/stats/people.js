/**
 * People tab of the statistics page (prj/stats-plan.md, 6.3).
 *
 * Only searching, filtering, sorting and CSV export live here. All numbers
 * come from model.people (compute.js). The component is nested inside
 * x-data="shiftPlanStats"; the template passes parent data (model.people,
 * token, ...) as arguments, so no method relies on parent scope via `this`.
 */

// Sort keys in column order; `desc` is the direction used on first click.
const SORT_KEYS = {
    name: { desc: false },
    shifts: { desc: true },
    minutes: { desc: true },
    doneMinutes: { desc: true },
    days: { desc: true },
    typeCount: { desc: true },
    nightMinutes: { desc: true },
    maxDayMinutes: { desc: true },
    warnings: { desc: true },
};

const WARNING_ORDER = ['overlap', 'rest', 'dayload'];

/** Lower case, NFC, diacritics removed: search should find "Ánna" with "anna". */
function searchKey(str) {
    return String(str || '').normalize('NFD').replace(/\p{M}/gu, '').normalize('NFC').toLocaleLowerCase().trim();
}

/** Warning weight for sorting: overlaps count more than rest/day-load hints. */
function warningScore(person) {
    return person.warnings.reduce((sum, w) => sum + (w.kind === 'overlap' ? 100 : 1), 0);
}

function sortValue(person, key) {
    return key === 'warnings' ? warningScore(person) : person[key];
}

/** RFC 4180 quoting (as encoding/csv) plus protection against spreadsheet formulas. */
function csvCell(value) {
    let s = String(value ?? '');
    if (/^[=+\-@\t\r]/.test(s)) s = `'${s}`;
    return /[",\r\n]/.test(s) || s !== s.trim() ? `"${s.replace(/"/g, '""')}"` : s;
}

/** Minutes → hours with decimal point, independent of the UI language. */
function csvHours(minutes) {
    return String(Math.round(((minutes || 0) / 60) * 100) / 100);
}

document.addEventListener('alpine:init', () => {
    Alpine.data('statsPeople', () => ({
        search: '',
        quick: 'all', // all | warn | single
        sortKey: 'name',
        sortDesc: false,
        open: {},

        sortKeys: Object.keys(SORT_KEYS),

        // Table columns (desktop order). `wide` columns only show from md up;
        // `num` columns are right aligned. Labels are stats.col_* keys.
        cols: [
            { key: 'name', label: 'col_name', wide: false, num: false },
            { key: 'shifts', label: 'col_shifts', wide: false, num: true },
            { key: 'minutes', label: 'col_hours', wide: false, num: true },
            { key: 'doneMinutes', label: 'col_done', wide: true, num: true },
            { key: 'days', label: 'col_days', wide: true, num: true },
            { key: 'typeCount', label: 'col_types', wide: true, num: true },
            { key: 'nightMinutes', label: 'col_night', wide: true, num: true },
            { key: 'maxDayMinutes', label: 'col_max_day', wide: true, num: true },
            { key: 'warnings', label: 'col_warning', wide: false, num: false },
        ],

        // --- Counts and rows (parent data is passed in) ---

        warnCount(people) {
            return (people || []).filter(p => p.warnings.length).length;
        },

        singleCount(people) {
            return (people || []).filter(p => p.shifts === 1).length;
        },

        /** Filtered and sorted rows. */
        rows(people) {
            const q = searchKey(this.search);
            const list = (people || []).filter(p => {
                if (this.quick === 'warn' && !p.warnings.length) return false;
                if (this.quick === 'single' && p.shifts !== 1) return false;
                return !q || searchKey(p.name).includes(q);
            });
            const dir = this.sortDesc ? -1 : 1;
            const key = this.sortKey;
            return list.sort((a, b) => {
                const va = sortValue(a, key);
                const vb = sortValue(b, key);
                const diff = key === 'name' ? String(va).localeCompare(String(vb)) : va - vb;
                return dir * diff || a.name.localeCompare(b.name);
            });
        },

        get localFilterActive() {
            return this.search.trim() !== '' || this.quick !== 'all';
        },

        resetLocal() {
            this.search = '';
            this.quick = 'all';
        },

        // --- Sorting ---

        sortBy(key) {
            if (!SORT_KEYS[key]) return;
            if (this.sortKey === key) {
                this.sortDesc = !this.sortDesc;
            } else {
                this.sortKey = key;
                this.sortDesc = SORT_KEYS[key].desc;
            }
        },

        /** Value for select on mobile; changing it applies the key's default direction. */
        setSortKey(key) {
            if (!SORT_KEYS[key]) return;
            this.sortKey = key;
            this.sortDesc = SORT_KEYS[key].desc;
        },

        ariaSort(key) {
            if (this.sortKey !== key) return 'none';
            return this.sortDesc ? 'descending' : 'ascending';
        },

        // --- Expanding ---

        isOpen(key) {
            return !!this.open[key];
        },

        toggle(key) {
            this.open = { ...this.open, [key]: !this.open[key] };
        },

        // --- Warnings ---

        /** Distinct warning kinds of a person in fixed order. */
        kinds(person) {
            const set = new Set(person.warnings.map(w => w.kind));
            return WARNING_ORDER.filter(k => set.has(k));
        },

        hasError(person) {
            return person.warnings.some(w => w.kind === 'overlap');
        },

        // --- Duplicate hints ---

        /** [{a, b}] display names for model.possibleDuplicates. */
        duplicates(model) {
            if (!model) return [];
            const names = new Map(model.people.map(p => [p.key, p.name]));
            return (model.possibleDuplicates || [])
                .filter(([a, b]) => names.has(a) && names.has(b))
                .map(([a, b]) => ({ a: names.get(a), b: names.get(b) }));
        },

        // --- Mini timeline ---

        /**
         * One column per plan day; segments are the person's shifts that day,
         * height relative to the person's busiest day. Grouping for display only.
         */
        timeline(person, days) {
            const max = Math.max(1, person.maxDayMinutes);
            const byDay = new Map();
            for (const it of person.items) {
                if (!byDay.has(it.day)) byDay.set(it.day, []);
                byDay.get(it.day).push(it);
            }
            return (days || []).map(day => ({
                day,
                segments: (byDay.get(day) || []).map(it => ({
                    id: it.shiftId,
                    status: it.status,
                    style: `height:${Math.max(4, Math.round((it.minutes / max) * 40))}px`,
                })),
            }));
        },

        // --- CSV (admin only, client side) ---

        /**
         * Downloads the given rows as CSV: comma separated, UTF-8 with BOM,
         * quoting like internal/export/csv_exporter.go (encoding/csv).
         * @param {Array} rows people as shown in the table
         * @param {string[]} headers localised column titles (9)
         * @param {Object} kindLabels localised warning kinds
         * @param {string} name file name without extension
         */
        downloadCsv(rows, headers, kindLabels, name) {
            const lines = [headers.map(csvCell).join(',')];
            for (const p of rows) {
                lines.push([
                    p.name,
                    p.shifts,
                    csvHours(p.minutes),
                    csvHours(p.doneMinutes),
                    p.days,
                    p.typeCount,
                    csvHours(p.nightMinutes),
                    csvHours(p.maxDayMinutes),
                    this.kinds(p).map(k => kindLabels[k] || k).join(';'),
                ].map(csvCell).join(','));
            }
            const blob = new Blob(['﻿', lines.join('\r\n'), '\r\n'], { type: 'text/csv;charset=utf-8' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `${String(name || 'plan').replace(/[^\p{L}\p{N}_-]+/gu, '-')}-people.csv`;
            document.body.appendChild(a);
            a.click();
            a.remove();
            setTimeout(() => URL.revokeObjectURL(url), 1000);
        },
    }));
});
