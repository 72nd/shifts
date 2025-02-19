document.addEventListener('alpine:init', () => {
    Alpine.data('shiftPlanView', () => ({
        // Reaktive Datenstruktur
        data: Alpine.reactive({
            plan: null,
            shiftTypes: [],
            currentLang: 'de',
            isLoading: true,
            error: null
        }),

        // Computed property für den Ladezustand
        get isReady() {
            return !this.data.isLoading && !this.data.error && this.data.plan;
        },

        // Initialisierung
        init() {
            // Setze Standardwerte
            this.data.plan = {
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
            this.data.isLoading = true;
            this.data.error = null;

            try {
                // Plan laden
                const planResponse = await fetch(`/api/shift-plans/${token}`);
                if (!planResponse.ok) {
                    throw new Error('Failed to load plan');
                }
                const planData = await planResponse.json();
                
                // Stelle sicher, dass alle Properties existieren
                this.data.plan = {
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
                this.data.shiftTypes = types.map(type => ({
                    ...type,
                    shiftsByDate: this.groupShiftsByDate(type.shifts || [])
                }));

                // Sprache setzen
                this.data.currentLang = this.data.plan.is_multilingual ? 'de' : 'en';
            } catch (error) {
                console.error('Error loading plan:', error);
                this.data.error = error.message;
                
                // Zurücksetzen auf sicheren Zustand
                this.data.plan = {
                    is_multilingual: false,
                    title_en: '',
                    title_de: '',
                    description_en: '',
                    description_de: '',
                    start_date: '',
                    end_date: '',
                    token: ''
                };
                this.data.shiftTypes = [];
            } finally {
                this.data.isLoading = false;
            }
        },

        // Teilnehmer hinzufügen
        async joinShift(shift) {
            if (!shift || !this.data.plan.token) return;

            try {
                const name = prompt(this.$store.i18n.t('shift_plans.view.enter_name'));
                if (!name) return;

                const response = await fetch(`/api/shifts/${shift.id}/participants`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name })
                });

                if (!response.ok) {
                    throw new Error('Failed to join shift');
                }

                // Daten neu laden
                await this.loadPlan(this.data.plan.token);
            } catch (error) {
                console.error('Error joining shift:', error);
                alert(this.$store.i18n.t('shift_plans.view.error_joining'));
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
                
                const locale = this.data.currentLang === 'de' ? 'de-DE' : 'en-US';
                const formatted = date.toLocaleDateString(locale, options);
                
                // Für Deutsch: Ersten Buchstaben groß schreiben
                if (this.data.currentLang === 'de') {
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
                    this.data.currentLang === 'de' ? 'de-DE' : 'en-US',
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
                    this.data.currentLang === 'de' ? 'de-DE' : 'en-US',
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