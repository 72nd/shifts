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
            }
            if (!this.description_de?.trim()) {
                this.errors.description_de = AlpineI18n.t('shift_plans.view.error_description_required');
                isValid = false;
            }
            if (!this.location_de?.trim()) {
                this.errors.location_de = AlpineI18n.t('shift_plans.view.error_location_required');
                isValid = false;
            }
        }

        // Always validate English (as it's the default language)
        if (!this.title_en?.trim()) {
            this.errors.title_en = AlpineI18n.t('shift_plans.view.error_title_required');
            isValid = false;
        }
        if (!this.description_en?.trim()) {
            this.errors.description_en = AlpineI18n.t('shift_plans.view.error_description_required');
            isValid = false;
        }
        if (!this.location_en?.trim()) {
            this.errors.location_en = AlpineI18n.t('shift_plans.view.error_location_required');
            isValid = false;
        }

        return isValid;
    }

    async submit() {
        if (!this.validate()) return false;

        try {
            const url = this.currentType
                ? `/api/shift-plans/${this.planToken}/types/${this.currentType.id}`
                : `/api/shift-plans/${this.planToken}/types`;

            const response = await fetch(url, {
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
            });

            if (!response.ok) {
                throw new Error(this.currentType ? 'Failed to update shift type' : 'Failed to create shift type');
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
            } : await response.json();

            this.close();
            
            // Notify parent component
            if (this.onSuccess) {
                this.onSuccess(shiftType);
            }
            
            return true;
        } catch (error) {
            console.error('Error saving shift type:', error);
            this.errors.title_en = AlpineI18n.t(
                this.currentType
                    ? 'shift_plans.view.error_updating_shift_type'
                    : 'shift_plans.view.error_creating_shift_type'
            );
            return false;
        }
    }
} 