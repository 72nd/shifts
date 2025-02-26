import { Modal } from './modal.js';

export class SortTypesModal extends Modal {
    constructor(planToken, adminToken, onSuccess) {
        super('sort_types_modal');
        this.planToken = planToken;
        this.adminToken = adminToken;
        this.onSuccess = onSuccess;
        this.sortable = null;
        this.reset();
    }

    reset() {
        this.types = [];
        this.errors = {};
    }

    openWith(types) {
        this.reset();
        // Create a copy of the types array with only necessary properties
        // Sort by the existing sort_order to maintain the current order
        this.types = [...types]
            .sort((a, b) => a.sort_order - b.sort_order)
            .map(type => ({
                id: type.id,
                title_de: type.title_de,
                title_en: type.title_en
            }));
        this.open();
    }

    initSortable(el) {
        this.sortable = new Sortable(el, {
            animation: 150,
            handle: '.handle',
            onEnd: (evt) => {
                // Convert HTMLCollection to Array and get all items in their new order
                const items = Array.from(evt.to.children);
                const newTypes = [];
                
                // Rebuild the types array in the new order
                for (let item of items) {
                    const typeId = parseInt(item.dataset.id);
                    const type = this.types.find(t => t.id === typeId);
                    if (type) {
                        newTypes.push(type);
                    }
                }
                
                // Update the types array with the new order
                this.types = newTypes;
            }
        });
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
                    type_ids: this.types.map(t => t.id)
                })
            });

            if (!response.ok) {
                throw new Error('Failed to update type order');
            }

            this.close();
            
            // Notify parent component
            if (this.onSuccess) {
                this.onSuccess(this.types);
            }
            
            return true;
        } catch (error) {
            console.error('Error updating type order:', error);
            this.errors.general = AlpineI18n.t('shift_plans.view.error_updating_type_order');
            return false;
        }
    }
} 