import { parseDateTime, parseDate, serializeDate, dateKey, logicalDay } from './datetime.js';
import { parseBoundaryTime } from './timetable-view-layout.js';

// URL value of ?status= mapped to a selectedStatus value; unknown values are ignored
const URL_STATUS_VALUES = { open: 'not_full' };

export class FilterManager {
    constructor(dayBoundary = '00:00') {
        this.setDayBoundary(dayBoundary);
        // View setting, not a filter: show declined participations (only offered if there are any)
        this.showDeclined = false;
        this.reset();
    }

    // Apply filters from URL params (type, day, status, q). Called once on load; invalid values are ignored.
    applyUrlFilters(params) {
        const typeIds = (params.get('type') || '')
            .split(',')
            .map(part => part.trim())
            .filter(part => /^\d+$/.test(part))
            .map(Number)
            .filter(id => id > 0);
        this.selectedTypes = [...new Set(typeIds)];

        const day = params.get('day') || '';
        if (/^\d{4}-\d{2}-\d{2}$/.test(day)) {
            const date = parseDate(day);
            // Reject impossible dates such as 2026-02-30 (they would roll over)
            if (date && serializeDate(date) === day) this.selectedDay = day;
        }

        const status = URL_STATUS_VALUES[params.get('status')];
        if (status) this.selectedStatus = status;

        const query = (params.get('q') || '').trim();
        if (query) this.searchQuery = query.slice(0, 100);
    }

    // Set the plan's daily boundary ("HH:MM") used for the day filter
    setDayBoundary(boundary) {
        this.dayBoundaryMinutes = parseBoundaryTime(boundary || '00:00');
    }

    // Logical day key ("YYYY-MM-DD") of a shift, same as the list grouping; '' if invalid
    getShiftDayKey(shift) {
        const start = parseDateTime(shift.start_time);
        return start ? dateKey(logicalDay(start, this.dayBoundaryMinutes)) : '';
    }

    reset() {
        this.selectedTypes = [];
        this.selectedDay = '';
        this.selectedStatus = 'all';
        this.searchQuery = '';
    }

    // Check if any filters are active
    hasActiveFilters() {
        return this.selectedTypes.length > 0 ||
            this.selectedDay !== '' ||
            this.selectedStatus !== 'all' ||
            this.searchQuery.trim() !== '';
    }

    // Get count of active filters
    getActiveFilterCount() {
        let count = 0;
        if (this.selectedTypes.length > 0) count++;
        if (this.selectedDay) count++;
        if (this.selectedStatus !== 'all') count++;
        if (this.searchQuery.trim()) count++;
        return count;
    }

    // Toggle a type selection
    toggleType(typeId) {
        const index = this.selectedTypes.indexOf(typeId);
        if (index === -1) {
            this.selectedTypes.push(typeId);
        } else {
            this.selectedTypes.splice(index, 1);
        }
    }

    // Toggle all types
    toggleAllTypes(allTypeIds) {
        if (this.selectedTypes.length === allTypeIds.length) {
            this.selectedTypes = [];
        } else {
            this.selectedTypes = [...allTypeIds];
        }
    }

    // Clear all filters
    clearFilters() {
        this.reset();
    }

    // Apply filters to shift types
    filterShiftTypes(shiftTypes) {
        if (!shiftTypes) return [];

        // First filter by selected types (if any)
        const filteredTypes = this.selectedTypes.length === 0 
            ? shiftTypes 
            : shiftTypes.filter(type => this.selectedTypes.includes(type.id));

        // Then apply other filters to the shifts of each type
        return filteredTypes.map(type => ({
            ...type,
            shifts: this.filterShifts(type.shifts)
        }));
    }

    // Apply filters to shifts
    filterShifts(shifts) {
        if (!shifts) return [];

        return shifts.filter(shift => {
            // Day filter
            if (this.selectedDay) {
                if (this.getShiftDayKey(shift) !== this.selectedDay) return false;
            }

            // Status filter
            const participantCount = shift.participants?.length || 0;
            const percentage = (participantCount / shift.required) * 100;

            switch (this.selectedStatus) {
                case 'empty':
                    if (participantCount > 0) return false;
                    break;
                case 'critical':
                    if (percentage >= 50 || participantCount === shift.required) return false;
                    break;
                case 'not_full':
                    if (participantCount >= shift.required) return false;
                    break;
                case 'full':
                    if (participantCount !== shift.required) return false;
                    break;
                case 'overbooked':
                    // Overbooked: more participants than required slots
                    // Exclude unlimited shifts (required === -1) and shifts that are not overbooked
                    if (shift.required === -1 || shift.required <= 0 || participantCount <= shift.required) return false;
                    break;
            }

            // Name search
            if (this.searchQuery.trim()) {
                const query = this.searchQuery.trim().toLowerCase();
                if (!shift.participants?.some(p => p.name.toLowerCase().includes(query))) {
                    return false;
                }
            }

            return true;
        });
    }

    // Get available days from shift types
    getAvailableDays(shiftTypes) {
        const days = new Set();
        shiftTypes.forEach(type => {
            type.shifts.forEach(shift => {
                const key = this.getShiftDayKey(shift);
                if (key) days.add(key);
            });
        });
        return [...days].sort();
    }

    // Get selected type names with translations
    getSelectedTypeNames(shiftTypes, isMultilingual, currentLocale) {
        return shiftTypes
            .filter(type => this.selectedTypes.includes(type.id))
            .map(type => ({
                id: type.id,
                name: isMultilingual
                    ? (currentLocale === 'de' ? type.title_de : type.title_en)
                    : type.title_en
            }));
    }
} 