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

        if (!this.password?.trim()) {
            this.errors.password = AlpineI18n.t('shift_plans.view.error_password_required');
            isValid = false;
        }

        return isValid;
    }

    async submit() {
        if (!this.validate()) {
            // Bei einfachen Validierungsfehlern zeigen wir nur die Feldfehlermeldung an
            // und aktivieren nicht das Error-Display
            return false;
        }

        try {
            const url = `/api/plans/${this.planToken}/verify-admin`;
            const options = {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ password: this.password.trim() })
            };
            
            const errorMessage = AlpineI18n.t('shift_plans.view.error_invalid_password');
            
            // Special handling for 401 errors
            const response = await fetch(url, options);
            
            if (!response.ok) {
                if (response.status === 401) {
                    // Bei ungültigem Passwort zeigen wir die Fehlermeldung direkt beim Feld an
                    this.errors.password = AlpineI18n.t('shift_plans.view.error_invalid_password');
                    return false;
                }
                
                // Für andere Fehler verwenden wir den generischen Error-Handler
                await this.handleResponseError(response, errorMessage);
                return false;
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
            // Handle network errors
            console.error('Network error during admin login:', error);
            this.setServerError(
                AlpineI18n.t('common.error_network'),
                {
                    message: error.message || 'Network error',
                    type: 'network_error'
                }
            );
            return false;
        }
    }
} 