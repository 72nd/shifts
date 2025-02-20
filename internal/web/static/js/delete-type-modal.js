import { Modal } from './modal.js';

export class DeleteTypeModal extends Modal {
    constructor(planToken, adminToken, onSuccess) {
        super('delete_shift_type_modal');
        this.planToken = planToken;
        this.adminToken = adminToken;
        this.onSuccess = onSuccess;
        this.reset();
    }

    reset() {
        this.currentType = null;
    }

    openWith(type) {
        this.reset();
        this.currentType = type;
        this.open();
    }

    async submit() {
        try {
            const response = await fetch(`/api/shift-plans/${this.planToken}/types/${this.currentType.id}`, {
                method: 'DELETE',
                headers: {
                    'X-Admin-Token': this.adminToken
                }
            });

            if (!response.ok) {
                throw new Error('Failed to delete shift type');
            }

            this.close();
            
            // Notify parent component about the deletion
            if (this.onSuccess) {
                this.onSuccess(this.currentType);
            }
            
            return true;
        } catch (error) {
            console.error('Error deleting shift type:', error);
            this.errors.general = AlpineI18n.t('shift_plans.view.error_deleting_shift_type');
            return false;
        }
    }
} 