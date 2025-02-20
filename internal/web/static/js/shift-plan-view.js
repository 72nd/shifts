// Import the modal classes
import { ShiftModal } from './shift-modal.js';
import { ShiftTypeModal } from './shift-type-modal.js';
import { DeleteTypeModal } from './delete-type-modal.js';
import { JoinModal } from './join-modal.js';
import { RemoveModal } from './remove-modal.js';
import { AdminModal } from './admin-modal.js';
import { FilterManager } from './filter-manager.js';

document.addEventListener('alpine:init', () => {
    Alpine.data('shiftPlanView', () => ({
        // State
        plan: null,
        shiftTypes: [],
        isLoading: true,
        error: null,

        // Filter state
        selectedTypes: [],
        selectedDay: '',
        selectedStatus: 'all',
        searchQuery: '',

        // Modal state
        currentShift: null,
        currentParticipant: null,

        // Admin state
        isAdmin: false,
        adminToken: '',

        // Initialize managers
        filterManager: null,
        shiftModal: null,
        shiftTypeModal: null,
        deleteTypeModal: null,
        joinModal: null,
        removeModal: null,
        adminModal: null,

        // Computed property für den Ladezustand
        get isReady() {
            return !this.isLoading && !this.error && this.plan;
        },

        // Computed property für verfügbare Tage
        get availableDays() {
            return this.filterManager.getAvailableDays(this.shiftTypes);
        },

        // Computed property für gefilterte Schichttypen
        get filteredShiftTypes() {
            return this.filterManager.filterShiftTypes(this.shiftTypes);
        },

        // Filter functions
        filterShiftsByDate(shifts) {
            return shifts;
        },

        // Filter actions
        toggleType(typeId) {
            this.filterManager.toggleType(typeId);
        },

        clearFilters() {
            this.filterManager.clearFilters();
        },

        activeFilterCount() {
            return this.filterManager.getActiveFilterCount();
        },

        selectedTypeNames() {
            return this.filterManager.getSelectedTypeNames(this.shiftTypes, this.plan.is_multilingual, this.$locale());
        },

        toggleAllTypes() {
            this.filterManager.toggleAllTypes(this.shiftTypes.map(t => t.id));
        },

        // Initialisierung
        init() {
            // Setze Standardwerte
            this.plan = {
                is_multilingual: false,
                title_en: '',
                title_de: '',
                description_en: '',
                description_de: '',
                start_date: '',
                end_date: '',
                token: ''
            };

            // Hole Token aus URL
            const pathParts = window.location.pathname.split('/');
            const token = pathParts[pathParts.length - 1];

            // Versuche Admin-Token aus localStorage zu laden
            const storedToken = localStorage.getItem(`admin_token_${token}`);
            if (storedToken) {
                this.isAdmin = true;
                this.adminToken = storedToken;
            }

            // Initialize managers
            this.filterManager = new FilterManager();
            this.shiftModal = new ShiftModal('', [], token, this.adminToken);
            this.shiftTypeModal = new ShiftTypeModal(false, token, this.adminToken, type => this.handleShiftTypeSuccess(type));
            this.deleteTypeModal = new DeleteTypeModal(token, this.adminToken, type => this.handleDeleteTypeSuccess(type));
            this.joinModal = new JoinModal(token, shift => this.handleShiftUpdate(shift));
            this.removeModal = new RemoveModal(token, shift => this.handleShiftUpdate(shift));
            this.adminModal = new AdminModal(token, token => this.handleAdminSuccess(token));

            // Starte Datenladen
            this.loadPlan(token);
        },

        // Gruppiere Schichten nach Datum
        groupShiftsByDate(shifts) {
            const groups = {};

            // Sortiere Schichten nach Startzeit
            shifts.sort((a, b) => new Date(a.start_time) - new Date(b.start_time));

            // Gruppiere nach Datum
            shifts.forEach(shift => {
                const date = new Date(shift.start_time);
                const dateKey = date.toISOString().split('T')[0];

                if (!groups[dateKey]) {
                    groups[dateKey] = [];
                }
                groups[dateKey].push(shift);
            });

            // Konvertiere in Array von Gruppen
            return Object.entries(groups).map(([date, shifts]) => ({
                date,
                shifts
            }));
        },

        // Daten laden
        async loadPlan(token) {
            this.isLoading = true;
            this.error = null;

            try {
                // Plan laden
                const headers = {};
                if (this.isAdmin && this.adminToken) {
                    headers['X-Admin-Token'] = this.adminToken;
                }

                const planResponse = await fetch(`/api/plans/${token}`, { headers });
                if (!planResponse.ok) {
                    throw new Error('Failed to load plan');
                }
                const planData = await planResponse.json();

                console.log('Loaded plan data:', planData); // Debug log

                // Stelle sicher, dass alle Properties existieren
                this.plan = {
                    is_multilingual: Boolean(planData.is_multilingual), // Explicitly convert to boolean
                    title_en: planData.title_en || '',
                    title_de: planData.title_de || '',
                    description_en: planData.description_en || '',
                    description_de: planData.description_de || '',
                    start_date: planData.start_date || '',
                    end_date: planData.end_date || '',
                    token: planData.token || token
                };

                console.log('Processed plan data:', this.plan); // Debug log

                // Schichttypen laden
                const typesResponse = await fetch(`/api/plans/${token}/types`, { headers });
                if (!typesResponse.ok) {
                    throw new Error('Failed to load shift types');
                }
                const types = await typesResponse.json();

                // Ensure each type has a shifts array and group shifts by date
                this.shiftTypes = types.map(type => ({
                    ...type,
                    shifts: type.shifts || [], // Ensure shifts is at least an empty array
                    shiftsByDate: this.groupShiftsByDate(type.shifts || [])
                }));

                // Update modals with new data
                this.shiftModal = new ShiftModal(this.plan.start_date, this.shiftTypes, this.plan.token, this.adminToken);
                this.shiftTypeModal = new ShiftTypeModal(this.plan.is_multilingual, this.plan.token, this.adminToken, type => this.handleShiftTypeSuccess(type));
                this.deleteTypeModal = new DeleteTypeModal(this.plan.token, this.adminToken, type => this.handleDeleteTypeSuccess(type));
                this.joinModal = new JoinModal(this.plan.token, shift => this.handleShiftUpdate(shift));
                this.removeModal = new RemoveModal(this.plan.token, shift => this.handleShiftUpdate(shift));

            } catch (error) {
                console.error('Error loading plan:', error);
                this.error = error.message;

                // Zurücksetzen auf sicheren Zustand
                this.plan = {
                    is_multilingual: false,
                    title_en: '',
                    title_de: '',
                    description_en: '',
                    description_de: '',
                    start_date: '',
                    end_date: '',
                    token: ''
                };
                this.shiftTypes = [];
            } finally {
                this.isLoading = false;
            }
        },

        // Modal functions
        openJoinModal(shift) {
            if (!shift) return;
            this.joinModal.openWith(shift);
        },

        openRemoveModal(shift, participant) {
            if (!shift || !participant) return;
            this.removeModal.openWith(shift, participant);
        },

        // Formatierungshelfer mit Null-Checks
        formatDate(dateStr) {
            if (!dateStr) return '';
            try {
                const date = new Date(dateStr);
                if (isNaN(date.getTime())) return '';

                const options = {
                    weekday: 'long',
                    year: 'numeric',
                    month: 'long',
                    day: 'numeric'
                };

                // Use the current language from Alpine store
                const locale = this.$locale() === 'de' ? 'de-DE' : 'en-US';
                const formatted = date.toLocaleDateString(locale, options);

                // For German: Capitalize first letter
                if (this.$locale() === 'de') {
                    return formatted.charAt(0).toUpperCase() + formatted.slice(1);
                }
                return formatted;
            } catch (error) {
                console.error('Error formatting date:', error);
                return '';
            }
        },

        formatTime(dateStr, duration) {
            if (!dateStr) return '';
            try {
                const startDate = new Date(dateStr);
                if (isNaN(startDate.getTime())) return '';

                // Format start time in 24h format
                const startTime = startDate.toLocaleTimeString(
                    this.$locale() === 'de' ? 'de-DE' : 'en-US',
                    {
                        hour: '2-digit',
                        minute: '2-digit',
                        hour12: false
                    }
                );

                // Calculate end time
                const endDate = new Date(startDate.getTime());
                endDate.setMinutes(endDate.getMinutes() + duration);
                const endTime = endDate.toLocaleTimeString(
                    this.$locale() === 'de' ? 'de-DE' : 'en-US',
                    {
                        hour: '2-digit',
                        minute: '2-digit',
                        hour12: false
                    }
                );

                return `${startTime} – ${endTime}`;
            } catch (error) {
                console.error('Error formatting time:', error);
                return '';
            }
        },

        formatDuration(minutes) {
            if (!minutes || isNaN(minutes)) return '';
            try {
                const hours = Math.floor(minutes / 60);
                const mins = minutes % 60;
                return `${hours.toString().padStart(2, '0')}:${mins.toString().padStart(2, '0')}`;
            } catch (error) {
                console.error('Error formatting duration:', error);
                return '';
            }
        },

        // Admin functions
        openAdminModal() {
            this.adminModal.open();
        },

        handleAdminSuccess(token) {
            // Store admin state
            this.isAdmin = true;
            this.adminToken = token;

            // Save admin token in localStorage
            localStorage.setItem(`admin_token_${this.plan.token}`, token);

            // Reload data to get admin-only fields
            this.loadPlan(this.plan.token);
        },

        logoutAdmin() {
            this.isAdmin = false;
            this.adminToken = '';

            // Remove admin token from localStorage
            localStorage.removeItem(`admin_token_${this.plan.token}`);

            // Reload data to remove admin-only fields
            this.loadPlan(this.plan.token);
        },

        // Shift Type Modal functions
        openShiftTypeModal(type = null) {
            this.shiftTypeModal.openWith(type);
        },

        handleShiftTypeSuccess(type) {
            // If editing an existing type, update it in the list
            const existingIndex = this.shiftTypes.findIndex(t => t.id === type.id);
            if (existingIndex !== -1) {
                // Preserve the shifts when updating
                type.shifts = this.shiftTypes[existingIndex].shifts || [];
                type.shiftsByDate = this.groupShiftsByDate(type.shifts);
                this.shiftTypes.splice(existingIndex, 1, type);
            } else {
                // Add new type to the list
                type.shifts = [];
                type.shiftsByDate = [];
                this.shiftTypes.push(type);
            }
        },

        // Shift Modal functions
        openShiftModal(shift = null, type = null) {
            this.shiftModal.openWith(shift, type);
        },

        // Delete Type Modal functions
        openDeleteShiftTypeModal(type) {
            this.deleteTypeModal.openWith(type);
        },

        handleDeleteTypeSuccess(type) {
            // Remove the type from the list
            const index = this.shiftTypes.findIndex(t => t.id === type.id);
            if (index !== -1) {
                this.shiftTypes.splice(index, 1);
            }
        },

        // Helper function to update a shift in the shiftTypes array
        handleShiftUpdate(updatedShift) {
            // Find the type that contains this shift
            const typeIndex = this.shiftTypes.findIndex(type =>
                type.shifts.some(shift => shift.id === updatedShift.id)
            );

            if (typeIndex !== -1) {
                // Update the shift in the type's shifts array
                const shiftIndex = this.shiftTypes[typeIndex].shifts.findIndex(
                    shift => shift.id === updatedShift.id
                );

                if (shiftIndex !== -1) {
                    this.shiftTypes[typeIndex].shifts[shiftIndex] = updatedShift;
                    // Update the shiftsByDate grouping
                    this.shiftTypes[typeIndex].shiftsByDate = this.groupShiftsByDate(this.shiftTypes[typeIndex].shifts);
                }
            }
        }
    }));
}); 