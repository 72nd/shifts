import { minutesOfDay, logicalDay, dateKey } from './datetime.js';

const MINUTES_PER_DAY = 24 * 60;

function toMinutes(date) {
    return minutesOfDay(date);
}

function pad(value) {
    return String(value).padStart(2, '0');
}

export function parseBoundaryTime(boundary = '00:00') {
    if (typeof boundary !== 'string') return 0;
    const [hours, minutes] = boundary.split(':').map(num => parseInt(num, 10));
    const safeHours = Number.isFinite(hours) ? hours : 0;
    const safeMinutes = Number.isFinite(minutes) ? minutes : 0;
    const total = safeHours * 60 + safeMinutes;
    if (total < 0) return 0;
    if (total >= MINUTES_PER_DAY) return MINUTES_PER_DAY - 1;
    return total;
}

// date: floating Date (wall clock in UTC fields, see datetime.js)
export function resolveDayContext(date, boundaryMinutes) {
    const minutes = toMinutes(date);
    const normalizedDate = logicalDay(date, boundaryMinutes);

    const offsetMinutes = minutes < boundaryMinutes
        ? minutes + (MINUTES_PER_DAY - boundaryMinutes)
        : minutes - boundaryMinutes;

    return {
        dateKey: dateKey(normalizedDate),
        offsetMinutes,
        normalizedDate
    };
}

export function formatDateKey(date) {
    return dateKey(date);
}

export function clampToDay(minutes) {
    if (!Number.isFinite(minutes)) return 0;
    return Math.max(0, Math.min(minutes, MINUTES_PER_DAY));
}

export function assignOverlapLanes(shifts) {
    if (!Array.isArray(shifts) || shifts.length === 0) {
        return 0;
    }

    const laneEndTimes = [];
    const sorted = [...shifts].sort((a, b) => {
        if (a.startMinutes === b.startMinutes) {
            return (a.durationMinutes || 0) - (b.durationMinutes || 0);
        }
        return a.startMinutes - b.startMinutes;
    });

    sorted.forEach(shift => {
        const laneIndex = laneEndTimes.findIndex(end => shift.startMinutes >= end);
        if (laneIndex === -1) {
            shift.laneIndex = laneEndTimes.length;
            laneEndTimes.push(shift.endMinutes);
        } else {
            shift.laneIndex = laneIndex;
            laneEndTimes[laneIndex] = shift.endMinutes;
        }
    });

    const laneCount = laneEndTimes.length || 1;
    shifts.forEach(shift => {
        shift.laneCount = laneCount;
    });
    return laneCount;
}

/** Wall-clock label "HH:MM" for a minute offset from the day boundary (offset 1440 is the next boundary). */
export function formatClock(offsetMinutes, boundaryMinutes = 0) {
    const clock = (offsetMinutes + boundaryMinutes) % MINUTES_PER_DAY;
    return `${pad(Math.floor(clock / 60))}:${pad(clock % 60)}`;
}

/** 15 min slots of [startMinutes, endMinutes), labelled with wall-clock hours (offset + day boundary). */
export function createHourScale(slotMinutes = 15, startMinutes = 0, endMinutes = MINUTES_PER_DAY, boundaryMinutes = 0) {
    const slots = [];
    for (let minute = startMinutes; minute < endMinutes; minute += slotMinutes) {
        const clock = (minute + boundaryMinutes) % MINUTES_PER_DAY;
        const isHour = clock % 60 === 0;
        slots.push({
            minute,
            isHour,
            label: isHour ? `${pad(clock / 60)}:00` : ''
        });
    }
    return slots;
}

// Collapsing of empty stretches of a day (timetable)
export const GAP_MIN_COLLAPSE_MINUTES = 120; // collapse only if at least this much empty time is hidden
export const GAP_CONTEXT_MINUTES = 30; // empty time kept next to a shift (rounded to full hours)

/**
 * Splits a day (0..1440 minutes after the day boundary) into segments.
 * `visible` segments are drawn to scale. `gap` segments are empty stretches of
 * at least GAP_MIN_COLLAPSE_MINUTES between or around shifts, drawn as a thin band.
 * An expanded gap stays a band (to collapse it again) followed by its full-size time range.
 * Shifts never lie inside a gap, so every shift sits in exactly one visible segment.
 *
 * @param {{start: number, end: number}[]} intervals shift intervals of the whole day (all types)
 * @param {{dateKey: string, expandedGaps?: Object<string, boolean>}} options
 * @returns {{kind: 'visible'|'gap', key: string, start: number, end: number, expanded?: boolean}[]}
 */
export function buildDaySegments(intervals, { dateKey, expandedGaps = {} }) {
    const sorted = [...intervals].sort((a, b) => a.start - b.start);
    const occupied = [];
    sorted.forEach(interval => {
        const last = occupied[occupied.length - 1];
        if (last && interval.start <= last.end) {
            last.end = Math.max(last.end, interval.end);
        } else {
            occupied.push({ start: interval.start, end: interval.end });
        }
    });

    // Empty stretches between the occupied ones, shrunk to whole hours around the context
    const gaps = [];
    let cursor = 0;
    const addGap = (from, to) => {
        const start = from === 0 ? 0 : Math.ceil((from + GAP_CONTEXT_MINUTES) / 60) * 60;
        const end = to === MINUTES_PER_DAY ? MINUTES_PER_DAY : Math.floor((to - GAP_CONTEXT_MINUTES) / 60) * 60;
        if (end - start >= GAP_MIN_COLLAPSE_MINUTES) gaps.push({ start, end });
    };
    occupied.forEach(block => {
        addGap(cursor, block.start);
        cursor = block.end;
    });
    addGap(cursor, MINUTES_PER_DAY);

    const segments = [];
    let visibleFrom = 0;
    const pushVisible = (to) => {
        if (to > visibleFrom) {
            segments.push({ kind: 'visible', key: `v${visibleFrom}`, start: visibleFrom, end: to });
        }
    };
    gaps.forEach(gap => {
        const key = `${dateKey}|${gap.start}-${gap.end}`;
        const expanded = Boolean(expandedGaps[key]);
        pushVisible(gap.start);
        segments.push({ kind: 'gap', key, start: gap.start, end: gap.end, expanded });
        // Collapsed: skip the range. Expanded: the range joins the next visible segment.
        visibleFrom = expanded ? gap.start : gap.end;
    });
    pushVisible(MINUTES_PER_DAY);
    return segments;
}

export function generateTypeColor(seed) {
    const normalizedSeed = typeof seed === 'string' ? seed : String(seed ?? '');
    let hash = 0;
    for (let i = 0; i < normalizedSeed.length; i++) {
        hash = normalizedSeed.charCodeAt(i) + ((hash << 5) - hash);
    }
    const hue = Math.abs(hash) % 360;
    return `hsl(${hue}, 65%, 65%)`;
}

export { MINUTES_PER_DAY };

