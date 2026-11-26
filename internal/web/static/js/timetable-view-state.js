// State and behaviour of the timetable view, mixed into the `shiftPlanView` Alpine component
// (see composeView in shift-plan-view.js). It relies on the host for `plan`, `shiftTypes`,
// `filteredShiftTypes`, `isAdmin`, `isTimetableView` and the shift modals.
//
// Rendering lives in templates/partials/timetable-view.html and timetable-shift.html,
// the model in timetable-view-model.js and the pure layout maths in timetable-view-layout.js.
import { buildTimetableModel } from './timetable-view-model.js';

const EMPTY_MODEL = Object.freeze({ days: [], allColumns: [], normalizedTypes: [], hasAnyShifts: false });

// Fallback sticky offsets in px if the elements cannot be measured
const DAY_NAV_HEIGHT_FALLBACK = 40;
const TYPE_HEADERS_HEIGHT_FALLBACK = 36;

/** Returns a fresh object per component: state must not be shared between instances. */
export function timetableViewState() {
    return {
        // State
        timetableModel: EMPTY_MODEL, // rebuilt by initTimetableModel whenever its inputs change
        timetableLayoutObserver: null, // ResizeObserver: font size / width changes
        timetableSlotPx: 0, // resolved slot height in px (from CSS --timetable-slot-height)
        expandedTimetableShifts: {}, // { cellKey: true }, per drawn box
        shiftOverflowState: {}, // { cellKey: true } for boxes whose content is cut off
        expandedTimetableGaps: {}, // collapsed empty stretches the user opened: { 'dateKey|start-end': true }
        activeDayKey: null,
        scrollTimeout: null,

        /**
         * Keeps `timetableModel` up to date. The model is built once per change instead of
         * once per template expression and scroll event; nothing is built outside the timetable view.
         */
        initTimetableModel() {
            Alpine.effect(() => {
                if (!this.isTimetableView) return;
                this.timetableModel = buildTimetableModel({
                    shiftTypes: this.filteredShiftTypes || [],
                    plan: this.plan || {},
                    locale: this.$locale(),
                    expandedGaps: this.expandedTimetableGaps
                });
            });
        },

        /** Collapse or expand an empty stretch of a day (the band in the timetable). */
        toggleTimetableGap(key) {
            if (!key) return;
            const next = { ...this.expandedTimetableGaps };
            if (next[key]) {
                delete next[key];
            } else {
                next[key] = true;
            }
            this.expandedTimetableGaps = next;
            this.$nextTick(() => this.detectAllShiftOverflows());
        },

        getTimetableShiftStyle(shift) {
            if (!shift) return {};
            // Sizes follow the CSS variable (rem based), so they scale with the browser font size
            const slot = 'var(--timetable-slot-height)';
            // segmentTop: start relative to the visible segment (empty stretches are collapsed)
            const top = `calc(${Math.max(0, shift.segmentTop ?? shift.startMinutes ?? 0)} * ${slot} / 15)`;
            const height = `max(${slot}, calc(${shift.durationMinutes || 15} * ${slot} / 15))`;
            const laneCount = Math.max(shift.laneCount || 1, 1);
            const laneIndex = Math.min(shift.laneIndex || 0, laneCount - 1);
            const widthPercent = 100 / laneCount;
            const leftPercent = widthPercent * laneIndex;
            const horizontalPadding = laneCount > 1 ? 6 : 4;

            return {
                top,
                height,
                left: `calc(${leftPercent}% + 2px)`,
                width: `calc(${widthPercent}% - ${horizontalPadding}px)`,
                '--shift-color': shift.typeColor || 'oklch(var(--p))'
            };
        },

        isTimetableShiftExpanded(cellKey) {
            return Boolean(this.expandedTimetableShifts?.[cellKey]);
        },

        toggleTimetableShift(cellKey) {
            if (!cellKey) return;
            // Only toggle if there's overflow
            if (!this.hasShiftOverflow(cellKey)) return;
            this.expandedTimetableShifts = {
                ...this.expandedTimetableShifts,
                [cellKey]: !this.expandedTimetableShifts[cellKey]
            };
            this.$nextTick(() => this.detectAllShiftOverflows());
        },

        hasShiftOverflow(cellKey) {
            return Boolean(this.shiftOverflowState?.[cellKey]);
        },

        /** Resolve --timetable-slot-height (rem or px) to pixels. */
        readTimetableSlotPx() {
            const container = this.$refs.timetableContainer;
            if (!container) return 0;
            const raw = getComputedStyle(container).getPropertyValue('--timetable-slot-height').trim();
            const value = parseFloat(raw) || 0;
            if (raw.endsWith('rem')) {
                return value * (parseFloat(getComputedStyle(document.documentElement).fontSize) || 16);
            }
            return value;
        },

        /**
         * Re-detect card overflow when the slot height (browser font size) or the
         * container width changes; `resize` does not fire for font size changes.
         */
        observeTimetableLayout() {
            this.$nextTick(() => {
                const container = this.$refs.timetableContainer;
                if (!container || typeof ResizeObserver === 'undefined') return;
                if (this.timetableLayoutObserver) this.timetableLayoutObserver.disconnect();
                this.timetableSlotPx = this.readTimetableSlotPx();
                let lastWidth = container.clientWidth;
                this.timetableLayoutObserver = new ResizeObserver(() => {
                    const slotPx = this.readTimetableSlotPx();
                    const width = container.clientWidth;
                    // Only react to real changes: overflow state changes never resize these
                    if (slotPx === this.timetableSlotPx && width === lastWidth) return;
                    this.timetableSlotPx = slotPx;
                    lastWidth = width;
                    this.detectAllShiftOverflows();
                });
                this.timetableLayoutObserver.observe(container);
                // The slot element resizes with the font size even if the container (max-height) does not
                const slotEl = container.querySelector('.timetable-time-slot');
                if (slotEl) this.timetableLayoutObserver.observe(slotEl);
            });
        },

        /** One RAF pass over all timetable shift cells — single reactivity update (performance). */
        detectAllShiftOverflows() {
            requestAnimationFrame(() => {
                const container = this.$refs.timetableContainer;
                if (!container) return;
                const newState = {};
                container.querySelectorAll('[data-cell-key]').forEach(el => {
                    const cellKey = el.dataset.cellKey;
                    if (!cellKey) return;
                    const contentEl = el.querySelector('.timetable-shift-content');
                    if (!contentEl) return;
                    const style = getComputedStyle(el);
                    const paddingTop = parseFloat(style.paddingTop) || 0;
                    const paddingBottom = parseFloat(style.paddingBottom) || 0;
                    const availableHeight = el.clientHeight - paddingTop - paddingBottom;
                    newState[cellKey] = contentEl.scrollHeight > (availableHeight - 3);
                });
                const old = this.shiftOverflowState || {};
                const keys = Object.keys(newState);
                const same = keys.length === Object.keys(old).length && keys.every(k => old[k] === newState[k]);
                if (!same) this.shiftOverflowState = newState;
            });
        },

        /** Height in px of the sticky bars above the day sections (day navigation + type headers). */
        timetableStickyOffset() {
            const container = this.$refs.timetableContainer;
            if (!container) return DAY_NAV_HEIGHT_FALLBACK + TYPE_HEADERS_HEIGHT_FALLBACK;
            const dayNav = parseInt(getComputedStyle(container).getPropertyValue('--timetable-day-nav-height'), 10)
                || DAY_NAV_HEIGHT_FALLBACK;
            const headers = container.querySelector('.timetable-type-headers')?.offsetHeight
                || TYPE_HEADERS_HEIGHT_FALLBACK;
            return dayNav + headers;
        },

        // Scroll to a specific day in the timetable
        scrollToDay(dateKey) {
            const container = this.$refs.timetableContainer;
            const daySection = dateKey && container?.querySelector(`[data-day-key="${dateKey}"]`);
            if (!daySection) return;

            // Current scroll + offset of the day from the container top - sticky bars
            const offsetInContainer = daySection.getBoundingClientRect().top - container.getBoundingClientRect().top;
            const scrollTop = container.scrollTop + offsetInContainer - this.timetableStickyOffset();

            container.scrollTo({ top: Math.max(0, scrollTop), behavior: 'smooth' });

            // Update active day immediately for better UX
            this.activeDayKey = dateKey;
        },

        // Handle scroll events to update active day
        handleTimetableScroll() {
            // Debounce scroll handling
            if (this.scrollTimeout) {
                clearTimeout(this.scrollTimeout);
            }

            this.scrollTimeout = setTimeout(() => {
                this.updateActiveDay();
            }, 100);
        },

        // Update which day is currently visible based on scroll position
        updateActiveDay() {
            const container = this.$refs.timetableContainer;
            const days = this.timetableModel.days;
            if (!container || days.length === 0) return;

            const containerRect = container.getBoundingClientRect();
            const viewportTop = container.scrollTop + this.timetableStickyOffset() + 20; // Small offset for better UX
            const viewportBottom = container.scrollTop + container.clientHeight;

            // The day with the largest visible area wins; fall back to the first day
            let activeDay = days[0].dateKey;
            let maxVisibleArea = 0;
            days.forEach(day => {
                const daySection = container.querySelector(`[data-day-key="${day.dateKey}"]`);
                if (!daySection) return;

                const dayTop = daySection.getBoundingClientRect().top - containerRect.top + container.scrollTop;
                const dayBottom = dayTop + daySection.offsetHeight;
                const visibleArea = Math.max(0, Math.min(viewportBottom, dayBottom) - Math.max(viewportTop, dayTop));
                if (visibleArea > maxVisibleArea) {
                    maxVisibleArea = visibleArea;
                    activeDay = day.dateKey;
                }
            });

            if (this.activeDayKey !== activeDay) {
                this.activeDayKey = activeDay;
            }
        },

        // Initialize active day when timetable is shown
        initTimetableActiveDay() {
            this.$nextTick(() => {
                const days = this.timetableModel.days;
                if (days.length === 0) return;
                // Start with the first day, then follow the scroll position
                this.activeDayKey = days[0].dateKey;
                setTimeout(() => this.updateActiveDay(), 100);
            });
        },

        // The model holds copies of the shifts; actions work on the originals and need their type
        findShiftAndType(shiftId) {
            for (const type of this.shiftTypes) {
                const shift = type.shifts?.find(s => s.id === shiftId);
                if (shift) return { shift, type };
            }
            return null;
        },

        handleTimetableRegister(shift) {
            const found = this.findShiftAndType(shift?.id);
            if (found) this.openJoinModal(found.shift);
        },

        handleTimetableShiftEdit(shift) {
            const found = this.findShiftAndType(shift?.id);
            if (found) this.openShiftModal(found.shift, found.type, false);
        },

        handleTimetableShiftDuplicate(shift) {
            const found = this.findShiftAndType(shift?.id);
            if (found) this.openShiftModal(found.shift, found.type, true);
        },

        handleTimetableShiftDelete(shift) {
            const found = this.findShiftAndType(shift?.id);
            if (found) this.openDeleteShiftModal(found.shift, found.type);
        },

        handleTimetableShiftToggleLock(shift) {
            const found = this.findShiftAndType(shift?.id);
            if (!found) return;
            if (found.shift.is_locked) {
                this.unlockShift(found.shift, found.type);
            } else {
                this.lockShift(found.shift, found.type);
            }
        },

        handleTimetableParticipantRemove(shift, participant, event) {
            if (!shift || !participant) return;
            event.stopPropagation();
            const found = this.findShiftAndType(shift.id);
            if (!found) return;
            if (found.shift.is_locked && !this.isAdmin) return;
            this.openRemoveModal(found.shift, participant);
        }
    };
}
