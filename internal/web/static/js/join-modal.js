import { Modal } from './modal.js';

export class JoinModal extends Modal {
    constructor(planToken, onSuccess, adminToken = '') {
        super('join_modal');
        this.planToken = planToken;
        this.adminToken = adminToken;
        this.onSuccess = onSuccess;
        this.reset();
    }

    reset() {
        super.clearError();
        this.errors = {}; // Field-level validation errors
        this.currentShift = null;
        this.name = localStorage.getItem('last_participant_name') || '';
    }

    openWith(shift) {
        this.reset();
        this.currentShift = shift;
        this.open();
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
        if (!this.validate()) {
            return false;
        }

        const url = `/api/plans/${this.planToken}/shifts/${this.currentShift.id}/participants`;
        const options = {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: this.name.trim() })
        };
        if (this.adminToken) {
            options.headers['X-Admin-Token'] = this.adminToken;
        }

        const errorMessage = AlpineI18n.t('shift_plans.view.error_joining');
        
        // Verwende die sendRequest-Methode aus der Modal-Klasse, aber ohne automatische Fehlerbehandlung
        const { data, status, response } = await this.sendRequest(url, options, errorMessage, false);
        
        // Fehlerbehandlung
        if (data === null) {
            // Spezielle Behandlung für 409 Conflict (Schicht voll)
            if (status === 409) {
                this.setUserFriendlyError(
                    AlpineI18n.t('shift_plans.view.error_shift_full_title'),
                    AlpineI18n.t('shift_plans.view.error_shift_full_message')
                );
                return false;
            }
            
            // Für alle anderen Fehler verwenden wir die Standard-Fehlerbehandlung
            if (response) {
                const requestInfo = {
                    url,
                    method: options.method,
                    headers: options.headers,
                    body: { name: this.name.trim() },
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