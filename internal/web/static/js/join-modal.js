import { Modal } from './modal.js';

export class JoinModal extends Modal {
    constructor(planToken, onSuccess) {
        super('join_modal');
        this.planToken = planToken;
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
        if (!this.validate()) {
            // Wenn Validierungsfehler auftreten, zeigen wir sie auch im Error-Display an
            if (this.errors.name) {
                this.setServerError(
                    this.errors.name,
                    {
                        message: this.errors.name,
                        type: 'validation_error',
                        field: 'name',
                        value: this.name
                    }
                );
            }
            return false;
        }

        const url = `/api/plans/${this.planToken}/shifts/${this.currentShift.id}/participants`;
        const options = {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: this.name.trim() })
        };
        
        const errorMessage = AlpineI18n.t('shift_plans.view.error_joining');
        
        const result = await this.sendRequest(url, options, errorMessage);
        
        if (result !== null) {
            // Save name to localStorage after successful join
            localStorage.setItem('last_participant_name', this.name.trim());

            // Update the current shift's participants
            this.currentShift.participants = this.currentShift.participants || [];
            this.currentShift.participants.push(result);

            this.close();
            
            // Notify parent component with updated shift data
            if (this.onSuccess) {
                this.onSuccess(this.currentShift);
            }
            
            return true;
        }
        
        return false;
    }
} 