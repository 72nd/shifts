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
        super.clearError();
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
        const url = `/api/plans/${this.planToken}/shifts/${this.currentShift.id}`;
        const options = {
            method: 'DELETE',
            headers: {
                'X-Admin-Token': this.adminToken
            }
        };
        
        const errorMessage = AlpineI18n.t('shift_plans.view.error_deleting_shift');
        
        const result = await this.sendRequest(url, options, errorMessage);
        
        if (result !== null) {
            this.close();
            
            // Notify parent component about the deletion
            if (this.onSuccess) {
                this.onSuccess(this.currentShift, this.currentType);
            }
            
            return true;
        }
        
        return false;
    }
} 