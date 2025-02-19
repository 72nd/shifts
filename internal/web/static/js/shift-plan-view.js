document.addEventListener('alpine:init', () => {
    Alpine.data('shiftPlanView', () => ({
        // State
        plan: null,
        shiftTypes: [],
        isLoading: true,
        error: null,
        
        // Modal state
        currentShift: null,
        currentParticipant: null,
        joinName: '',
        joinNameError: null,

        // Computed property für den Ladezustand
        get isReady() {
            return !this.isLoading && !this.error && this.plan;
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
                const planResponse = await fetch(`/api/shift-plans/${token}`);
                if (!planResponse.ok) {
                    throw new Error('Failed to load plan');
                }
                const planData = await planResponse.json();
                
                // Stelle sicher, dass alle Properties existieren
                this.plan = {
                    is_multilingual: planData.is_multilingual || false,
                    title_en: planData.title_en || '',
                    title_de: planData.title_de || '',
                    description_en: planData.description_en || '',
                    description_de: planData.description_de || '',
                    start_date: planData.start_date || '',
                    end_date: planData.end_date || '',
                    token: planData.token || token
                };

                // Schichttypen laden
                const typesResponse = await fetch(`/api/shift-plans/${token}/types`);
                if (!typesResponse.ok) {
                    throw new Error('Failed to load shift types');
                }
                const types = await typesResponse.json();
                
                // Gruppiere Schichten nach Datum für jeden Typ
                this.shiftTypes = types.map(type => ({
                    ...type,
                    shiftsByDate: this.groupShiftsByDate(type.shifts || [])
                }));

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
            this.joinName = '';
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
                const response = await fetch(`/api/shifts/${this.currentShift.id}/participants`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name: this.joinName.trim() })
                });

                if (!response.ok) {
                    throw new Error('Failed to join shift');
                }

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
                const response = await fetch(`/api/shifts/${this.currentShift.id}/participants/${this.currentParticipant.id}`, {
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
        }
    }));
}); 