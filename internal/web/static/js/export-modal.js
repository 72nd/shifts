import { Modal } from './modal.js';

export class ExportModal extends Modal {
    constructor(planToken, adminToken, filterManager) {
        super('export_modal');
        this.planToken = planToken;
        this.adminToken = adminToken;
        this.filterManager = filterManager;
        this.reset();
    }

    reset() {
        super.clearError();
        this.exportFormat = 'json';
        this.isExporting = false;
        this.useFilters = false;
    }

    open() {
        this.reset();
        super.open();
    }

    activeFilterCount() {
        if (!this.filterManager) return 0;
        return this.filterManager.getActiveFilterCount();
    }

    async downloadExport() {
        try {
            this.isExporting = true;
            super.clearError();

            // Build URL with format parameter
            let url = `/api/plans/${this.planToken}/export?format=${this.exportFormat}`;

            // Add filter parameters if useFilters is enabled
            if (this.useFilters && this.filterManager) {
                const filterParams = this.buildFilterParams();
                if (filterParams) {
                    url += '&' + filterParams;
                }
            }

            const options = {
                method: 'GET',
                headers: {
                    'X-Admin-Token': this.adminToken || ''
                }
            };

            const response = await fetch(url, options);
            
            if (!response.ok) {
                const errorMessage = AlpineI18n.t('shift_plans.view.export_modal.error');
                await this.handleResponseError(response, errorMessage, {
                    url,
                    method: options.method,
                    headers: options.headers
                });
                return false;
            }

            // Get the filename from the Content-Disposition header or use a default
            let filename = 'shift-plan.json';
            const contentDisposition = response.headers.get('Content-Disposition');
            if (contentDisposition) {
                const matches = /filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/.exec(contentDisposition);
                if (matches != null && matches[1]) {
                    filename = matches[1].replace(/['"]/g, '');
                }
            }

            // Download the file
            const blob = await response.blob();
            const url2 = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url2;
            a.download = filename;
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url2);
            document.body.removeChild(a);

            this.close();
            return true;
        } catch (error) {
            console.error('Export error:', error);
            this.setServerError(
                AlpineI18n.t('shift_plans.view.export_modal.error'),
                {
                    message: error.message,
                    stack: error.stack,
                    type: 'export_error'
                }
            );
            return false;
        } finally {
            this.isExporting = false;
        }
    }

    buildFilterParams() {
        if (!this.filterManager) return '';

        const params = new URLSearchParams();

        // Add type IDs filter
        if (this.filterManager.selectedTypes && this.filterManager.selectedTypes.length > 0) {
            params.append('type_ids', this.filterManager.selectedTypes.join(','));
        }

        // Add day filter
        if (this.filterManager.selectedDay) {
            params.append('day', this.filterManager.selectedDay);
        }

        // Add status filter
        if (this.filterManager.selectedStatus && this.filterManager.selectedStatus !== 'all') {
            params.append('status', this.filterManager.selectedStatus);
        }

        // Add search query
        if (this.filterManager.searchQuery && this.filterManager.searchQuery.trim()) {
            params.append('search', this.filterManager.searchQuery.trim());
        }

        return params.toString();
    }
} 