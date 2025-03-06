export class Modal {
    constructor(modalId) {
        this.modalId = modalId;
        this.errors = {};
        this.hasError = false;
        this.errorTitle = '';
        this.errorDetails = {};
        this.showTechnicalDetails = false;
        
        // Eigenschaften für benutzerfreundliche Fehler
        this.hasUserFriendlyError = false;
        this.userFriendlyErrorTitle = '';
        this.userFriendlyErrorMessage = '';
    }

    open() {
        const modal = document.getElementById(this.modalId);
        if (modal) {
            modal.showModal();
        }
    }

    close() {
        const modal = document.getElementById(this.modalId);
        if (modal) {
            modal.close();
        }
    }

    setError(field, message) {
        this.errors[field] = message;
    }

    hasErrors() {
        return Object.keys(this.errors).length > 0;
    }

    /**
     * Sets a user-friendly error without technical details
     * @param {string} title - Error title
     * @param {string} message - Error message
     */
    setUserFriendlyError(title, message) {
        // Benutzerfreundliche Fehleranzeige aktivieren
        this.hasUserFriendlyError = true;
        this.userFriendlyErrorTitle = title;
        this.userFriendlyErrorMessage = message;
        
        // Technische Fehleranzeige deaktivieren
        this.hasError = false;
    }

    /**
     * Sets a server error with technical details
     * @param {string} title - User-friendly error title
     * @param {object} details - Technical error details
     */
    setServerError(title, details) {
        // Technische Fehleranzeige aktivieren
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
        
        // Benutzerfreundliche Fehleranzeige deaktivieren
        this.hasUserFriendlyError = false;
    }

    /**
     * Clears all errors (both technical and user-friendly)
     */
    clearError() {
        this.errors = {};
        
        // Technische Fehler zurücksetzen
        this.hasError = false;
        this.errorTitle = '';
        this.errorDetails = {};
        this.showTechnicalDetails = false;
        
        // Benutzerfreundliche Fehler zurücksetzen
        this.hasUserFriendlyError = false;
        this.userFriendlyErrorTitle = '';
        this.userFriendlyErrorMessage = '';
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
     * Handles an error response from a fetch request
     * @param {Response} response - The fetch response object
     * @param {string} defaultErrorMessage - Default error message to show
     * @param {Object} requestInfo - Information about the request
     * @returns {Promise<void>}
     */
    async handleResponseError(response, defaultErrorMessage, requestInfo = {}) {
        let errorDetails = {
            message: defaultErrorMessage,
            status: response.status,
            statusText: response.statusText,
            request: requestInfo
        };

        try {
            // Try to parse the response body
            const contentType = response.headers.get('content-type');
            if (contentType && contentType.includes('application/json')) {
                const errorJson = await response.json();
                errorDetails.body = errorJson;
                
                // Use the server-provided error message if available
                if (errorJson.error && errorJson.error.message) {
                    errorDetails.message = errorJson.error.message;
                }
                
                // Use the server-provided error title if available
                if (errorJson.error && errorJson.error.title) {
                    errorDetails.title = errorJson.error.title;
                }
            } else {
                // For non-JSON responses, get the text
                errorDetails.body = await response.text();
            }
        } catch (e) {
            // If we can't parse the response, just use what we have
            errorDetails.parseError = e.message;
        }

        // Set the error in the modal
        // Verwende den Titel aus den errorDetails, falls vorhanden, sonst den defaultErrorMessage
        const errorTitle = errorDetails.title || defaultErrorMessage;
        this.setServerError(errorTitle, errorDetails);
    }

    /**
     * Sends a request to the server with error handling
     * @param {string} url - The URL to send the request to
     * @param {object} options - Fetch options
     * @param {string} errorMessage - Error message to display on failure
     * @param {boolean} handleErrors - Whether to automatically handle errors (default: true)
     * @returns {Promise<{data: object|null, status: number|null}>} - Response data and status
     */
    async sendRequest(url, options, errorMessage, handleErrors = true) {
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
            
            // Wenn Fehler nicht automatisch behandelt werden sollen, geben wir den Status zurück
            if (!handleErrors) {
                if (!response.ok) {
                    return { data: null, status: response.status, response };
                }
                
                // If the response is successful but empty (204 No Content)
                if (response.status === 204) {
                    return { data: {}, status: 204 };
                }
                
                // Try to parse the response as JSON
                try {
                    const data = await response.json();
                    return { data, status: response.status };
                } catch (e) {
                    // If the response is not JSON, return an empty object
                    return { data: {}, status: response.status };
                }
            }
            
            // Automatische Fehlerbehandlung (bisheriges Verhalten)
            if (!response.ok) {
                await this.handleResponseError(response, errorMessage, requestInfo);
                return { data: null, status: response.status };
            }

            // If the response is successful but empty (204 No Content)
            if (response.status === 204) {
                return { data: {}, status: 204 };
            }

            // Try to parse the response as JSON
            try {
                const data = await response.json();
                return { data, status: response.status };
            } catch (e) {
                // If the response is not JSON, return an empty object
                return { data: {}, status: response.status };
            }
        } catch (error) {
            // Network error or other exception
            this.setServerError(errorMessage, {
                message: error.message,
                stack: error.stack,
                request: requestInfo,
                type: 'network_error'
            });
            return { data: null, status: null };
        }
    }
}