import { fetchWithAuth } from './auth.js';
import { loadShiftTypes } from './shifts.js';
import { t } from './i18n.js';

export function showParticipantsModal(typeId, shiftId) {
    // Find the shift and its participants
    fetch('/shifttypes')
        .then(response => response.json())
        .then(shiftTypes => {
            const shiftType = shiftTypes[typeId];
            const shift = shiftType.shifts.find(s => s.id === shiftId);
            
            if (shift) {
                const participantsList = document.getElementById('participantsList');
                participantsList.innerHTML = `
                    <div class="d-flex justify-content-between mb-3">
                        <div>
                            <strong>Besetzung:</strong> ${shift.participants.length}/${shift.required}
                        </div>
                        <div>
                            <div class="progress" style="width: 100px; height: 8px;">
                                <div class="progress-bar ${shift.participants.length >= shift.required ? 'bg-success' : 'bg-primary'}" 
                                     style="width: ${(shift.participants.length / shift.required) * 100}%">
                                </div>
                            </div>
                        </div>
                    </div>
                    ${shift.participants.length > 0 ? `
                        <ul class="list-group">
                            ${shift.participants.map(p => `
                                <li class="list-group-item d-flex justify-content-between align-items-center">
                                    ${p}
                                    <button class="btn btn-sm btn-outline-danger" 
                                            onclick="removeParticipant('${typeId}', '${shiftId}', '${p}')">
                                        <i class="bi bi-person-x"></i>
                                    </button>
                                </li>
                            `).join('')}
                        </ul>
                    ` : `
                        <p class="text-muted text-center mb-0">
                            Noch keine Teilnehmer eingetragen
                        </p>
                    `}
                `;
                
                new bootstrap.Modal(document.getElementById('participantsModal')).show();
            }
        });
}

export function showJoinModal(typeId, shiftId) {
    const form = document.getElementById('joinShiftForm');
    form.elements.typeId.value = typeId;
    form.elements.shiftId.value = shiftId;
    
    // Check if shift type has password
    fetch('/shifttypes')
        .then(response => response.json())
        .then(shiftTypes => {
            const shiftType = shiftTypes[typeId];
            const passwordField = document.getElementById('passwordField');
            
            if (shiftType.password) {
                passwordField.classList.remove('d-none');
                form.elements.password.required = true;
            } else {
                passwordField.classList.add('d-none');
                form.elements.password.required = false;
                form.elements.password.value = '';
            }
        });
    
    new bootstrap.Modal(document.getElementById('joinShiftModal')).show();
}

export function joinShift() {
    const form = document.getElementById('joinShiftForm');
    const formData = new FormData(form);
    const name = formData.get('name');
    const password = formData.get('password');

    fetchWithAuth(`/shifttypes/${formData.get('typeId')}/shifts/${formData.get('shiftId')}/join`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ name, password }),
    })
    .then(response => {
        if (response.ok) {
            bootstrap.Modal.getInstance(document.getElementById('joinShiftModal')).hide();
            form.reset();
            loadShiftTypes();
        } else if (response.status === 401) {
            showError('error_wrong_password');
        } else {
            showError('error_joining_shift');
        }
    });
}

export function removeParticipant(typeId, shiftId, participantName) {
    // First check if password is required
    fetch('/shifttypes')
        .then(response => response.json())
        .then(shiftTypes => {
            const shiftType = shiftTypes[typeId];
            
            if (shiftType.password) {
                const password = prompt(t('shift_password_hint') + '\n\n' + t('shift_password') + ':');
                if (!password) return;
                
                doRemoveParticipant(typeId, shiftId, participantName, password);
            } else {
                if (confirm(t('confirm_remove_participant', { name: participantName }))) {
                    doRemoveParticipant(typeId, shiftId, participantName);
                }
            }
        });
}

function doRemoveParticipant(typeId, shiftId, participantName, password = null) {
    fetchWithAuth(`/shifttypes/${typeId}/shifts/${shiftId}/leave`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ name: participantName, password }),
    })
    .then(response => {
        if (response.ok) {
            loadShiftTypes();
        } else if (response.status === 401) {
            showError('error_wrong_password');
        } else {
            showError('error_leaving_shift');
        }
    });
}

function showError(key) {
    alert(t(key));
}

// Add functions to window object for event handling
window.joinShift = joinShift;
window.removeParticipant = removeParticipant; 