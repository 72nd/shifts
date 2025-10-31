export class FilterManager {
    constructor() {
        this.reset();
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
                const shiftDate = new Date(shift.start_time).toISOString().split('T')[0];
                if (shiftDate !== this.selectedDay) return false;
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
                const date = new Date(shift.start_time);
                days.add(date.toISOString().split('T')[0]);
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