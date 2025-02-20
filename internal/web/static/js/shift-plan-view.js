// Import the modal classes
import { ShiftModal } from './shift-modal.js';

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
        joinName: '',
        joinNameError: null,

        // Admin state
        isAdmin: false,
        adminPassword: '',
        adminError: null,
        adminToken: '',

        // Shift Type Modal state
        currentShiftType: null,
        shiftTypeForm: {
            title_de: '',
            title_en: '',
            description_de: '',
            description_en: '',
            location_de: '',
            location_en: ''
        },
        shiftTypeErrors: {
            title_de: null,
            title_en: null,
            description_de: null,
            description_en: null,
            location_de: null,
            location_en: null
        },

        // Initialize modals
        shiftModal: null,

        // Computed property für den Ladezustand
        get isReady() {
            return !this.isLoading && !this.error && this.plan;
        },

        // Computed property für verfügbare Tage
        get availableDays() {
            const days = new Set();
            this.shiftTypes.forEach(type => {
                type.shifts.forEach(shift => {
                    const date = new Date(shift.start_time);
                    days.add(date.toISOString().split('T')[0]);
                });
            });
            return [...days].sort();
        },

        // Computed property für gefilterte Schichttypen
        get filteredShiftTypes() {
            if (!this.shiftTypes) return [];
            
            return this.shiftTypes
                .filter(type => this.selectedTypes.length === 0 || this.selectedTypes.includes(type.id))
                .map(type => ({
                    ...type,
                    shiftsByDate: this.filterShiftsByDate(type.shifts)
                }))
                .filter(type => type.shiftsByDate.length > 0);
        },

        // Filter functions
        filterShiftsByDate(shifts) {
            if (!shifts) return [];

            // Filter shifts based on all criteria
            const filteredShifts = shifts.filter(shift => {
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
                    case 'full':
                        if (participantCount !== shift.required) return false;
                        break;
                }

                // Name search
                if (this.searchQuery.trim()) {
                    const query = this.searchQuery.trim().toLowerCase();
                    const hasMatchingParticipant = shift.participants?.some(
                        p => p.name.toLowerCase().includes(query)
                    );
                    if (!hasMatchingParticipant) return false;
                }

                return true;
            });

            // Group filtered shifts by date
            return this.groupShiftsByDate(filteredShifts);
        },

        // Helper function to check if any filters are active
        hasActiveFilters() {
            return this.selectedTypes.length > 0 ||
                   this.selectedDay !== '' ||
                   this.selectedStatus !== 'all' ||
                   this.searchQuery.trim() !== '';
        },

        // Filter actions
        toggleType(typeId) {
            const index = this.selectedTypes.indexOf(typeId);
            if (index === -1) {
                this.selectedTypes.push(typeId);
            } else {
                this.selectedTypes.splice(index, 1);
            }
        },

        clearFilters() {
            this.selectedTypes = [];
            this.selectedDay = '';
            this.selectedStatus = 'all';
            this.searchQuery = '';
        },

        // Helper function to count active filters
        activeFilterCount() {
            let count = 0;
            if (this.selectedTypes.length > 0) count++;
            if (this.selectedDay) count++;
            if (this.selectedStatus !== 'all') count++;
            if (this.searchQuery.trim()) count++;
            return count;
        },

        // Helper function to get names of selected types
        selectedTypeNames() {
            return this.shiftTypes
                .filter(type => this.selectedTypes.includes(type.id))
                .map(type => ({
                    id: type.id,
                    name: this.plan.is_multilingual 
                        ? (this.$locale() === 'de' ? type.title_de : type.title_en) 
                        : type.title_en
                }));
        },

        // Toggle all types
        toggleAllTypes() {
            if (this.selectedTypes.length === this.shiftTypes.length) {
                this.selectedTypes = [];
            } else {
                this.selectedTypes = this.shiftTypes.map(type => type.id);
            }
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
            
            // Initialize ShiftModal with empty data
            this.shiftModal = new ShiftModal('', []);
            
            // Hole Token aus URL
            const pathParts = window.location.pathname.split('/');
            const token = pathParts[pathParts.length - 1];
            
            // Versuche Admin-Token aus localStorage zu laden
            const storedToken = localStorage.getItem(`admin_token_${token}`);
            if (storedToken) {
                this.isAdmin = true;
                this.adminToken = storedToken;
            }
            
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

                const planResponse = await fetch(`/api/shift-plans/${token}`, { headers });
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
                const typesResponse = await fetch(`/api/shift-plans/${token}/types`, { headers });
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

                // Update ShiftModal with new data
                this.shiftModal = new ShiftModal(this.plan.start_date, this.shiftTypes);
                this.shiftModal.setTokens(this.plan.token, this.adminToken);

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
            this.currentShift = shift;
            
            // Load last used name from localStorage
            this.joinName = localStorage.getItem('last_participant_name') || '';
            
            this.joinNameError = null;
            const modal = document.getElementById('join_modal');
            if (modal) modal.showModal();
        },

        closeJoinModal() {
            this.currentShift = null;
            this.joinName = '';
            this.joinNameError = null;
            const modal = document.getElementById('join_modal');
            if (modal) modal.close();
        },

        async submitJoinModal() {
            if (!this.joinName?.trim()) {
                this.joinNameError = this.$t('shift_plans.view.error_name_required');
                return;
            }

            // Check if name already exists in this shift
            const nameExists = this.currentShift.participants?.some(
                p => p.name.toLowerCase() === this.joinName.trim().toLowerCase()
            );

            if (nameExists) {
                this.joinNameError = this.$t('shift_plans.view.error_name_exists');
                return;
            }

            try {
                const response = await fetch(`/api/shift-plans/${this.plan.token}/shifts/${this.currentShift.id}/participants`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name: this.joinName.trim() })
                });

                if (!response.ok) {
                    throw new Error('Failed to join shift');
                }

                // Save name to localStorage after successful join
                localStorage.setItem('last_participant_name', this.joinName.trim());

                // Daten neu laden und Modal schließen
                await this.loadPlan(this.plan.token);
                this.closeJoinModal();
            } catch (error) {
                console.error('Error joining shift:', error);
                this.joinNameError = this.$t('shift_plans.view.error_joining');
            }
        },

        openRemoveModal(shift, participant) {
            if (!shift || !participant) return;
            this.currentShift = shift;
            this.currentParticipant = participant;
            const modal = document.getElementById('remove_modal');
            if (modal) modal.showModal();
        },

        closeRemoveModal() {
            this.currentShift = null;
            this.currentParticipant = null;
            const modal = document.getElementById('remove_modal');
            if (modal) modal.close();
        },

        async submitRemoveModal() {
            try {
                const response = await fetch(`/api/shift-plans/${this.plan.token}/shifts/${this.currentShift.id}/participants/${this.currentParticipant.id}`, {
                    method: 'DELETE'
                });

                if (!response.ok) {
                    throw new Error('Failed to remove participant');
                }

                // Daten neu laden und Modal schließen
                await this.loadPlan(this.plan.token);
                this.closeRemoveModal();
            } catch (error) {
                console.error('Error removing participant:', error);
                alert(this.$t('shift_plans.view.error_removing'));
            }
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
            this.adminPassword = '';
            this.adminError = null;
            const modal = document.getElementById('admin_modal');
            if (modal) modal.showModal();
        },

        closeAdminModal() {
            this.adminPassword = '';
            this.adminError = null;
            const modal = document.getElementById('admin_modal');
            if (modal) modal.close();
        },

        async submitAdminLogin() {
            if (!this.adminPassword?.trim()) {
                this.adminError = this.$t('shift_plans.view.error_password_required');
                return;
            }

            try {
                const response = await fetch(`/api/shift-plans/${this.plan.token}/verify-admin`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ password: this.adminPassword.trim() })
                });

                if (!response.ok) {
                    throw new Error('Invalid password');
                }

                // Store admin state
                this.isAdmin = true;
                this.adminToken = this.adminPassword.trim();
                
                // Save admin token in localStorage
                localStorage.setItem(`admin_token_${this.plan.token}`, this.adminToken);
                
                this.closeAdminModal();

                // Reload data to get admin-only fields
                await this.loadPlan(this.plan.token);
            } catch (error) {
                console.error('Error logging in:', error);
                this.adminError = this.$t('shift_plans.view.error_invalid_password');
            }
        },

        logoutAdmin() {
            this.isAdmin = false;
            this.adminToken = '';
            this.adminPassword = '';
            
            // Remove admin token from localStorage
            localStorage.removeItem(`admin_token_${this.plan.token}`);
            
            // Reload data to remove admin-only fields
            this.loadPlan(this.plan.token);
        },

        // Shift Type Modal functions
        openShiftTypeModal(type = null) {
            this.currentShiftType = type;
            this.shiftTypeErrors = {
                title_de: null,
                title_en: null,
                description_de: null,
                description_en: null,
                location_de: null,
                location_en: null
            };
            
            if (type) {
                // Edit mode: Fill form with existing data
                this.shiftTypeForm = {
                    title_de: type.title_de || '',
                    title_en: type.title_en || '',
                    description_de: type.description_de || '',
                    description_en: type.description_en || '',
                    location_de: type.location_de || '',
                    location_en: type.location_en || ''
                };
            } else {
                // Create mode: Reset form
                this.shiftTypeForm = {
                    title_de: '',
                    title_en: '',
                    description_de: '',
                    description_en: '',
                    location_de: '',
                    location_en: ''
                };
            }
            
            const modal = document.getElementById('shift_type_modal');
            if (modal) modal.showModal();
        },

        closeShiftTypeModal() {
            this.currentShiftType = null;
            this.shiftTypeForm = this.getEmptyShiftTypeForm();
            this.shiftTypeErrors = {};
            document.getElementById('shift_type_modal').close();
        },

        openDeleteShiftTypeModal(type) {
            this.currentShiftType = type;
            document.getElementById('delete_shift_type_modal').showModal();
        },

        closeDeleteShiftTypeModal() {
            this.currentShiftType = null;
            document.getElementById('delete_shift_type_modal').close();
        },

        async submitDeleteShiftType() {
            try {
                const response = await fetch(`/api/shift-plans/${this.plan.token}/types/${this.currentShiftType.id}`, {
                    method: 'DELETE',
                    headers: {
                        'X-Admin-Token': this.adminToken
                    }
                });

                if (!response.ok) {
                    throw new Error('Failed to delete shift type');
                }

                // Remove the shift type from the list
                this.shiftTypes = this.shiftTypes.filter(type => type.id !== this.currentShiftType.id);
                this.closeDeleteShiftTypeModal();
            } catch (error) {
                console.error('Error deleting shift type:', error);
                alert(this.$t('shift_plans.view.error_deleting_shift_type'));
            }
        },

        async submitShiftTypeModal() {
            // Reset errors
            this.shiftTypeErrors = {
                title_de: null,
                title_en: null,
                description_de: null,
                description_en: null,
                location_de: null,
                location_en: null
            };
            
            let hasErrors = false;
            
            // For multilingual plans, validate both languages
            if (this.plan.is_multilingual) {
                // German validation
                if (!this.shiftTypeForm.title_de?.trim()) {
                    this.shiftTypeErrors.title_de = this.$t('shift_plans.view.error_title_required');
                    hasErrors = true;
                }
                if (!this.shiftTypeForm.description_de?.trim()) {
                    this.shiftTypeErrors.description_de = this.$t('shift_plans.view.error_description_required');
                    hasErrors = true;
                }
                if (!this.shiftTypeForm.location_de?.trim()) {
                    this.shiftTypeErrors.location_de = this.$t('shift_plans.view.error_location_required');
                    hasErrors = true;
                }
            }
            
            // Always validate English (as it's the default language)
            if (!this.shiftTypeForm.title_en?.trim()) {
                this.shiftTypeErrors.title_en = this.$t('shift_plans.view.error_title_required');
                hasErrors = true;
            }
            if (!this.shiftTypeForm.description_en?.trim()) {
                this.shiftTypeErrors.description_en = this.$t('shift_plans.view.error_description_required');
                hasErrors = true;
            }
            if (!this.shiftTypeForm.location_en?.trim()) {
                this.shiftTypeErrors.location_en = this.$t('shift_plans.view.error_location_required');
                hasErrors = true;
            }
            
            if (hasErrors) {
                return;
            }
            
            try {
                const url = this.currentShiftType
                    ? `/api/shift-plans/${this.plan.token}/types/${this.currentShiftType.id}`
                    : `/api/shift-plans/${this.plan.token}/types`;
                
                const response = await fetch(url, {
                    method: this.currentShiftType ? 'PUT' : 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-Admin-Token': this.adminToken
                    },
                    body: JSON.stringify(this.shiftTypeForm)
                });

                if (!response.ok) {
                    throw new Error(this.currentShiftType ? 'Failed to update shift type' : 'Failed to create shift type');
                }

                // Reload data and close modal
                await this.loadPlan(this.plan.token);
                this.closeShiftTypeModal();
            } catch (error) {
                console.error('Error saving shift type:', error);
                // Show error in the first error field
                this.shiftTypeErrors.title_en = this.$t(
                    this.currentShiftType 
                        ? 'shift_plans.view.error_updating_shift_type'
                        : 'shift_plans.view.error_creating_shift_type'
                );
            }
        },

        // Shift Modal functions
        openShiftModal(shift = null, type = null) {
            this.shiftModal.openWith(shift, type);
        },

        // Helper function to calculate relative day from a date
        getRelativeDayFromDate(date) {
            const startDate = new Date(this.plan.start_date);
            const targetDate = new Date(date);
            const diffTime = targetDate - startDate;
            const diffDays = Math.floor(diffTime / (1000 * 60 * 60 * 24)) + 1;
            return diffDays;
        },

        // Helper function to update start time based on relative day
        updateStartTimeFromRelativeDay(day) {
            const startDate = new Date(this.plan.start_date);
            startDate.setDate(startDate.getDate() + (parseInt(day) - 1));
            startDate.setHours(10, 0, 0, 0);
            this.shiftForm.start_time = startDate.toISOString().slice(0, 16);
        }
    }));
}); 