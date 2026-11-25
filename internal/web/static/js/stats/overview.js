// Overview tab: presentation helpers only (rows for the bars, heatmap cell styling, "now" marker).
// All numbers come from the StatsModel; no statistics are computed here.
import { nowInZone, logicalDay, dateKey, minutesOfDay } from '../datetime.js';

// Opacity steps of the primary colour by fill ratio. Inline styles because the
// Tailwind CDN build does not know the daisyUI colour tokens with opacity.
const STEPS = [
    [1, 1],      // full
    [0.75, 0.75],
    [0.5, 0.5],
    [0, 0.25],   // below half
];

document.addEventListener('alpine:init', () => {
    Alpine.data('statsOverview', () => ({
        now: null,
        timer: null,
        heatDay: null, // day shown when the heatmap rows are areas (local, not in the URL)

        init() {
            this.tick();
            this.timer = setInterval(() => this.tick(), 60000);
        },

        destroy() {
            clearInterval(this.timer);
        },

        tick() {
            if (this.plan?.time_zone) this.now = nowInZone(this.plan.time_zone);
        },

        // --- Bars ---

        get dayRows() {
            if (!this.model) return [];
            return this.model.days.map(d => this.barRow('day', d.day, this.dayLabel(d.day), d,
                this.filter.days.includes(d.day)));
        },

        get typeRows() {
            if (!this.model) return [];
            return this.model.types.map(t => this.barRow('type', t.typeId, this.t(t.title), t,
                this.filter.typeIds.includes(t.typeId)));
        },

        barRow(kind, key, label, d, selected) {
            return {
                kind, key, label, selected,
                percent: d.percent,
                low: d.percent !== null && d.percent < 50,
                full: d.percent === 100,
                open: d.open,
            };
        },

        selectRow(row) {
            if (row.kind === 'day') this.setDayFilter(row.selected ? '' : row.key);
            else this.setTypeFilter(row.selected ? 0 : row.key);
        },

        // --- Heatmap ---

        hourLabel(h) {
            return String(h).padStart(2, '0');
        },

        cellStyle(cell) {
            if (!cell) return '';
            if (cell.ratio === null) return '';
            const step = STEPS.find(([min]) => cell.ratio >= min) || STEPS[STEPS.length - 1];
            let style = `background-color: oklch(var(--p) / ${step[1]});`;
            if (cell.ratio < 0.5) {
                style += ' background-image: repeating-linear-gradient(45deg, transparent 0 3px, oklch(var(--er) / 0.6) 3px 5px);';
            }
            return style;
        },

        cellClass(cell) {
            if (cell.ratio === null) return 'border border-dashed border-base-content/40 bg-base-200';
            if (cell.ratio < 0.5) return 'border-2 border-error';
            return 'border border-transparent';
        },

        /** Hour columns of the grid: all 24 by day, trimmed to the hours with shifts by area. */
        get heatHours() {
            const hm = this.model.heatmap;
            const idx = this.heatRows === 'types' ? this.typeHourIdx : hm.hours.map((_, i) => i);
            return idx.map(hourIdx => ({ hourIdx, hour: hm.hours[hourIdx] }));
        },

        /**
         * Rows of the grid in both modes: { key, label, day, dayIdx, typeId, cells: [{hourIdx, cell}] }.
         * dayIdx always refers to model.heatmap.days (typeHeatmap.days is the same list), so the
         * "now" marker works in both modes.
         */
        get heatLines() {
            const hm = this.model.heatmap;
            if (this.heatRows !== 'types') {
                return hm.days.map((day, dayIdx) => ({
                    key: day, label: this.dayLabel(day), day, dayIdx, typeId: null,
                    cells: hm.cells[dayIdx].map((cell, hourIdx) => ({ hourIdx, cell })),
                }));
            }
            const day = this.typeDay;
            const dayIdx = this.typeDayIdx;
            const hours = this.typeHourIdx;
            return this.model.typeHeatmap.rows
                .filter(row => row.cells[dayIdx].some(c => c))
                .map(row => ({
                    key: row.typeId, label: this.t(row.title), day, dayIdx, typeId: row.typeId,
                    cells: hours.map(hourIdx => ({ hourIdx, cell: row.cells[dayIdx][hourIdx] })),
                }));
        },

        /** Day shown by area: the chosen one, else today (logical day), else the first day. */
        get typeDay() {
            const days = this.model.typeHeatmap.days;
            if (this.heatDay && days.includes(this.heatDay)) return this.heatDay;
            const today = this.nowMark ? days[this.nowMark.dayIdx] : null;
            return today || days[0];
        },

        get typeDayIdx() {
            return this.model.typeHeatmap.days.indexOf(this.typeDay);
        },

        canStep(delta) {
            const i = this.typeDayIdx + delta;
            return i >= 0 && i < this.model.typeHeatmap.days.length;
        },

        stepDay(delta) {
            if (this.canStep(delta)) this.heatDay = this.model.typeHeatmap.days[this.typeDayIdx + delta];
        },

        /** Hour indices from the first to the last hour with a shift in any area on any day (stable across days). */
        get typeHourIdx() {
            let first = 24, last = -1;
            for (const row of this.model.typeHeatmap.rows) {
                for (const day of row.cells) {
                    day.forEach((c, i) => {
                        if (!c) return;
                        if (i < first) first = i;
                        if (i > last) last = i;
                    });
                }
            }
            if (last < 0) return [];
            return Array.from({ length: last - first + 1 }, (_, i) => first + i);
        },

        cellLabel(line, hourIdx, cell) {
            let when = `${this.dayLabel(line.day)} ${this.hourLabel(this.model.heatmap.hours[hourIdx])}:00`;
            if (line.typeId !== null) when = `${line.label}, ${when}`;
            let what;
            if (!cell) what = this.$t('stats.heatmap_none');
            else if (cell.ratio === null) what = `∞ ${this.$t('stats.unlimited')}`;
            else if (cell.openPeak > 0) {
                what = this.$t('stats.heatmap_cell_open', {
                    filled: this.number(cell.filled), required: this.number(cell.required), open: this.number(cell.openPeak),
                });
            } else what = this.$t('stats.heatmap_cell', { filled: this.number(cell.filled), required: this.number(cell.required) });
            return `${when}: ${what}`;
        },

        cellUrl(line, cell) {
            return this.planUrl({
                type: line.typeId ?? undefined,
                day: line.day,
                status: cell.ratio !== null && cell.ratio < 1 ? 'open' : undefined,
                view: 'list',
            });
        },

        /** Position of the current time in the heatmap grid, or null when outside. */
        get nowMark() {
            const hm = this.model?.heatmap;
            if (!this.now || !hm || !hm.days.length) return null;
            const boundary = hm.hours[0] * 60;
            const dayIdx = hm.days.indexOf(dateKey(logicalDay(this.now, boundary)));
            if (dayIdx < 0) return null;
            const offset = (minutesOfDay(this.now) - boundary + 1440) % 1440;
            return { dayIdx, hourIdx: Math.floor(offset / 60), frac: (offset % 60) / 60 };
        },

        isNow(dayIdx, hourIdx) {
            const m = this.nowMark;
            return !!m && m.dayIdx === dayIdx && m.hourIdx === hourIdx;
        },
    }));
});
