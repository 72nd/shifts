import { Modal } from './modal.js';

export class AdminModal extends Modal {
    constructor(planToken, onSuccess) {
        super('admin_modal');
        this.planToken = planToken;
        this.onSuccess = onSuccess;
        this.reset();
    }

    reset() {
        super.clearError();
        this.password = '';
        this.errors = {}; // Field-level validation errors
    }

    validate() {
        this.errors = {};
        let isValid = true;

        if (!this.password) {
            this.errors.password = AlpineI18n.t('shift_plans.view.error_password_required');
            isValid = false;
        }

        return isValid;
    }

    async submit() {
        if (!this.validate()) return false;

        const url = `/api/plans/${this.planToken}/verify-admin`;
        const options = {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                password: this.password
            })
        };
        
        const errorMessage = AlpineI18n.t('shift_plans.view.error_invalid_password');
        
        // Verwende die sendRequest-Methode ohne automatische Fehlerbehandlung
        const { data, status, response } = await this.sendRequest(url, options, errorMessage, false);
        
        // Erfolgreicher Login
        if (data !== null && data.token) {
            // Debug-Ausgabe für Fehlersuche
            console.log('Admin login successful, received token:', data.token ? '[REDACTED]' : 'empty');
            
            // Close modal and notify parent about success
            this.close();
            if (this.onSuccess) {
                this.onSuccess(data.token);
            }
            return true;
        }
        
        // Fehlerbehandlung
        if (status === 401) {
            // Falsches Passwort - benutzerfreundliche Fehlermeldung
            this.setUserFriendlyError(
                AlpineI18n.t('shift_plans.view.error_invalid_password_title'),
                AlpineI18n.t('shift_plans.view.error_invalid_password_message')
            );
            return false;
        }
        
        // Für alle anderen Fehler verwenden wir die Standard-Fehlerbehandlung
        if (response) {
            const requestInfo = {
                url,
                method: options.method,
                headers: options.headers,
                body: { password: '[REDACTED]' }, // Passwort nicht im Log anzeigen
                timestamp: new Date().toISOString()
            };
            await this.handleResponseError(response, errorMessage, requestInfo);
        }
        
        return false;
    }
} 