import { Modal } from './modal.js';

export class SettingsModal extends Modal {
    constructor(planToken, adminToken, onSuccess) {
        super('settings_modal');
        this.planToken = planToken;
        this.adminToken = adminToken;
        this.onSuccess = onSuccess;
        this.reset();
    }

    reset() {
        super.clearError();
        this.is_multilingual = false;
        this.collaborative_mode = false;
        this.hide_declined = true; // shown as "hide not selected sign-ups"; the API stores show_declined
        this.coordination = {}; // coordination settings of the plan, sent back unchanged
        this.title_de = '';
        this.title_en = '';
        this.description_de = '';
        this.description_en = '';
        this.start_date = '';
        this.end_date = '';
        this.day_boundary_time = '00:00'; // Default to midnight
        this.errors = {}; // Field-level validation errors
        this.showDeleteConfirm = false;
        this.deletePassword = '';
        this.deleteError = '';
    }

    openWith(plan) {
        this.reset();
        
        // Fill form with current plan data
        this.is_multilingual = plan.is_multilingual;
        this.collaborative_mode = plan.collaborative_mode || false;
        this.hide_declined = !plan.show_declined;
        this.coordination = {
            experience_field_id: plan.experience_field_id ?? null,
            attendance_mode: plan.attendance_mode || 'off',
            default_target_minutes: plan.default_target_minutes ?? null,
            min_rest_minutes: plan.min_rest_minutes ?? null,
            max_day_minutes: plan.max_day_minutes ?? null
        };
        this.title_de = plan.title_de || '';
        this.title_en = plan.title_en || '';
        this.description_de = plan.description_de || '';
        this.description_en = plan.description_en || '';
        this.day_boundary_time = plan.day_boundary_time || '00:00';
        
        // Plan dates are plain YYYY-MM-DD strings, usable as input values as they are
        this.start_date = plan.start_date || '';
        this.end_date = plan.end_date || '';

        this.open();
    }

    validate() {
        this.errors = {};
        let isValid = true;

        // Validate required fields
        if (!this.title_en) {
            this.errors.title_en = AlpineI18n.t('shift_plans.new.error_title_required');
            isValid = false;
        } else if (this.title_en.trim().length > 150) {
            this.errors.title_en = AlpineI18n.t('shift_plans.new.error_title_too_long', { max: 150 });
            isValid = false;
        }

        if (this.is_multilingual && !this.title_de) {
            this.errors.title_de = AlpineI18n.t('shift_plans.new.error_title_required');
            isValid = false;
        } else if (this.is_multilingual && this.title_de.trim().length > 150) {
            this.errors.title_de = AlpineI18n.t('shift_plans.new.error_title_too_long', { max: 150 });
            isValid = false;
        }

        // Beschreibungen sind jetzt optional

        if (!this.start_date) {
            this.errors.start_date = AlpineI18n.t('shift_plans.new.error_start_date_required');
            isValid = false;
        }

        if (!this.end_date) {
            this.errors.end_date = AlpineI18n.t('shift_plans.new.error_end_date_required');
            isValid = false;
        }

        // Validate date range
        if (this.start_date && this.end_date) {
            // YYYY-MM-DD strings compare correctly as strings
            if (this.end_date < this.start_date) {
                this.errors.end_date = AlpineI18n.t('shift_plans.new.error_end_date_before_start');
                isValid = false;
            }
        }

        // Validate time format
        if (!this.day_boundary_time || !/^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$/.test(this.day_boundary_time)) {
            this.errors.day_boundary_time = AlpineI18n.t('shift_plans.settings.error_invalid_time');
            isValid = false;
        }

        return isValid;
    }

    async submit() {
        // Bei Validierungsfehlern zeigen wir nur die Feldfehlermeldungen an
        // und aktivieren nicht das Error-Display
        if (!this.validate()) return false;

        const url = `/api/plans/${this.planToken}`;
        const options = {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                'X-Admin-Token': this.adminToken || '' // Stelle sicher, dass adminToken nicht undefined ist
            },
            body: JSON.stringify({
                is_multilingual: this.is_multilingual,
                collaborative_mode: this.collaborative_mode,
                title_de: this.is_multilingual ? this.title_de : null,
                title_en: this.title_en,
                description_de: this.is_multilingual ? this.description_de : null,
                description_en: this.description_en,
                start_date: this.start_date,
                end_date: this.end_date,
                day_boundary_time: this.day_boundary_time,
                ...this.coordination,
                show_declined: !this.hide_declined
            })
        };
        
        // Debug-Ausgabe für Fehlersuche
        console.log('Sending settings update with options:', {
            url,
            method: options.method,
            headers: { ...options.headers, 'X-Admin-Token': this.adminToken ? '[REDACTED]' : 'empty' },
            body: JSON.parse(options.body)
        });
        
        const errorMessage = AlpineI18n.t('shift_plans.view.error_updating_settings');
        
        // Verwende die aktualisierte sendRequest-Methode
        const { data } = await this.sendRequest(url, options, errorMessage);
        
        if (data !== null) {
            // Close modal and notify parent about success
            this.close();
            if (this.onSuccess) {
                this.onSuccess(data);
            }
            return true;
        }
        
        return false;
    }

    cancelDelete() {
        this.showDeleteConfirm = false;
        this.deletePassword = '';
        this.deleteError = '';
    }

    async confirmDelete() {
        this.deleteError = '';

        // Verify the password matches the admin token
        if (this.deletePassword !== this.adminToken) {
            this.deleteError = AlpineI18n.t('shift_plans.settings.delete_wrong_password');
            return;
        }

        const url = `/api/plans/${this.planToken}`;
        const options = {
            method: 'DELETE',
            headers: {
                'X-Admin-Token': this.adminToken
            }
        };

        const errorMessage = AlpineI18n.t('shift_plans.settings.delete_error');
        console.log('Sending delete request...', { url, options: { ...options, headers: { 'X-Admin-Token': '[REDACTED]' } } });
        
        const response = await this.sendRequest(url, options, errorMessage);
        console.log('Delete response:', response);

        // Check for successful deletion (HTTP 204 No Content)
        if (response.status === 204) {
            console.log('Delete successful, redirecting to confirmation page...');
            // Close modal and redirect to deletion confirmation page
            this.close();
            // Ensure the redirection happens after a small delay to allow the modal to close
            setTimeout(() => {
                window.location.href = '/plan-deleted';
            }, 100);
        }
    }
} 