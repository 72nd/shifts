export class Modal {
    constructor(modalId) {
        this.modalId = modalId;
        this.errors = {};
    }

    open() {
        const modal = document.getElementById(this.modalId);
        if (modal) modal.showModal();
    }

    close() {
        const modal = document.getElementById(this.modalId);
        if (modal) {
            modal.close();
            this.errors = {};
        }
    }

    setError(field, message) {
        this.errors[field] = message;
    }

    hasErrors() {
        return Object.values(this.errors).some(error => error !== null);
    }
}