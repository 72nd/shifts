import { Modal } from './modal.js';
import { parseDate, parseDateTime, serializeDate, serializeDateTime, toInputValue, addDays, daysBetween, formatDate, formatTime } from './datetime.js';

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
        // Attendance: a plan day ("YYYY-MM-DD", empty = no limit) and a time ("HH:MM")
        this.fromDay = '';
        this.fromTime = '';
        this.untilDay = '';
        this.untilTime = '';
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

    /** Whether the plan asks for any details at all (otherwise joining is one step as always). Needs coordination. */
    get asksDetails() {
        return !!this.plan?.coordination_enabled && (this.participantFields.length > 0 || this.asksAttendance);
    }

    get locale() {
        return window.AlpineI18n?.locale === 'en' ? 'en' : 'de';
    }

    /** Plan days as "YYYY-MM-DD", plus days already chosen or known that lie outside the plan. */
    get dayOptions() {
        const start = parseDate(this.plan?.start_date);
        const end = parseDate(this.plan?.end_date);
        const days = new Set();
        if (start && end) {
            for (let i = 0, n = daysBetween(start, end); i <= n; i++) days.add(serializeDate(addDays(start, i)));
        }
        for (const d of [this.fromDay, this.untilDay]) if (d) days.add(d);
        return [...days].sort();
    }

    dayLabel(key) {
        return formatDate(key, this.locale, { weekday: 'short', day: 'numeric', month: 'numeric' });
    }

    /** Whether the person is known and has any saved details. */
    get hasKnownDetails() {
        const k = this.known;
        return !!k?.exists && (Object.keys(k.values || {}).length > 0 || !!k.available_from || !!k.available_until);
    }

    /** Segment group for choice fields with up to 4 options, otherwise a select. */
    isSegmented(f) {
        return f.kind === 'choice' && (f.options || []).length <= 4;
    }

    label(f) {
        const de = this.locale !== 'en';
        return (de ? f.label_de : f.label_en) || f.label_en || f.label_de || '';
    }

    optionText(o) {
        const de = this.locale !== 'en';
        return (de ? o.de : o.en) || o.en || o.de || '';
    }

    /** One line with the known details, e.g. "Experienced, from Fri, 25/06 08:00". */
    get knownSummary() {
        if (!this.hasKnownDetails) return '';
        const parts = [];
        for (const f of this.participantFields) {
            const v = this.known.values?.[f.id];
            if (v === undefined || v === null || v === '' || v === false) continue;
            if (f.kind === 'choice') {
                const o = (f.options || []).find(o => o.id === v);
                parts.push(o ? this.optionText(o) : String(v));
            } else if (f.kind === 'boolean') {
                parts.push(this.label(f));
            } else {
                parts.push(String(v));
            }
        }
        if (this.asksAttendance) {
            const when = (str) => {
                const d = parseDateTime(str);
                return d ? `${formatDate(d, this.locale, { weekday: 'short', day: 'numeric', month: 'numeric' })} ${formatTime(d, this.locale)}` : '';
            };
            const from = this.known.available_from ? when(this.known.available_from) : '';
            const until = this.known.available_until ? when(this.known.available_until) : '';
            if (from && until) parts.push(AlpineI18n.t('coord.attendance_range', { from, until }));
            else if (from) parts.push(AlpineI18n.t('coord.attendance_from', { from }));
            else if (until) parts.push(AlpineI18n.t('coord.attendance_until', { until }));
        }
        return parts.join(', ');
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
        if (!this.asksDetails || !name) {
            this.known = null;
            this.knownName = '';
            return;
        }
        if (name === this.knownName) return;
        this.knownName = name;
        this.known = null;
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
        const split = (str) => {
            const d = str ? parseDateTime(str) : null;
            return d ? [serializeDate(d), toInputValue(d).slice(11)] : ['', ''];
        };
        [this.fromDay, this.fromTime] = split(known?.available_from);
        [this.untilDay, this.untilTime] = split(known?.available_until);
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
            const from = this.composeAttendance(this.fromDay, this.fromTime, '00:00');
            const until = this.composeAttendance(this.untilDay, this.untilTime, '23:59');
            if (from) person.available_from = from;
            if (until) person.available_until = until;
        }
        return person;
    }

    /** "YYYY-MM-DDTHH:MM:SS" from a day and a time; empty day means no limit, empty time the given default. */
    composeAttendance(day, time, fallbackTime) {
        if (!day) return '';
        const d = parseDateTime(`${day}T${time || fallbackTime}`);
        return d ? serializeDateTime(d) : '';
    }

    validateDetails() {
        this.errors = {};
        const missing = this.missingRequired(this.values);
        if (missing.length) this.errors.details = AlpineI18n.t('shift_plans.view.error_details_required');
        const from = this.asksAttendance ? this.composeAttendance(this.fromDay, this.fromTime, '00:00') : '';
        const until = this.asksAttendance ? this.composeAttendance(this.untilDay, this.untilTime, '23:59') : '';
        if (from && until && until < from) {
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