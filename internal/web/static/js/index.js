document.addEventListener('alpine:init', () => {
    Alpine.data('indexPage', () => ({
        planToken: '',

        async createNewPlan() {
            window.location.href = '/shift-plans/new';
        },

        async joinPlan() {
            if (!this.planToken) {
                // TODO: Show error message to user
                return;
            }

            try {
                const response = await fetch(`/api/shift-plans/${this.planToken}`);
                if (!response.ok) {
                    throw new Error('Plan not found');
                }

                window.location.href = `/shift-plans/${this.planToken}`;
            } catch (error) {
                console.error('Error joining plan:', error);
                // TODO: Show error message to user
            }
        }
    }));
}); 