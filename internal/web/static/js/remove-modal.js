import { Modal } from './modal.js';

export class RemoveModal extends Modal {
    constructor(planToken, onSuccess) {
        super('remove_modal');
        this.planToken = planToken;
        this.onSuccess = onSuccess;
        this.reset();
    }

    reset() {
        super.clearError();
        this.currentShift = null;
        this.currentParticipant = null;
    }

    openWith(shift, participant) {
        if (!shift || !participant) return;
        this.reset();
        this.currentShift = shift;
        this.currentParticipant = participant;
        this.open();
    }

    async submit() {
        const url = `/api/plans/${this.planToken}/shifts/${this.currentShift.id}/participants/${this.currentParticipant.id}`;
        const options = {
            method: 'DELETE'
        };
        
        const errorMessage = AlpineI18n.t('shift_plans.view.error_removing');
        
        const result = await this.sendRequest(url, options, errorMessage);
        
        if (result !== null) {
            // Update the current shift's participants
            this.currentShift.participants = this.currentShift.participants.filter(
                p => p.id !== this.currentParticipant.id
            );

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