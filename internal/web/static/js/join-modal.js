import { Modal } from './modal.js';
import { fromInputValue, serializeDateTime } from './datetime.js';

export class JoinModal extends Modal {
    constructor(planToken, onSuccess, adminToken = '', onFull = null) {
        super('join_modal');
        this.planToken = planToken;
        this.adminToken = adminToken;
        this.onSuccess = onSuccess;
        this.onFull = onFull; // called on 409 so the caller can refresh its data
        this.reset();
    }

    reset() {
        super.clearError();
        this.errors = {}; // Field-level validation errors
        this.currentShift = null;
        this.plan = null;
        this.name = localStorage.getItem('last_participant_name') || '';
        // Step 2 ("Your details"): only for plans that ask participants for person fields or attendance
        this.step = 1;
        this.known = null; // answer of the people lookup for the entered name
        this.knownName = '';
        this.from = '';
        this.until = '';
        this.values = {};
    }

    openWith(shift, plan) {
        this.reset();
        this.currentShift = shift;
        this.plan = plan;
        this.open();
        this.lookup();
    }

    /** Person fields a participant fills in. */
    get participantFields() {
        return (this.plan?.fields || []).filter(f => f.editable_by === 'participant');
    }

    get asksAttendance() {
        return this.plan?.attendance_mode === 'participant';
    }

    /** Whether the plan asks for any details at all (otherwise joining is one step as always). */
    get asksDetails() {
        return this.participantFields.length > 0 || this.asksAttendance;
    }

    /** Whether the person was found and has everything the plan requires. */
    get isComplete() {
        return !!this.known?.exists && this.missingRequired(this.known.values || {}).length === 0;
    }

    missingRequired(values) {
        return this.participantFields.filter(f => f.required && (values[f.id] === undefined || values[f.id] === null || values[f.id] === ''));
    }

    /** Looks up what the person entered earlier (only for plans that ask for details). */
    async lookup() {
        const name = this.name.trim();
        if (!this.asksDetails || !name || name === this.knownName) return;
        this.knownName = name;
        try {
            const response = await fetch(`/api/plans/${this.planToken}/people/lookup?name=${encodeURIComponent(name)}`);
            this.known = response.ok ? await response.json() : null;
        } catch (e) {
            this.known = null;
        }
    }

    /** Opens step 2, filled with what is known about the person. */
    showDetails() {
        const known = this.known?.exists ? this.known : null;
        this.values = { ...(known?.values || {}) };
        this.from = known?.available_from ? known.available_from.slice(0, 16) : '';
        this.until = known?.available_until ? known.available_until.slice(0, 16) : '';
        this.step = 2;
    }

    backToName() {
        this.step = 1;
    }

    /** Request part "person" from step 2 (null if nothing was entered). */
    personPayload() {
        if (this.step !== 2) return null;
        const values = {};
        for (const f of this.participantFields) {
            const v = this.values[f.id];
            if (v === undefined || v === null || v === '') continue;
            values[f.id] = f.kind === 'number' ? Number(v) : v;
        }
        const person = { values };
        if (this.asksAttendance) {
            if (this.from) person.available_from = serializeDateTime(fromInputValue(this.from));
            if (this.until) person.available_until = serializeDateTime(fromInputValue(this.until));
        }
        return person;
    }

    validateDetails() {
        this.errors = {};
        const missing = this.missingRequired(this.values);
        if (missing.length) this.errors.details = AlpineI18n.t('shift_plans.view.error_details_required');
        if (this.from && this.until && this.until < this.from) {
            this.errors.details = AlpineI18n.t('shift_plans.view.error_attendance_order');
        }
        return !this.errors.details;
    }

    validate() {
        this.errors = {};
        let isValid = true;

        if (!this.name?.trim()) {
            this.errors.name = AlpineI18n.t('shift_plans.view.error_name_required');
            isValid = false;
        } else if (this.name.trim().length > 20) {
            this.errors.name = AlpineI18n.t('shift_plans.view.error_name_too_long', { max: 20 });
            isValid = false;
        }

        // Check if name already exists in this shift
        const nameExists = this.currentShift.participants?.some(
            p => p.name.toLowerCase() === this.name.trim().toLowerCase()
        );

        if (nameExists) {
            this.errors.name = AlpineI18n.t('shift_plans.view.error_name_exists');
            isValid = false;
        }

        return isValid;
    }

    async submit() {
        // Bei Validierungsfehlern zeigen wir nur die Feldfehlermeldungen an
        // und aktivieren nicht das Error-Display
        if (this.step === 1 && !this.validate()) {
            return false;
        }

        if (this.asksDetails) {
            if (this.step === 1) {
                await this.lookup();
                // Known people with complete details join in one step; everybody else enters details first
                if (!this.isComplete) {
                    this.showDetails();
                    return false;
                }
            } else if (!this.validateDetails()) {
                return false;
            }
        }

        const person = this.personPayload();
        const body = person ? { name: this.name.trim(), person } : { name: this.name.trim() };

        const url = `/api/plans/${this.planToken}/shifts/${this.currentShift.id}/participants`;
        const options = {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        };
        if (this.adminToken) {
            options.headers['X-Admin-Token'] = this.adminToken;
        }

        const errorMessage = AlpineI18n.t('shift_plans.view.error_joining');
        
        // Verwende die sendRequest-Methode aus der Modal-Klasse, aber ohne automatische Fehlerbehandlung
        const { data, status, response } = await this.sendRequest(url, options, errorMessage, false);
        
        // Fehlerbehandlung
        if (data === null) {
            // Spezielle Behandlung für 409 Conflict: bereits abgelehnt oder Schicht voll
            if (status === 409) {
                const text = response ? await response.clone().text() : '';
                if (text.startsWith('participant_declined')) {
                    this.setUserFriendlyError(
                        AlpineI18n.t('shift_plans.view.error_joining'),
                        AlpineI18n.t('shift_plans.view.join_declined')
                    );
                    if (this.onFull) this.onFull();
                    return false;
                }
                this.setUserFriendlyError(
                    AlpineI18n.t('shift_plans.view.error_shift_full_title'),
                    AlpineI18n.t('shift_plans.view.error_shift_full_message')
                );
                if (this.onFull) this.onFull();
                return false;
            }
            
            // Für alle anderen Fehler verwenden wir die Standard-Fehlerbehandlung
            if (response) {
                const requestInfo = {
                    url,
                    method: options.method,
                    headers: options.headers,
                    body,
                    timestamp: new Date().toISOString()
                };
                await this.handleResponseError(response, errorMessage, requestInfo);
            }
            return false;
        }
        
        // Save name to localStorage after successful join
        localStorage.setItem('last_participant_name', this.name.trim());

        // Update the current shift's participants
        this.currentShift.participants = this.currentShift.participants || [];
        this.currentShift.participants.push(data);

        this.close();
        
        // Notify parent component with updated shift data
        if (this.onSuccess) {
            this.onSuccess(this.currentShift);
        }
        
        return true;
    }
} 