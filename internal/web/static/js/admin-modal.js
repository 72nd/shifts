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
            this.errors.password = AlpineI18n.t('shift_plans.view.admin_password_required');
            isValid = false;
        }

        return isValid;
    }

    async submit() {
        if (!this.validate()) return false;

        const url = `/api/plans/${this.planToken}/admin`;
        const options = {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                password: this.password
            })
        };
        
        const errorMessage = AlpineI18n.t('shift_plans.view.admin_login_error');
        
        const result = await this.sendRequest(url, options, errorMessage);
        
        if (result !== null && result.token) {
            // Debug-Ausgabe für Fehlersuche
            console.log('Admin login successful, received token:', result.token ? '[REDACTED]' : 'empty');
            
            // Close modal and notify parent about success
            this.close();
            if (this.onSuccess) {
                this.onSuccess(result.token);
            }
            return true;
        }
        
        return false;
    }
} 