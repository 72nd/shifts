export class Modal {
    constructor(id) {
        this.id = id;
        this.isOpen = false;
        this.errors = {};
        this.technicalError = null;
        this.userErrorMessage = '';
        this.copySuccess = false;
    }

    reset() {
        this.errors = {};
        this.technicalError = null;
        this.userErrorMessage = '';
        this.copySuccess = false;
    }

    open() {
        const modal = document.getElementById(this.id);
        if (modal) {
            modal.showModal();
            this.isOpen = true;
        }
    }

    close() {
        const modal = document.getElementById(this.id);
        if (modal) {
            modal.close();
            this.isOpen = false;
            this.reset();
        }
    }

    async showTechnicalError(error, response = null) {
        let errorDetails = {
            message: error.message,
            timestamp: new Date().toISOString(),
            modalId: this.id,
            modalState: this.getModalState()
        };

        if (response) {
            try {
                const errorText = await response.text();
                errorDetails = {
                    ...errorDetails,
                    status: response.status,
                    statusText: response.statusText,
                    serverResponse: errorText
                };
            } catch (e) {
                console.error('Failed to read error response:', e);
            }
        }

        this.technicalError = errorDetails;
        this.userErrorMessage = this.getUserFriendlyError(error);
        
        // Log the error for debugging
        console.error('Technical error occurred:', errorDetails);
    }

    getUserFriendlyError(error) {
        // Try to get a translated error message based on the error code/message
        const errorKey = error.code || 'unknown';
        return AlpineI18n.t(`errors.${this.id}.${errorKey}`) || 
               AlpineI18n.t('errors.common.unknown');
    }

    getModalState() {
        // Get relevant state information for error context
        // Child classes should override this if needed
        return {};
    }

    formatError(error) {
        if (!error) return '';
        try {
            return JSON.stringify(error, null, 2);
        } catch (e) {
            return String(error);
        }
    }

    async copyErrorDetails() {
        if (!this.technicalError) return;
        
        try {
            const errorText = this.formatError(this.technicalError);
            await navigator.clipboard.writeText(errorText);
            
            // Show success feedback
            this.copySuccess = true;
            setTimeout(() => {
                this.copySuccess = false;
            }, 2000);
        } catch (e) {
            console.error('Failed to copy error details:', e);
            this.errors.copy = AlpineI18n.t('errors.common.copy_failed');
        }
    }

    clearTechnicalError() {
        this.technicalError = null;
        this.userErrorMessage = '';
    }
}