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
        // Coordination settings (see the "Coordination" section of the dialog)
        this.load_factor = '1';
        this.popularity = 'auto';
        this.exp_min_count = 0;
        this.exp_min_level = '';
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
            this.load_factor = String(type.load_factor ?? 1);
            this.popularity = type.popularity || 'auto';
            this.exp_min_count = type.exp_min_count || 0;
            this.exp_min_level = type.exp_min_level || '';
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

        const factor = this.parsedLoadFactor();
        if (!(factor >= 0.1 && factor <= 5)) {
            this.errors.load_factor = AlpineI18n.t('coord.error_load_factor');
            isValid = false;
        }

        return isValid;
    }

    /** Load factor as a number (comma or point), NaN if not a number. */
    parsedLoadFactor() {
        return Number(String(this.load_factor).trim().replace(',', '.'));
    }

    /** Short text for the collapsed section: only what differs from the defaults. */
    coordSummary() {
        const parts = [];
        const factor = this.parsedLoadFactor();
        if (factor !== 1 && !Number.isNaN(factor)) parts.push(`${AlpineI18n.t('coord.load_factor')} ${factor}`);
        if (this.popularity !== 'auto') parts.push(AlpineI18n.t(`coord.popularity_${this.popularity}`));
        if (Number(this.exp_min_count) > 0) parts.push(AlpineI18n.t('coord.exp_need'));
        return parts.join(', ');
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
                    location_en: this.location_en,
                    load_factor: this.parsedLoadFactor(),
                    popularity: this.popularity,
                    exp_min_count: Number(this.exp_min_count) || 0,
                    exp_min_level: this.exp_min_level || ''
                })
            };

            const errorMessage = this.currentType
                ? AlpineI18n.t('shift_plans.view.error_updating_shift_type')
                : AlpineI18n.t('shift_plans.view.error_creating_shift_type');

            const { data } = await this.sendRequest(url, options, errorMessage);
            
            if (!data) {
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
                location_en: this.location_en,
                load_factor: this.parsedLoadFactor(),
                popularity: this.popularity,
                exp_min_count: Number(this.exp_min_count) || 0,
                exp_min_level: this.exp_min_level || null
            } : data;

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