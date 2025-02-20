import { Modal } from './modal.js';

export class DeleteShiftModal extends Modal {
    constructor(planToken, adminToken, onSuccess) {
        super('delete_shift_modal');
        this.planToken = planToken;
        this.adminToken = adminToken;
        this.onSuccess = onSuccess;
        this.reset();
    }

    reset() {
        this.currentShift = null;
        this.currentType = null;
    }

    openWith(shift, type) {
        this.reset();
        this.currentShift = shift;
        this.currentType = type;
        this.open();
    }

    async submit() {
        try {
            const response = await fetch(`/api/plans/${this.planToken}/shifts/${this.currentShift.id}`, {
                method: 'DELETE',
                headers: {
                    'X-Admin-Token': this.adminToken
                }
            });

            if (!response.ok) {
                throw new Error('Failed to delete shift');
            }

            this.close();
            
            // Notify parent component about the deletion
            if (this.onSuccess) {
                this.onSuccess(this.currentShift, this.currentType);
            }
            
            return true;
        } catch (error) {
            console.error('Error deleting shift:', error);
            this.errors.general = AlpineI18n.t('shift_plans.view.error_deleting_shift');
            return false;
        }
    }
} 