import { Modal } from './modal.js';

export class AdminModal extends Modal {
    constructor(planToken, onSuccess) {
        super('admin_modal');
        this.planToken = planToken;
        this.onSuccess = onSuccess;
        this.reset();
    }

    reset() {
        this.password = '';
    }

    validate() {
        this.errors = {};
        let isValid = true;

        if (!this.password?.trim()) {
            this.errors.password = AlpineI18n.t('shift_plans.view.error_password_required');
            isValid = false;
        }

        return isValid;
    }

    async submit() {
        if (!this.validate()) return false;

        try {
            const response = await fetch(`/api/shift-plans/${this.planToken}/verify-admin`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ password: this.password.trim() })
            });

            if (!response.ok) {
                if (response.status === 401) {
                    this.errors.password = AlpineI18n.t('shift_plans.view.error_invalid_password');
                    return false;
                }
                throw new Error('Failed to verify admin password');
            }

            // First try to get the response as text
            const responseText = await response.text();
            
            // If the response is empty, we might still be successful but without data
            if (!responseText) {
                this.close();
                // Use the password as token if no token is provided
                if (this.onSuccess) {
                    this.onSuccess(this.password.trim());
                }
                return true;
            }

            // Try to parse as JSON if we have content
            try {
                const data = JSON.parse(responseText);
                this.close();
                if (this.onSuccess) {
                    // Use the token from response if available, otherwise use password
                    this.onSuccess(data.token || this.password.trim());
                }
                return true;
            } catch (e) {
                // If parsing fails but response was ok, we still consider it a success
                this.close();
                if (this.onSuccess) {
                    this.onSuccess(this.password.trim());
                }
                return true;
            }
        } catch (error) {
            console.error('Error verifying admin password:', error);
            this.errors.general = error.message;
            return false;
        }
    }
} 