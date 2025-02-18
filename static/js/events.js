/**
 * Erstellt einen neuen Schichtplan/Event
 * @param {Object} options - Optionen für die Erstellung
 * @param {boolean} options.showUrlModal - Ob nach der Erstellung das URL-Modal angezeigt werden soll
 * @param {boolean} options.redirectToEvent - Ob nach der Erstellung zur Event-Seite weitergeleitet werden soll
 */
export async function createEvent(options = { showUrlModal: true, redirectToEvent: false }) {
    const formId = options.showUrlModal ? 'newShiftPlanForm' : 'newEventForm';
    const form = document.getElementById(formId);
    const formData = new FormData(form);
    
    const startDate = new Date(formData.get('startDate'));
    const dayZero = new Date(startDate);
    dayZero.setDate(dayZero.getDate() - 1);
    
    const eventData = {
        title: formData.get('title'),
        startDate: formData.get('startDate'),
        endDate: formData.get('endDate'),
        adminPassword: formData.get('adminPassword'),
        dayZero: dayZero.toISOString().split('T')[0],
        debugMode: formData.get('debugMode') === 'on'
    };

    try {
        const response = await fetch('/api/events', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(eventData)
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const result = await response.json();
        
        // Hide the creation modal if it exists
        const modal = bootstrap.Modal.getInstance(document.getElementById(options.showUrlModal ? 'newShiftPlanModal' : 'newEventModal'));
        if (modal) {
            modal.hide();
        }
        
        if (options.showUrlModal) {
            // Show the URL display modal
            const url = `${window.location.origin}/events/${result.id}`;
            document.getElementById('shiftPlanUrl').value = url;
            document.getElementById('goToShiftPlanBtn').href = url;
            new bootstrap.Modal(document.getElementById('urlDisplayModal')).show();
        }
        
        if (options.redirectToEvent) {
            window.location.href = `/events/${result.id}`;
        }
        
        // Reset the form
        form.reset();
    } catch (error) {
        console.error('Error creating event:', error);
        alert('Fehler beim Erstellen des Schichtplans');
    }
}

export async function copyUrl() {
    const urlInput = document.getElementById('shiftPlanUrl');
    try {
        await navigator.clipboard.writeText(urlInput.value);
        
        // Temporär den Button-Text ändern als Feedback
        const button = document.querySelector('#urlDisplayModal button.btn-outline-primary');
        const originalHTML = button.innerHTML;
        button.innerHTML = '<i class="bi bi-check"></i>';
        setTimeout(() => {
            button.innerHTML = originalHTML;
        }, 2000);
    } catch (err) {
        console.error('Failed to copy text: ', err);
        alert('Fehler beim Kopieren der URL');
    }
} 