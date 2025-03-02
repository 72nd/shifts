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
        this.title_de = '';
        this.title_en = '';
        this.description_de = '';
        this.description_en = '';
        this.start_date = '';
        this.end_date = '';
        this.errors = {}; // Field-level validation errors
    }

    openWith(plan) {
        this.reset();
        
        // Fill form with current plan data
        this.is_multilingual = plan.is_multilingual;
        this.title_de = plan.title_de || '';
        this.title_en = plan.title_en || '';
        this.description_de = plan.description_de || '';
        this.description_en = plan.description_en || '';
        
        // Format dates to YYYY-MM-DD for input fields
        const startDate = new Date(plan.start_date);
        const endDate = new Date(plan.end_date);
        this.start_date = startDate.toISOString().split('T')[0];
        this.end_date = endDate.toISOString().split('T')[0];

        this.open();
    }

    validate() {
        this.errors = {};
        let isValid = true;

        // Validate required fields
        if (!this.title_en) {
            this.errors.title_en = AlpineI18n.t('shift_plans.new.error_title_required');
            isValid = false;
        }

        if (this.is_multilingual && !this.title_de) {
            this.errors.title_de = AlpineI18n.t('shift_plans.new.error_title_required');
            isValid = false;
        }

        if (!this.description_en) {
            this.errors.description_en = AlpineI18n.t('shift_plans.new.error_description_required');
            isValid = false;
        }

        if (this.is_multilingual && !this.description_de) {
            this.errors.description_de = AlpineI18n.t('shift_plans.new.error_description_required');
            isValid = false;
        }

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
            const start = new Date(this.start_date);
            const end = new Date(this.end_date);
            if (end < start) {
                this.errors.end_date = AlpineI18n.t('shift_plans.new.error_end_date_before_start');
                isValid = false;
            }
        }

        return isValid;
    }

    async submit() {
        if (!this.validate()) return false;

        const url = `/api/plans/${this.planToken}`;
        const options = {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                'X-Admin-Token': this.adminToken
            },
            body: JSON.stringify({
                is_multilingual: this.is_multilingual,
                title_de: this.is_multilingual ? this.title_de : null,
                title_en: this.title_en,
                description_de: this.is_multilingual ? this.description_de : null,
                description_en: this.description_en,
                start_date: new Date(this.start_date).toISOString(),
                end_date: new Date(this.end_date).toISOString()
            })
        };
        
        const errorMessage = AlpineI18n.t('shift_plans.view.error_updating_settings');
        
        const result = await this.sendRequest(url, options, errorMessage);
        
        if (result !== null) {
            // Close modal and notify parent about success
            this.close();
            if (this.onSuccess) {
                this.onSuccess(result);
            }
            return true;
        }
        
        return false;
    }
} 