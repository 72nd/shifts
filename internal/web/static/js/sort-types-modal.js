import { Modal } from './modal.js';

export class SortTypesModal extends Modal {
    constructor(planToken, adminToken, onSuccess) {
        super('sort_types_modal');
        this.planToken = planToken;
        this.adminToken = adminToken;
        this.onSuccess = onSuccess;
        this.reset();
    }

    reset() {
        this.errors = {};
        this.localTypes = []; // Local copy for managing changes within the modal
        this.originalTypes = []; // Backup of original order
    }

    openWith(types) {
        this.reset();
        // Make copies to manage local state
        this.localTypes = [...types];
        this.originalTypes = [...types];
        this.open();
    }

    handleSort(item, position) {
        let types = this.sortTypesModal.localTypes;

        // Get the type ID from the moved item
        const typeId = parseInt(item);
        if (!typeId) {
            console.warn('Invalid type ID:', item);
            return;
        }

        // Find the type object that was moved
        const type = types.find(t => t.id === typeId);
        if (!type) {
            console.warn('Type not found:', typeId);
            return;
        }

        // Create a new array with the updated order
        const newTypes = [...types];
        // First remove the item from its current position
        const currentIndex = newTypes.findIndex(t => t.id === typeId);
        const [removed] = newTypes.splice(currentIndex, 1);
        // Then insert it at the new position
        newTypes.splice(position, 0, removed);

        // Update our local types array with the new order
        this.sortTypesModal.localTypes = newTypes;
    }

    async submit() {
        try {
            const response = await fetch(`/api/plans/${this.planToken}/types/order`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Admin-Token': this.adminToken
                },
                body: JSON.stringify({
                    type_ids: this.localTypes.map(t => t.id)
                })
            });

            if (!response.ok) {
                throw new Error('Failed to update type order');
            }

            // Close modal and notify parent to update with new order
            this.close();
            if (this.onSuccess) {
                this.onSuccess(this.localTypes);
            }
            
            return true;
        } catch (error) {
            console.error('Error updating type order:', error);
            this.errors.general = AlpineI18n.t('shift_plans.view.error_updating_type_order');
            return false;
        }
    }

    close() {
        // Restore original order if not saved
        this.localTypes = [...this.originalTypes];
        super.close();
    }
} 