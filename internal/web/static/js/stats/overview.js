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

        cellLabel(day, hourIdx, cell) {
            const when = `${this.dayLabel(day)} ${this.hourLabel(this.model.heatmap.hours[hourIdx])}:00`;
            let what;
            if (!cell) what = this.$t('stats.heatmap_none');
            else if (cell.ratio === null) what = `∞ ${this.$t('stats.unlimited')}`;
            else what = this.$t('stats.heatmap_cell', { filled: this.number(cell.filled), required: this.number(cell.required) });
            return `${when}: ${what}`;
        },

        cellUrl(day, cell) {
            return this.planUrl({ day, status: cell.ratio !== null && cell.ratio < 1 ? 'open' : undefined, view: 'list' });
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
