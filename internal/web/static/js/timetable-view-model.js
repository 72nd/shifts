import {
    MINUTES_PER_DAY,
    assignOverlapLanes,
    buildDaySegments,
    clampToDay,
    createHourScale,
    formatClock,
    generateTypeColor,
    parseBoundaryTime,
    resolveDayContext
} from './timetable-view-layout.js';
import { parseDateTime, parseDate, addMinutes, addDays, dateKey as toDateKey, daysBetween, logicalDay } from './datetime.js';

function getLocalizedValue({ de, en }, isMultilingual, locale) {
    if (!isMultilingual) {
        return en ?? de ?? '';
    }
    if (locale === 'de') {
        return de ?? en ?? '';
    }
    return en ?? de ?? '';
}

function formatTimeRange(date, durationMinutes, locale) {
    if (!(date instanceof Date) || Number.isNaN(date.getTime())) {
        return '';
    }

    const safeDuration = Number.isFinite(durationMinutes) ? durationMinutes : 0;
    const endDate = addMinutes(date, safeDuration);

    const formatter = new Intl.DateTimeFormat(locale === 'de' ? 'de-DE' : 'en-GB', {
        timeZone: 'UTC',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false
    });

    return `${formatter.format(date)} – ${formatter.format(endDate)}`;
}

function formatDisplayDate(date, locale, startDate = null) {
    if (!(date instanceof Date) || Number.isNaN(date.getTime())) {
        return '';
    }
    const formatted = new Intl.DateTimeFormat(locale === 'de' ? 'de-DE' : 'en-US', {
        timeZone: 'UTC',
        weekday: 'long',
        month: 'long',
        day: 'numeric'
    }).format(date);
    
    // Calculate day count if startDate is provided
    if (startDate) {
        const start = parseDate(startDate);
        if (start) {
            // Day count: start date is Day 1, day before is Day 0, etc.
            const dayCount = daysBetween(start, logicalDay(date, 0)) + 1;
            return `${formatted} (${dayCount})`;
        }
    }

    return formatted;
}

/** The day entry for a key, created on first use. */
function getDay(dayMap, key, date) {
    if (!dayMap.has(key)) {
        dayMap.set(key, { dateKey: key, date, columns: new Map() });
    }
    return dayMap.get(key);
}

function newColumn(typeMeta) {
    return { typeId: typeMeta.id, typeTitle: typeMeta.title, color: typeMeta.color, shifts: [] };
}

/** The column of a shift type within a day, created on first use. */
function getColumn(dayEntry, typeMeta) {
    if (!dayEntry.columns.has(typeMeta.id)) {
        dayEntry.columns.set(typeMeta.id, newColumn(typeMeta));
    }
    return dayEntry.columns.get(typeMeta.id);
}

/**
 * One drawn box of a shift. `id` is the shift, `cellKey` the box: a shift reaching past
 * midnight is drawn once per day it touches (see continuation boxes below).
 */
function normalizeShift({
    shift,
    typeMeta,
    offsetMinutes,
    locale,
    isMultilingual,
    collaborativeMode
}) {
    const participants = Array.isArray(shift.participants) ? shift.participants : [];
    const declinedParticipants = Array.isArray(shift.declinedParticipants) ? shift.declinedParticipants : [];
    const required = typeof shift.required === 'number' ? shift.required : 0;
    const fullDuration = Math.max(Number(shift.duration) || 0, 0);
    const startMinutes = clampToDay(offsetMinutes);
    // Drawn only up to the end of the day; the rest becomes a continuation (see splitAtDayEnd)
    const durationMinutes = Math.min(fullDuration, MINUTES_PER_DAY - startMinutes);
    const endMinutes = startMinutes + durationMinutes;

    // Determine fill status for visual styling
    let fillStatus = 'empty';
    if (required === -1) {
        fillStatus = 'unlimited';
    } else if (participants.length === 0) {
        fillStatus = 'empty';
    } else if (participants.length >= required) {
        fillStatus = 'full';
    } else {
        fillStatus = 'partial';
    }

    // Determine if user can register for this shift
    // In collaborative mode, users can register even if the shift is full
    const isLocked = Boolean(shift.is_locked);
    const canRegister = !isLocked && (collaborativeMode || required === -1 || participants.length < required);

    return {
        id: shift.id,
        cellKey: String(shift.id), // expand/overflow state is kept per drawn box (a midnight shift has two)
        typeId: typeMeta.id,
        typeTitle: typeMeta.title,
        startMinutes,
        endMinutes: Math.max(endMinutes, startMinutes + 5),
        durationMinutes,
        startTime: shift.start_time,
        overflowMinutes: fullDuration - durationMinutes,
        isContinuation: false,
        displayTime: formatTimeRange(parseDateTime(shift.start_time), fullDuration, locale),
        participants,
        declinedParticipants,
        participantsCount: participants.length,
        required,
        unlimited: required === -1,
        fillRatio: required > 0 ? Math.min(participants.length / required, 1) : 1,
        fillText: required === -1 ? `${participants.length}/∞` : `${participants.length}/${required}`,
        fillStatus,
        canRegister,
        comment: getLocalizedValue({
            de: shift.comment_de,
            en: shift.comment_en
        }, isMultilingual, locale),
        commentHtml: getLocalizedValue({
            de: shift.comment_de_html,
            en: shift.comment_en_html
        }, isMultilingual, locale),
        typeColor: typeMeta.color,
        isLocked
    };
}

