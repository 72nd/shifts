export function formatDateTime(dateTime) {
    const date = new Date(dateTime);
    return date.toLocaleString('de-DE', {
        weekday: 'short',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}

export function formatDate(dateTime) {
    const date = new Date(dateTime);
    return date.toLocaleDateString('de-DE', {
        weekday: 'long',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit'
    });
}

export function formatTime(dateTime) {
    const date = new Date(dateTime);
    return date.toLocaleTimeString('de-DE', {
        hour: '2-digit',
        minute: '2-digit'
    });
}

export function calculateEndTime(startTime, durationMinutes) {
    const start = new Date(startTime);
    const end = new Date(start.getTime() + durationMinutes * 60000);
    return end.toLocaleTimeString('de-DE', {
        hour: '2-digit',
        minute: '2-digit'
    });
}

export function getWeekNumber(date) {
    const d = new Date(date);
    d.setHours(0, 0, 0, 0);
    d.setDate(d.getDate() + 4 - (d.getDay() || 7));
    const yearStart = new Date(d.getFullYear(), 0, 1);
    return Math.ceil((((d - yearStart) / 86400000) + 1) / 7);
}

export function getDateKey(date) {
    const d = new Date(date);
    return d.toISOString().split('T')[0];
}

export function groupShiftsByDay(shifts) {
    const grouped = new Map();
    
    shifts.forEach(shift => {
        const date = new Date(shift.startTime);
        const key = getDateKey(date);
        
        if (!grouped.has(key)) {
            grouped.set(key, []);
        }
        grouped.get(key).push(shift);
    });

    // Sort shifts within each day by startTime
    grouped.forEach(dayShifts => {
        dayShifts.sort((a, b) => new Date(a.startTime) - new Date(b.startTime));
    });

    return grouped;
} 