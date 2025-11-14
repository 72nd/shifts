const MINUTES_PER_DAY = 24 * 60;

function toMinutes(date) {
    return date.getHours() * 60 + date.getMinutes();
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

export function resolveDayContext(date, boundaryMinutes) {
    const dateClone = new Date(date.getTime());
    const minutes = toMinutes(dateClone);

    if (minutes < boundaryMinutes) {
        dateClone.setDate(dateClone.getDate() - 1);
    }

    const offsetMinutes = minutes < boundaryMinutes
        ? minutes + (MINUTES_PER_DAY - boundaryMinutes)
        : minutes - boundaryMinutes;

    const normalizedDate = new Date(dateClone.getFullYear(), dateClone.getMonth(), dateClone.getDate());

    return {
        dateKey: formatDateKey(normalizedDate),
        offsetMinutes,
        normalizedDate
    };
}

export function formatDateKey(date) {
    const year = date.getFullYear();
    const month = pad(date.getMonth() + 1);
    const day = pad(date.getDate());
    return `${year}-${month}-${day}`;
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

export function createHourScale(slotMinutes = 15) {
    const slots = [];
    for (let minute = 0; minute < MINUTES_PER_DAY; minute += slotMinutes) {
        const isHour = minute % 60 === 0;
        slots.push({
            minute,
            isHour,
            label: isHour ? `${pad(Math.floor(minute / 60))}:00` : ''
        });
    }
    return slots;
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

