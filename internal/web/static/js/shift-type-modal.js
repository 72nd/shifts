import { Modal } from './modal.js';

export class ShiftTypeModal extends Modal {
    constructor(isMultilingual, planToken, adminToken, onSuccess) {
        super('shift_type_modal');
        this.isMultilingual = isMultilingual;
        this.planToken = planToken;
        this.adminToken = adminToken;
        this.onSuccess = onSuccess;
        this.reset();
    }

    reset() {
        this.currentType = null;
        this.title_de = '';
        this.title_en = '';
        this.description_de = '';
        this.description_en = '';
        this.location_de = '';
        this.location_en = '';
        this.clearError();
    }

    openWith(type = null) {
        this.reset();
        this.currentType = type;

        if (type) {
            // Edit mode: Fill form with existing data
            this.title_de = type.title_de || '';
            this.title_en = type.title_en || '';
            this.description_de = type.description_de || '';
            this.description_en = type.description_en || '';
            this.location_de = type.location_de || '';
            this.location_en = type.location_en || '';
        }

        this.open();
    }

    validate() {
        this.errors = {};
        let isValid = true;

        // For multilingual plans, validate both languages
        if (this.isMultilingual) {
            if (!this.title_de?.trim()) {
                this.errors.title_de = AlpineI18n.t('shift_plans.view.error_title_required');
                isValid = false;
            } else if (this.title_de.trim().length > 150) {
                this.errors.title_de = AlpineI18n.t('shift_plans.view.error_title_too_long', { max: 150 });
                isValid = false;
            }
            // Beschreibung und Ort sind jetzt optional
        }

        // Always validate English (as it's the default language)
        if (!this.title_en?.trim()) {
            this.errors.title_en = AlpineI18n.t('shift_plans.view.error_title_required');
            isValid = false;
        } else if (this.title_en.trim().length > 150) {
            this.errors.title_en = AlpineI18n.t('shift_plans.view.error_title_too_long', { max: 150 });
            isValid = false;
        }
        // Beschreibung und Ort sind jetzt optional

        return isValid;
    }

    async submit() {
        if (!this.validate()) return false;

        try {
            const url = this.currentType
                ? `/api/plans/${this.planToken}/types/${this.currentType.id}`
                : `/api/plans/${this.planToken}/types`;

            const options = {
                method: this.currentType ? 'PUT' : 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Admin-Token': this.adminToken
                },
                body: JSON.stringify({
                    title_de: this.title_de,
                    title_en: this.title_en,
                    description_de: this.description_de,
                    description_en: this.description_en,
                    location_de: this.location_de,
                    location_en: this.location_en
                })
            };

            const errorMessage = this.currentType
                ? AlpineI18n.t('shift_plans.view.error_updating_shift_type')
                : AlpineI18n.t('shift_plans.view.error_creating_shift_type');

            const result = await this.sendRequest(url, options, errorMessage);
            
            if (!result) {
                return false;
            }

            // For updates, we use the current data as the server doesn't return it
            const shiftType = this.currentType ? {
                ...this.currentType,
                title_de: this.title_de,
                title_en: this.title_en,
                description_de: this.description_de,
                description_en: this.description_en,
                location_de: this.location_de,
                location_en: this.location_en
            } : result;

            this.close();
            
            // Notify parent component
            if (this.onSuccess) {
                this.onSuccess(shiftType);
            }
            
            return true;
        } catch (error) {
            // This should only happen for network errors, as API errors are handled by sendRequest
            this.setServerError(
                AlpineI18n.t(this.currentType
                    ? 'shift_plans.view.error_updating_shift_type'
                    : 'shift_plans.view.error_creating_shift_type'),
                {
                    message: error.message,
                    stack: error.stack
                }
            );
            return false;
        }
    }
} 