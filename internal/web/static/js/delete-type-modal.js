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
        super.clearError();
        this.currentType = null;
    }

    openWith(type) {
        this.reset();
        this.currentType = type;
        this.open();
    }

    async submit() {
        const url = `/api/plans/${this.planToken}/types/${this.currentType.id}`;
        const options = {
            method: 'DELETE',
            headers: {
                'X-Admin-Token': this.adminToken
            }
        };
        
        const errorMessage = AlpineI18n.t('shift_plans.view.error_deleting_shift_type');
        
        const result = await this.sendRequest(url, options, errorMessage);
        
        if (result !== null) {
            this.close();
            
            // Notify parent component about the deletion
            if (this.onSuccess) {
                this.onSuccess(this.currentType);
            }
            
            return true;
        }
        
        return false;
    }
} 