/**
 * Splits one day into drawn segments (see buildDaySegments) and distributes the
 * shifts and columns over them. Shifts get `segmentTop`, their start in minutes
 * relative to their segment, which is what the template positions them by.
 */
function buildDaySegmentsForDay(dateKey, columns, boundaryMinutes, expandedGaps) {
    const intervals = [];
    columns.forEach(column => column.shifts.forEach(shift => {
        intervals.push({ start: shift.startMinutes, end: shift.endMinutes });
    }));

    const segments = buildDaySegments(intervals, { dateKey, expandedGaps }).map(segment => {
        const from = formatClock(segment.start, boundaryMinutes);
        const to = formatClock(segment.end, boundaryMinutes);
        if (segment.kind === 'gap') {
            return { ...segment, from, to, hours: Math.round((segment.end - segment.start) / 60) };
        }
        return {
            ...segment,
            slots: createHourScale(15, segment.start, segment.end, boundaryMinutes),
            slotCount: (segment.end - segment.start) / 15,
            columns: columns.map(column => {
                const shifts = column.shifts.filter(shift =>
                    shift.startMinutes >= segment.start && shift.startMinutes < segment.end);
                shifts.forEach(shift => { shift.segmentTop = shift.startMinutes - segment.start; });
                return { ...column, shifts };
            })
        };
    });
    const firstVisible = segments.find(segment => segment.kind === 'visible');
    if (firstVisible) firstVisible.isFirstVisible = true;
    return segments;
}

export function buildTimetableModel({
    shiftTypes = [],
    plan = {},
    locale = 'en',
    expandedGaps = {}
}) {
    const boundaryMinutes = parseBoundaryTime(plan.day_boundary_time || '00:00');
    const isMultilingual = Boolean(plan.is_multilingual);
    const collaborativeMode = Boolean(plan.collaborative_mode);
    const normalizedTypes = shiftTypes.map((type, index) => {
        const title = getLocalizedValue({
            de: type.title_de,
            en: type.title_en
        }, isMultilingual, locale);

        return {
            id: type.id,
            title,
            color: type.color || generateTypeColor(`${type.id ?? index}`)
        };
    });

    const dayMap = new Map();

    shiftTypes.forEach((type, index) => {
        const typeMeta = normalizedTypes[index];
        if (!typeMeta || !Array.isArray(type.shifts)) {
            return;
        }

        type.shifts.forEach(shift => {
            const startDate = parseDateTime(shift.start_time);
            if (!startDate) {
                return;
            }

            const { dateKey, offsetMinutes, normalizedDate } = resolveDayContext(startDate, boundaryMinutes);

            const dayEntry = getDay(dayMap, dateKey, normalizedDate);
            const column = getColumn(dayEntry, typeMeta);
            const normalized = normalizeShift({
                shift,
                typeMeta,
                offsetMinutes,
                locale,
                isMultilingual,
                collaborativeMode
            });
            column.shifts.push(normalized);

            // A shift reaching past the end of its day continues at the top of the following day(s)
            let rest = normalized.overflowMinutes;
            let continuationDate = normalizedDate;
            while (rest > 0) {
                continuationDate = addDays(continuationDate, 1);
                const key = toDateKey(continuationDate);
                const nextColumn = getColumn(getDay(dayMap, key, continuationDate), typeMeta);
                const visible = Math.min(rest, MINUTES_PER_DAY);
                rest -= visible;
                nextColumn.shifts.push({
                    ...normalized,
                    startMinutes: 0,
                    endMinutes: Math.max(visible, 5),
                    durationMinutes: visible,
                    overflowMinutes: rest,
                    isContinuation: true,
                    cellKey: `${normalized.id}@${key}`
                });
            }
        });
    });

    const days = Array.from(dayMap.values())
        .sort((a, b) => a.dateKey.localeCompare(b.dateKey))
        .map(dayEntry => {
            const columns = normalizedTypes.map(typeMeta => {
                const column = dayEntry.columns.get(typeMeta.id) || newColumn(typeMeta);
                assignOverlapLanes(column.shifts);
                return {
                    ...column,
                    hasShifts: column.shifts.length > 0
                };
            });

            const segments = buildDaySegmentsForDay(dayEntry.dateKey, columns, boundaryMinutes, expandedGaps);
            return {
                dateKey: dayEntry.dateKey,
                dateLabel: formatDisplayDate(dayEntry.date, locale, plan.start_date),
                date: dayEntry.date,
                columns,
                hasShifts: columns.some(column => column.hasShifts),
                segments,
                // Exact height of the day (placeholder size for content-visibility)
                visibleSlots: segments.reduce((sum, segment) => sum + (segment.slotCount || 0), 0),
                gapCount: segments.filter(segment => segment.kind === 'gap').length
            };
        })
        .filter(day => day.hasShifts);

    // Create continuous column structure for unified table
    const allColumns = [];
    days.forEach((day, dayIndex) => {
        day.columns.forEach((column, colIndex) => {
            allColumns.push({
                ...column,
                dayKey: day.dateKey,
                dayLabel: day.dateLabel,
                dayDate: day.date,
                dayIndex,
                isFirstColumnOfDay: colIndex === 0,
                isLastColumnOfDay: colIndex === day.columns.length - 1
            });
        });
    });

    return {
        days,
        allColumns,
        normalizedTypes, // Export normalized types for header
        hasAnyShifts: allColumns.some(col => col.hasShifts),
        meta: {
            boundaryMinutes,
            slotMinutes: 15,
            totalMinutes: MINUTES_PER_DAY,
            typeCount: normalizedTypes.length,
            dayCount: days.length
        }
    };
}

