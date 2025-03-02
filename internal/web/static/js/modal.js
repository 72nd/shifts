export class Modal {
    constructor(modalId) {
        this.modalId = modalId;
        this.errors = {};
        this.hasError = false;
        this.errorTitle = '';
        this.errorDetails = {};
        this.showTechnicalDetails = false;
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
            this.clearError();
        }
    }

    setError(field, message) {
        this.errors[field] = message;
    }

    hasErrors() {
        return Object.values(this.errors).some(error => error !== null);
    }

    /**
     * Sets a server error with technical details
     * @param {string} title - User-friendly error title
     * @param {object} details - Technical error details
     */
    setServerError(title, details) {
        this.hasError = true;
        this.errorTitle = title;
        
        // Create a comprehensive error details object
        this.errorDetails = {
            error: {
                title: title,
                message: details.message || 'Unknown error',
                type: details.type || 'server_error',
                timestamp: new Date().toISOString()
            },
            request: details.request || {},
            response: {
                status: details.status,
                statusText: details.statusText,
                body: details.body
            },
            stack: details.stack,
            context: {
                url: window.location.href,
                modalId: this.modalId,
                userAgent: navigator.userAgent,
                appVersion: navigator.appVersion,
                platform: navigator.platform,
                language: navigator.language,
                viewportSize: {
                    width: window.innerWidth,
                    height: window.innerHeight
                },
                timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone,
                dateTime: {
                    local: new Date().toString(),
                    utc: new Date().toUTCString()
                }
            }
        };
        
        this.showTechnicalDetails = false;
    }

    /**
     * Clears the server error
     */
    clearError() {
        this.hasError = false;
        this.errorTitle = '';
        this.errorDetails = {};
        this.showTechnicalDetails = false;
    }

    /**
     * Copies error details to clipboard
     */
    copyErrorDetails() {
        const errorText = JSON.stringify(this.errorDetails, null, 2);
        const copyButton = document.querySelector(`#${this.modalId} .error-copy-button`);
        const originalText = copyButton ? copyButton.textContent : '';
        
        navigator.clipboard.writeText(errorText)
            .then(() => {
                // Show a temporary success message in the button
                if (copyButton) {
                    copyButton.textContent = AlpineI18n.t('common.copied_to_clipboard');
                    setTimeout(() => {
                        copyButton.textContent = originalText;
                    }, 1500);
                }
            })
            .catch(err => {
                console.error('Failed to copy error details:', err);
            });
    }

    /**
     * Handles a server response error
     * @param {Response} response - The fetch response object
     * @param {string} defaultErrorMessage - Default error message to show
     * @param {object} requestInfo - Information about the request that caused the error
     * @returns {Promise<object>} - Error details object
     */
    async handleResponseError(response, defaultErrorMessage, requestInfo = {}) {
        let errorDetails = {
            status: response.status,
            statusText: response.statusText,
            request: requestInfo
        };

        try {
            // Try to read the response body
            const text = await response.text();
            try {
                // Try to parse as JSON
                errorDetails.body = JSON.parse(text);
            } catch (e) {
                // If not JSON, store as text
                errorDetails.body = text;
            }
        } catch (e) {
            errorDetails.body = 'Could not read response body';
            errorDetails.readError = e.message;
        }

        // Set the error with technical details
        this.setServerError(defaultErrorMessage, errorDetails);
        
        return errorDetails;
    }

    /**
     * Sends a request to the server with error handling
     * @param {string} url - The URL to send the request to
     * @param {object} options - Fetch options
     * @param {string} errorMessage - Error message to display on failure
     * @returns {Promise<object|null>} - Response data or null on error
     */
    async sendRequest(url, options, errorMessage) {
        const requestInfo = {
            url,
            method: options.method || 'GET',
            headers: options.headers || {},
            timestamp: new Date().toISOString()
        };
        
        if (options.body) {
            try {
                // Store the request body for debugging
                requestInfo.body = typeof options.body === 'string' 
                    ? JSON.parse(options.body) 
                    : options.body;
            } catch (e) {
                requestInfo.body = options.body;
            }
        }
        
        try {
            const response = await fetch(url, options);

            if (!response.ok) {
                await this.handleResponseError(response, errorMessage, requestInfo);
                return null;
            }

            // If the response is successful but empty (204 No Content)
            if (response.status === 204) {
                return {};
            }

            // Try to parse the response as JSON
            try {
                return await response.json();
            } catch (e) {
                // If the response is not JSON, return an empty object
                return {};
            }
        } catch (error) {
            // Network error or other exception
            this.setServerError(errorMessage, {
                message: error.message,
                stack: error.stack,
                request: requestInfo,
                type: 'network_error'
            });
            return null;
        }
    }
}