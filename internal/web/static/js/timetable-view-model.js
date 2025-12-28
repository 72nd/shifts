import {
    MINUTES_PER_DAY,
    assignOverlapLanes,
    clampToDay,
    createHourScale,
    generateTypeColor,
    parseBoundaryTime,
    resolveDayContext
} from './timetable-view-layout.js';

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
    
    // Extract UTC hours and minutes to avoid timezone conversion
    const startHours = date.getUTCHours();
    const startMinutes = date.getUTCMinutes();
    const startTime = `${String(startHours).padStart(2, '0')}:${String(startMinutes).padStart(2, '0')}`;

    // Calculate end time in UTC
    const endDate = new Date(date.getTime());
    endDate.setUTCMinutes(endDate.getUTCMinutes() + safeDuration);
    const endHours = endDate.getUTCHours();
    const endMinutes = endDate.getUTCMinutes();
    const endTime = `${String(endHours).padStart(2, '0')}:${String(endMinutes).padStart(2, '0')}`;

    return `${startTime} – ${endTime}`;
}

function formatDisplayDate(date, locale, startDate = null) {
    if (!(date instanceof Date) || Number.isNaN(date.getTime())) {
        return '';
    }
    const formatted = new Intl.DateTimeFormat(locale === 'de' ? 'de-DE' : 'en-US', {
        weekday: 'long',
        month: 'long',
        day: 'numeric'
    }).format(date);
    
    // Calculate day count if startDate is provided
    if (startDate) {
        const start = new Date(startDate);
        if (!isNaN(start.getTime())) {
            // Set both dates to midnight for accurate day calculation
            const dateMidnight = new Date(date.getFullYear(), date.getMonth(), date.getDate());
            const startMidnight = new Date(start.getFullYear(), start.getMonth(), start.getDate());
            
            // Calculate difference in days
            const diffTime = dateMidnight.getTime() - startMidnight.getTime();
            const diffDays = Math.round(diffTime / (1000 * 60 * 60 * 24));
            
            // Day count: start date is Day 1, day before is Day 0, etc.
            const dayCount = diffDays + 1;
            
            return `${formatted} (${dayCount})`;
        }
    }
    
    return formatted;
}

function normalizeShift({
    shift,
    typeMeta,
    offsetMinutes,
    locale,
    isMultilingual,
    collaborativeMode
}) {
    const participants = Array.isArray(shift.participants) ? shift.participants : [];
    const required = typeof shift.required === 'number' ? shift.required : 0;
    const durationMinutes = clampToDay(Number(shift.duration) || 0);
    const startMinutes = clampToDay(offsetMinutes);
    const endMinutes = clampToDay(startMinutes + durationMinutes);

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

    const result = {
        id: shift.id,
        typeId: typeMeta.id,
        typeTitle: typeMeta.title,
        startMinutes,
        endMinutes: Math.max(endMinutes, startMinutes + 5),
        durationMinutes,
        startTime: shift.start_time,
        displayTime: formatTimeRange(new Date(shift.start_time), durationMinutes, locale),
        participants,
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
    
    return result;
}

export function buildTimetableModel({
    shiftTypes = [],
    plan = {},
    locale = 'en'
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
            const startDate = new Date(shift.start_time);
            if (Number.isNaN(startDate.getTime())) {
                return;
            }

            const { dateKey, offsetMinutes, normalizedDate } = resolveDayContext(startDate, boundaryMinutes);

            if (!dayMap.has(dateKey)) {
                dayMap.set(dateKey, {
                    dateKey,
                    date: normalizedDate,
                    columns: new Map()
                });
            }

            const dayEntry = dayMap.get(dateKey);

            if (!dayEntry.columns.has(typeMeta.id)) {
                dayEntry.columns.set(typeMeta.id, {
                    typeId: typeMeta.id,
                    typeTitle: typeMeta.title,
                    color: typeMeta.color,
                    shifts: []
                });
            }

            const column = dayEntry.columns.get(typeMeta.id);
            column.shifts.push(normalizeShift({
                shift,
                typeMeta,
                offsetMinutes,
                locale,
                isMultilingual,
                collaborativeMode
            }));
        });
    });

    const days = Array.from(dayMap.values())
        .sort((a, b) => a.dateKey.localeCompare(b.dateKey))
        .map(dayEntry => {
            const columns = normalizedTypes.map(typeMeta => {
                const column = dayEntry.columns.get(typeMeta.id) || {
                    typeId: typeMeta.id,
                    typeTitle: typeMeta.title,
                    color: typeMeta.color,
                    shifts: []
                };
                assignOverlapLanes(column.shifts);
                return {
                    ...column,
                    hasShifts: column.shifts.length > 0
                };
            });

            return {
                dateKey: dayEntry.dateKey,
                dateLabel: formatDisplayDate(dayEntry.date, locale, plan.start_date),
                date: dayEntry.date,
                columns,
                hasShifts: columns.some(column => column.hasShifts)
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
        timeScale: createHourScale(15),
        meta: {
            boundaryMinutes,
            slotMinutes: 15,
            totalMinutes: MINUTES_PER_DAY,
            typeCount: normalizedTypes.length,
            dayCount: days.length
        }
    };
}

