import { Modal } from './modal.js';

export class JoinModal extends Modal {
    constructor(planToken, onSuccess) {
        super('join_modal');
        this.planToken = planToken;
        this.onSuccess = onSuccess;
        this.reset();
    }

    reset() {
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
        if (!this.validate()) return false;

        try {
            const response = await fetch(`/api/shift-plans/${this.planToken}/shifts/${this.currentShift.id}/participants`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: this.name.trim() })
            });

            if (!response.ok) {
                throw new Error('Failed to join shift');
            }

            // Get the updated participant data
            const participant = await response.json();

            // Save name to localStorage after successful join
            localStorage.setItem('last_participant_name', this.name.trim());

            // Update the current shift's participants
            this.currentShift.participants = this.currentShift.participants || [];
            this.currentShift.participants.push(participant);

            this.close();
            
            // Notify parent component with updated shift data
            if (this.onSuccess) {
                this.onSuccess(this.currentShift);
            }
            
            return true;
        } catch (error) {
            console.error('Error joining shift:', error);
            this.errors.name = AlpineI18n.t('shift_plans.view.error_joining');
            return false;
        }
    }
} 