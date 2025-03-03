import { fetchWithAuth } from './auth.js';
import { formatDateTime, formatDate, formatTime, calculateEndTime, groupShiftsByDay } from './utils.js';
import { showJoinModal } from './participants.js';
import { t, currentLanguage } from './i18n.js';

// Add filter function to window object
window.applyFilters = applyFilters;
window.toggleDescription = toggleDescription;

// Listen for language changes
document.addEventListener('languageChanged', () => {
    loadShiftTypes();
});

function applyFilters() {
    const typeFilter = document.getElementById('shiftTypeFilter').value;
    const dateFilter = document.getElementById('timeRangeFilter').value;
    const statusFilter = document.getElementById('statusFilter').value;
    const participantSearch = document.getElementById('participantSearch').value.toLowerCase().trim();

    // Update active filters display
    const activeFilters = [];
    if (typeFilter !== 'all') {
        const typeText = document.querySelector(`#shiftTypeFilter option[value="${typeFilter}"]`).text;
        activeFilters.push(`${t('shift_type_filter')}: ${typeText}`);
    }
    if (dateFilter !== 'all') {
        const dateText = document.querySelector(`#timeRangeFilter option[value="${dateFilter}"]`).text;
        activeFilters.push(`${t('day')}: ${dateText}`);
    }
    if (statusFilter !== 'all') {
        const statusText = document.querySelector(`#statusFilter option[value="${statusFilter}"]`).text;
        activeFilters.push(`${t('status')}: ${statusText}`);
    }
    if (participantSearch) {
        activeFilters.push(`${t('participant_search')}: ${participantSearch}`);
    }

    // Show/hide active filters summary
    const activeFiltersDiv = document.getElementById('activeFilters');
    const resetFiltersBtn = document.getElementById('resetFiltersBtn');
    
    if (activeFilters.length > 0) {
        document.getElementById('activeFiltersText').textContent = activeFilters.join(' | ');
        activeFiltersDiv.style.display = 'block';
        resetFiltersBtn.style.display = 'block';
        
        // Add filter-active class to select elements and search input
        ['shiftTypeFilter', 'timeRangeFilter', 'statusFilter'].forEach(id => {
            const select = document.getElementById(id);
            if (select.value !== 'all') {
                select.classList.add('filter-active');
            } else {
                select.classList.remove('filter-active');
            }
        });
        
        const participantSearchInput = document.getElementById('participantSearch');
        if (participantSearch) {
            participantSearchInput.classList.add('filter-active');
        } else {
            participantSearchInput.classList.remove('filter-active');
        }
    } else {
        activeFiltersDiv.style.display = 'none';
        resetFiltersBtn.style.display = 'none';
        
        // Remove filter-active class from all elements
        ['shiftTypeFilter', 'timeRangeFilter', 'statusFilter'].forEach(id => {
            document.getElementById(id).classList.remove('filter-active');
        });
        document.getElementById('participantSearch').classList.remove('filter-active');
    }

    // Show/hide shift type cards based on type filter
    document.querySelectorAll('.shift-type-card').forEach(card => {
        if (typeFilter === 'all' || card.dataset.shiftTypeId === typeFilter) {
            card.style.display = '';
        } else {
            card.style.display = 'none';
        }
    });

    // Show/hide shifts based on date, status, and participant filters
    document.querySelectorAll('.day-section').forEach(daySection => {
        const shiftRows = daySection.querySelectorAll('tr[data-shift-info]');
        let visibleShifts = 0;

        shiftRows.forEach(row => {
            const shiftInfo = JSON.parse(row.dataset.shiftInfo);
            let show = true;

            // Date filter
            if (dateFilter !== 'all') {
                const shiftDate = new Date(shiftInfo.startTime);
                const filterDate = new Date(dateFilter);
                
                // Compare dates without time
                shiftDate.setHours(0, 0, 0, 0);
                filterDate.setHours(0, 0, 0, 0);
                
                if (shiftDate.getTime() !== filterDate.getTime()) {
                    show = false;
                }
            }

            // Status filter
            if (statusFilter !== 'all' && show) {
                const occupancyRate = shiftInfo.participants.length / shiftInfo.required;
                if (statusFilter === 'full' && occupancyRate < 1) {
                    show = false;
                } else if (statusFilter === 'open' && occupancyRate >= 1) {
                    show = false;
                } else if (statusFilter === 'empty' && shiftInfo.participants.length > 0) {
                    show = false;
                } else if (statusFilter === 'critical' && occupancyRate >= 0.5) {
                    show = false;
                }
            }

            // Participant search
            if (participantSearch && show) {
                const hasMatchingParticipant = shiftInfo.participants.some(
                    participant => participant.toLowerCase().includes(participantSearch)
                );
                if (!hasMatchingParticipant) {
                    show = false;
                }
            }

            row.style.display = show ? '' : 'none';
            if (show) visibleShifts++;
        });

        // Show/hide the entire day section based on whether it has visible shifts
        daySection.style.display = visibleShifts > 0 ? '' : 'none';
    });

    // Show/hide shift type cards if they have no visible shifts
    document.querySelectorAll('.shift-type-card').forEach(card => {
        if (card.style.display !== 'none') { // Only check cards that weren't hidden by type filter
            const hasVisibleShifts = Array.from(card.querySelectorAll('.day-section')).some(
                section => section.style.display !== 'none'
            );
            card.style.display = hasVisibleShifts ? '' : 'none';
        }
    });
}

export function loadShiftTypes() {
    fetch('/shifttypes')
        .then(response => response.json())
        .then(shiftTypes => {
            const container = document.getElementById('shiftTypesContainer');
            container.innerHTML = '';
            
            // Update shift type select in new shift form
            const select = document.querySelector('#newShiftForm select[name="shiftType"]');
            select.innerHTML = '';
            
            Object.entries(shiftTypes).forEach(([id, shiftType]) => {
                // Add to select options
                select.innerHTML += `<option value="${id}">${getLocalizedTitle(shiftType)}</option>`;
            });

            // Create filter controls
            const filterControls = createFilterControls(shiftTypes);
            container.appendChild(filterControls);

            // Create shift type tabs
            const shiftTypeCards = createShiftTypeCards(shiftTypes);
            container.appendChild(shiftTypeCards);
        });
}

function createShiftTypeOptions(shiftTypes) {
    return Object.entries(shiftTypes)
        .map(([id, type]) => `<option value="${id}">${getLocalizedTitle(type)}</option>`)
        .join('');
}

function createDateOptions(dates) {
    return Array.from(dates)
        .sort()
        .map(dateStr => {
            const date = new Date(dateStr);
            return `<option value="${dateStr}">${formatDate(date)}</option>`;
        })
        .join('');
}

function createFilterControls(shiftTypes) {
    // Collect all unique dates from shifts
    const dates = new Set();
    Object.values(shiftTypes).forEach(shiftType => {
        shiftType.shifts.forEach(shift => {
            const date = new Date(shift.startTime);
            date.setHours(0, 0, 0, 0);
            dates.add(date.toISOString());
        });
    });

    const div = document.createElement('div');
    div.className = 'mb-4';
    div.innerHTML = `
        <div class="row mb-3">
            <div class="col">
                <button class="btn btn-outline-primary me-2 admin-controls" data-bs-toggle="modal" data-bs-target="#newShiftTypeModal" data-i18n="new_shift_type">
                    <i class="bi bi-plus-circle me-1"></i> ${t('new_shift_type')}
                </button>
                <button class="btn btn-outline-primary me-2 admin-controls" data-bs-toggle="modal" data-bs-target="#newShiftModal" data-i18n="new_shift">
                    <i class="bi bi-calendar-plus me-1"></i> ${t('new_shift')}
                </button>
                <button class="btn btn-outline-primary admin-controls" onclick="showSortModal()" data-i18n="sort_shift_types">
                    <i class="bi bi-arrow-down-up me-1"></i> ${t('sort_shift_types')}
                </button>
            </div>
        </div>
        <div class="card filter-card">
            <div class="card-body">
                <div class="row align-items-end">
                    <div class="col-md-3">
                        <label class="form-label" data-i18n="shift_type_filter">${t('shift_type_filter')}</label>
                        <select class="form-select" id="shiftTypeFilter" onchange="applyFilters()">
                            <option value="all" data-i18n="all_shift_types">${t('all_shift_types')}</option>
                            ${createShiftTypeOptions(shiftTypes)}
                        </select>
                    </div>
                    <div class="col-md-3">
                        <label class="form-label" data-i18n="day">${t('day')}</label>
                        <select class="form-select" id="timeRangeFilter" onchange="applyFilters()">
                            <option value="all" data-i18n="all_days">${t('all_days')}</option>
                            ${createDateOptions(dates)}
                        </select>
                    </div>
                    <div class="col-md-2">
                        <label class="form-label" data-i18n="status">${t('status')}</label>
                        <select class="form-select" id="statusFilter" onchange="applyFilters()">
                            <option value="all" data-i18n="all">${t('all')}</option>
                            <option value="empty" data-i18n="completely_empty">${t('completely_empty')}</option>
                            <option value="critical" data-i18n="critical">${t('critical')}</option>
                            <option value="open" data-i18n="spots_available">${t('spots_available')}</option>
                            <option value="full" data-i18n="fully_booked">${t('fully_booked')}</option>
                        </select>
                    </div>
                    <div class="col-md-2">
                        <label class="form-label" data-i18n="participant_search">${t('participant_search')}</label>
                        <input type="text" class="form-control" id="participantSearch" 
                               onkeyup="applyFilters()"
                               data-i18n-placeholder="participant_search_placeholder">
                    </div>
                    <div class="col-md-2">
                        <button class="btn btn-outline-secondary w-100" onclick="resetFilters()" id="resetFiltersBtn" style="display: none;" data-i18n="reset_filters">
                            <i class="bi bi-x-circle"></i> ${t('reset_filters')}
                        </button>
                    </div>
                </div>
                <div id="activeFilters" class="mt-3" style="display: none;">
                    <div class="alert alert-info mb-0">
                        <i class="bi bi-funnel-fill me-2"></i>
                        <strong data-i18n="active_filters">${t('active_filters')}:</strong>
                        <span id="activeFiltersText"></span>
                    </div>
                </div>
            </div>
        </div>
    `;
    return div;
}

// Add to window object for click handling
window.toggleDescription = toggleDescription;
window.showEditShiftTypeModal = showEditShiftTypeModal;
window.showEditShiftModal = showEditShiftModal;
window.updateShiftType = updateShiftType;
window.updateShift = updateShift;
window.deleteShiftType = deleteShiftType;
window.deleteShift = deleteShift;
window.resetFilters = resetFilters;
window.showJoinModal = showJoinModal;
window.showSortModal = showSortModal;
window.saveSortOrder = saveSortOrder;
window.showNewShiftForType = showNewShiftForType;
window.duplicateShift = duplicateShift;
window.applyDayDelta = applyDayDelta;

function resetFilters() {
    document.getElementById('shiftTypeFilter').value = 'all';
    document.getElementById('timeRangeFilter').value = 'all';
    document.getElementById('statusFilter').value = 'all';
    document.getElementById('participantSearch').value = '';
    applyFilters();
}

function toggleDescription(elementId) {
    const description = document.getElementById(elementId);
    if (description) {
        description.classList.toggle('expanded');
        description.classList.toggle('truncated');
    }
}

function createShiftTypeCards(shiftTypes) {
    const container = document.createElement('div');
    container.setAttribute('role', 'region');
    container.setAttribute('aria-label', 'Schichtplan');

    // Sort shift types by sortOrder
    const sortedTypes = Object.entries(shiftTypes)
        .sort((a, b) => (a[1].sortOrder || 0) - (b[1].sortOrder || 0));

    sortedTypes.forEach(([typeId, shiftType]) => {
        const card = document.createElement('section');
        card.className = 'shift-type-card';
        card.dataset.shiftTypeId = typeId;
        card.setAttribute('aria-labelledby', `shift-type-${typeId}`);

        // Group shifts by day
        const shiftsByDay = groupShiftsByDay(shiftType.shifts);

        // Create description HTML with truncation if needed
        const description = getLocalizedDescription(shiftType) || '';
        const shouldTruncate = description.length > 150;
        const descriptionHtml = description ? `
            <div class="description-container">
                <p id="description-${typeId}" class="description ${shouldTruncate ? 'truncated' : ''}">${description}</p>
                ${shouldTruncate ? `
                    <button class="btn btn-link btn-sm description-toggle" 
                            onclick="toggleDescription('description-${typeId}')"
                            aria-controls="description-${typeId}"
                            aria-expanded="false">
                        <i class="bi bi-chevron-down"></i> ${t('show_more')}
                    </button>
                ` : ''}
            </div>
        ` : '';

        card.innerHTML = `
            <div class="shift-type-header">
                <div class="d-flex justify-content-between align-items-start">
                    <div class="flex-grow-1">
                        <h3 id="shift-type-${typeId}">
                            ${getLocalizedTitle(shiftType)}
                            ${shiftType.password ? '<i class="bi bi-lock-fill text-muted ms-2" title="' + t('shift_type_password_hint') + '"></i>' : ''}
                        </h3>
                        ${descriptionHtml}
                        ${shiftType.location ? `<p class="mb-0"><small class="text-muted">${t('location')}: <span class="location">${shiftType.location}</span></small></p>` : ''}
                    </div>
                    <div class="btn-group btn-group-sm admin-controls ms-3">
                        <button class="btn btn-outline-primary" 
                                onclick="showNewShiftForType('${typeId}')"
                                aria-label="${t('new_shift')} ${getLocalizedTitle(shiftType)}">
                            <i class="bi bi-plus-circle"></i>
                        </button>
                        <button class="btn btn-outline-primary" 
                                onclick="showEditShiftTypeModal('${typeId}')"
                                aria-label="${t('edit_shift_type')} ${getLocalizedTitle(shiftType)}">
                            <i class="bi bi-pencil"></i>
                        </button>
                        <button class="btn btn-outline-danger" 
                                onclick="deleteShiftType('${typeId}')"
                                aria-label="${t('delete')} ${getLocalizedTitle(shiftType)}">
                            <i class="bi bi-trash"></i>
                        </button>
                    </div>
                </div>
            </div>
            ${Array.from(shiftsByDay.entries()).map(([dateKey, dayShifts]) => {
                // Calculate summary for this day
                const summary = dayShifts.reduce((acc, shift) => {
                    const needed = shift.required - shift.participants.length;
                    if (needed > 0) {
                        acc.totalNeeded += needed;
                        if (shift.participants.length === 0) {
                            acc.emptyShifts++;
                        } else if (shift.participants.length / shift.required < 0.5) {
                            acc.criticalShifts++;
                        }
                    }
                    return acc;
                }, { totalNeeded: 0, emptyShifts: 0, criticalShifts: 0 });

                return `
                <div class="day-section mb-4" role="region" aria-label="Schichten am ${formatDate(dateKey)}">
                    <div class="d-flex justify-content-between align-items-center mb-2">
                        <h5 id="date-${dateKey}" class="mb-0">${formatDate(dateKey)}</h5>
                        ${summary.totalNeeded > 0 ? `
                            <div class="day-summary">
                                ${summary.emptyShifts > 0 ? `
                                    <span title="${t('completely_empty_shifts')}">
                                        <i class="bi bi-exclamation-triangle"></i> ${summary.emptyShifts} ${t('empty_shifts')}
                                    </span>
                                ` : ''}
                                ${summary.criticalShifts > 0 ? `
                                    <span title="${t('critically_staffed_shifts')}">
                                        <i class="bi bi-exclamation-circle"></i> ${summary.criticalShifts} ${t('critical_shifts')}
                                    </span>
                                ` : ''}
                                <span title="${t('needed_helpers')}">
                                    <i class="bi bi-person-plus"></i> ${summary.totalNeeded}
                                </span>
                            </div>
                        ` : ''}
                    </div>
                    <div class="table-responsive">
                        <table class="table table-sm table-hover" aria-labelledby="date-${dateKey}">
                            <thead>
                                <tr>
                                    <th scope="col" data-i18n="time">${t('time')}</th>
                                    <th scope="col" class="d-none d-md-table-cell" data-i18n="duration">${t('duration')}</th>
                                    <th scope="col" data-i18n="occupancy">${t('occupancy')}</th>
                                    <th scope="col" class="text-end" data-i18n="actions">${t('actions')}</th>
                                </tr>
                            </thead>
                            <tbody>
                                ${dayShifts.map(shift => {
                                    const startTime = formatTime(shift.startTime);
                                    const endTime = calculateEndTime(shift.startTime, shift.duration);
                                    const occupancyRate = shift.participants.length / shift.required;
                                    const progressBarClass = occupancyRate < 0.5 ? 'progress-bar-danger' : 
                                                   occupancyRate < 1 ? 'progress-bar-warning' : 
                                                   'progress-bar-success';
                                    return `
                                    <tr data-shift-info='${JSON.stringify({
                                        startTime: shift.startTime,
                                        participants: shift.participants,
                                        required: shift.required
                                    })}' ${shift.participants.length === 0 ? 'data-empty="true"' : ''} role="row">
                                        <td role="cell">
                                            <time datetime="${shift.startTime}">${startTime} - ${endTime}</time>
                                            <div class="d-md-none text-muted small">${shift.duration} Min.</div>
                                        </td>
                                        <td role="cell" class="d-none d-md-table-cell">${shift.duration} Min.</td>
                                        <td role="cell">
                                            <div class="d-flex align-items-center">
                                                <div class="progress flex-grow-1" role="progressbar" 
                                                     aria-valuenow="${shift.participants.length}"
                                                     aria-valuemin="0"
                                                     aria-valuemax="${shift.required}"
                                                     aria-label="Besetzung der Schicht">
                                                    <div class="progress-bar ${progressBarClass}" 
                                                         style="width: ${(occupancyRate) * 100}%">
                                                    </div>
                                                </div>
                                                <small class="ms-2" aria-label="Anzahl Teilnehmer">
                                                    ${shift.participants.length}/${shift.required}
                                                </small>
                                            </div>
                                            <div class="mt-1 small" role="list" aria-label="Teilnehmerliste">
                                                ${shift.participants.length > 0 ? 
                                                    shift.participants.map(p => `
                                                        <span class="badge" role="listitem">
                                                            ${p}
                                                            <button class="btn btn-link btn-sm" 
                                                                    onclick="removeParticipant('${typeId}', '${shift.id}', '${p}')"
                                                                    aria-label="${p} aus der Schicht entfernen">
                                                                <i class="bi bi-x" aria-hidden="true"></i>
                                                            </button>
                                                        </span>
                                                    `).join('') 
                                                    : `<span class="text-muted" role="listitem" data-i18n="no_participants">${t('no_participants')}</span>`
                                                }
                                            </div>
                                        </td>
                                        <td role="cell" class="text-end">
                                            <div class="btn-group btn-group-sm" role="group" aria-label="Schicht Aktionen">
                                                <button class="btn btn-outline-success" 
                                                        onclick="showJoinModal('${typeId}', '${shift.id}')"
                                                        ${occupancyRate < 1 ? '' : 'disabled'}
                                                        aria-label="${t('sign_up_for_shift')}"
                                                        title="${t('sign_up')}">
                                                    <i class="bi bi-person-plus" aria-hidden="true"></i>
                                                </button>
                                                <button class="btn btn-outline-primary admin-controls" 
                                                        onclick="duplicateShift('${typeId}', '${shift.id}')"
                                                        aria-label="${t('duplicate_shift')}"
                                                        title="${t('duplicate_shift')}">
                                                    <i class="bi bi-files" aria-hidden="true"></i>
                                                </button>
                                                <button class="btn btn-outline-primary admin-controls" 
                                                        onclick="showEditShiftModal('${typeId}', '${shift.id}')"
                                                        aria-label="${t('edit_shift')}"
                                                        title="${t('edit')}">
                                                    <i class="bi bi-pencil" aria-hidden="true"></i>
                                                </button>
                                                <button class="btn btn-outline-danger admin-controls" 
                                                        onclick="deleteShift('${typeId}', '${shift.id}')"
                                                        aria-label="${t('delete_shift')}"
                                                        title="${t('delete')}">
                                                    <i class="bi bi-trash" aria-hidden="true"></i>
                                                </button>
                                            </div>
                                        </td>
                                    </tr>
                                `}).join('')}
                            </tbody>
                        </table>
                    </div>
                </div>
            `}).join('')}
        `;
        container.appendChild(card);
    });

    return container;
}

export function createShiftType() {
    const form = document.getElementById('newShiftTypeForm');
    const formData = new FormData(form);
    
    // Validiere die Titel (max. 150 Zeichen)
    const titleDE = formData.get('titleDE');
    const titleEN = formData.get('titleEN');
    
    if (titleDE && titleDE.trim().length > 150) {
        showError('error_title_de_too_long');
        return;
    }
    
    if (titleEN && titleEN.trim().length > 150) {
        showError('error_title_en_too_long');
        return;
    }
    
    const shiftType = {
        id: uuidv4(),
        titleDE: titleDE,
        titleEN: titleEN,
        descriptionDE: formData.get('descriptionDE'),
        descriptionEN: formData.get('descriptionEN'),
        location: formData.get('location'),
        password: formData.get('password'),
        shifts: []
    };

    fetchWithAuth('/shifttypes', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(shiftType),
    })
    .then(response => {
        if (response.ok) {
            bootstrap.Modal.getInstance(document.getElementById('newShiftTypeModal')).hide();
            form.reset();
            loadShiftTypes();
        } else {
            showError('error_creating_shift_type');
        }
    });
}

export async function createShift() {
    const form = document.getElementById('newShiftForm');
    const formData = new FormData(form);
    const typeId = formData.get('shiftType');
    const repeatCount = parseInt(formData.get('repeatCount'));

    const shiftData = {
        startTime: formData.get('startTime'),
        duration: parseInt(formData.get('duration')),
        required: parseInt(formData.get('required'))
    };

    let currentStartTime = shiftData.startTime;

    for (let i = 0; i < repeatCount; i++) {
        const shift = {
            id: uuidv4(),
            startTime: currentStartTime,
            duration: shiftData.duration,
            required: shiftData.required,
            participants: []
        };

        try {
            const response = await fetchWithAuth(`/shifttypes/${typeId}/shifts`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(shift),
            });
            
            if (!response.ok) {
                throw new Error(t('error_creating_shift'));
            }

            const nextDate = new Date(currentStartTime);
            nextDate.setMinutes(nextDate.getMinutes() + shift.duration);
            currentStartTime = nextDate.toISOString().slice(0, 16);
        } catch (error) {
            alert(error.message);
            break;
        }
    }

    bootstrap.Modal.getInstance(document.getElementById('newShiftModal')).hide();
    form.reset();
    loadShiftTypes();
}

export function deleteShift(typeId, shiftId) {
    if (confirm('Möchten Sie diese Schicht wirklich löschen?')) {
        fetchWithAuth(`/shifttypes/${typeId}/shifts/${shiftId}`, {
            method: 'DELETE',
        })
        .then(response => {
            if (response.ok) {
                loadShiftTypes();
            } else {
                alert('Fehler beim Löschen der Schicht');
            }
        });
    }
}

export function deleteShiftType(typeId) {
    if (confirm('Möchten Sie diesen Schichttyp und alle zugehörigen Schichten wirklich löschen?')) {
        fetchWithAuth(`/shifttypes/${typeId}`, {
            method: 'DELETE',
        })
        .then(response => {
            if (response.ok) {
                loadShiftTypes();
            } else {
                alert('Fehler beim Löschen des Schichttyps');
            }
        });
    }
}

function showEditShiftTypeModal(typeId) {
    const form = document.getElementById('editShiftTypeForm');
    
    // Fetch current data from server
    fetch('/shifttypes')
        .then(response => response.json())
        .then(shiftTypes => {
            const shiftType = shiftTypes[typeId];
            if (shiftType) {
                form.elements.id.value = typeId;
                form.elements.titleDE.value = shiftType.titleDE;
                form.elements.titleEN.value = shiftType.titleEN;
                form.elements.descriptionDE.value = shiftType.descriptionDE || '';
                form.elements.descriptionEN.value = shiftType.descriptionEN || '';
                form.elements.location.value = shiftType.location || '';
                form.elements.password.value = shiftType.password || '';
                
                new bootstrap.Modal(document.getElementById('editShiftTypeModal')).show();
            }
        });
}

function showEditShiftModal(typeId, shiftId) {
    const form = document.getElementById('editShiftForm');
    const row = document.querySelector(`tr[data-shift-info] button[onclick*="${shiftId}"]`).closest('tr');
    if (row) {
        const shiftInfo = JSON.parse(row.dataset.shiftInfo);
        
        form.elements.typeId.value = typeId;
        form.elements.shiftId.value = shiftId;
        form.elements.startTime.value = new Date(shiftInfo.startTime).toISOString().slice(0, 16);
        form.elements.duration.value = shiftInfo.duration;
        form.elements.required.value = shiftInfo.required;
        
        new bootstrap.Modal(document.getElementById('editShiftModal')).show();
    }
}

function updateShiftType() {
    const form = document.getElementById('editShiftTypeForm');
    const formData = new FormData(form);
    const typeId = formData.get('id');
    
    const shiftType = {
        id: typeId,
        titleDE: formData.get('titleDE'),
        titleEN: formData.get('titleEN'),
        descriptionDE: formData.get('descriptionDE'),
        descriptionEN: formData.get('descriptionEN'),
        location: formData.get('location'),
        password: formData.get('password')
    };

    // First fetch the current shift type to preserve the shifts
    fetch(`/shifttypes`)
        .then(response => response.json())
        .then(shiftTypes => {
            const currentShiftType = shiftTypes[typeId];
            shiftType.shifts = currentShiftType.shifts; // Preserve existing shifts

            return fetchWithAuth(`/shifttypes/${typeId}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(shiftType),
            });
        })
        .then(response => {
            if (response.ok) {
                bootstrap.Modal.getInstance(document.getElementById('editShiftTypeModal')).hide();
                form.reset();
                loadShiftTypes();
            } else {
                showError('error_updating_shift_type');
            }
        })
        .catch(error => {
            console.error('Error updating shift type:', error);
            showError('error_updating_shift_type');
        });
}

function updateShift() {
    const form = document.getElementById('editShiftForm');
    const formData = new FormData(form);
    
    const shift = {
        id: formData.get('shiftId'),
        startTime: new Date(formData.get('startTime')).toISOString(),
        duration: parseInt(formData.get('duration')),
        required: parseInt(formData.get('required'))
    };

    fetchWithAuth(`/shifttypes/${formData.get('typeId')}/shifts/${shift.id}`, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(shift),
    })
    .then(response => {
        if (response.ok) {
            bootstrap.Modal.getInstance(document.getElementById('editShiftModal')).hide();
            form.reset();
            loadShiftTypes();
        } else {
            alert('Fehler beim Aktualisieren der Schicht');
        }
    });
}

function showError(key) {
    alert(t(key));
}

function confirmDelete(key, params = {}) {
    return confirm(t(key, params));
}

function getLocalizedTitle(shiftType) {
    return currentLanguage === 'de' ? shiftType.titleDE : shiftType.titleEN;
}

function getLocalizedDescription(shiftType) {
    return currentLanguage === 'de' ? shiftType.descriptionDE : shiftType.descriptionEN;
}

function showSortModal() {
    fetch('/shifttypes')
        .then(response => response.json())
        .then(shiftTypes => {
            const sortedTypes = Object.entries(shiftTypes)
                .sort((a, b) => a[1].sortOrder - b[1].sortOrder)
                .map(([id, type]) => ({ id, title: getLocalizedTitle(type) }));

            const modal = document.createElement('div');
            modal.className = 'modal fade';
            modal.id = 'sortModal';
            modal.setAttribute('tabindex', '-1');
            modal.innerHTML = `
                <div class="modal-dialog">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title" data-i18n="sort_shift_types">${t('sort_shift_types')}</h5>
                            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                        </div>
                        <div class="modal-body">
                            <p class="text-muted" data-i18n="sort_shift_types_hint">${t('sort_shift_types_hint')}</p>
                            <ul class="list-group" id="sortableList">
                                ${sortedTypes.map((type, index) => `
                                    <li class="list-group-item" data-id="${type.id}" data-order="${index}">
                                        <i class="bi bi-grip-vertical me-2 text-muted"></i>
                                        ${type.title}
                                    </li>
                                `).join('')}
                            </ul>
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" data-bs-dismiss="modal" data-i18n="cancel">${t('cancel')}</button>
                            <button type="button" class="btn btn-primary" onclick="saveSortOrder()" data-i18n="save">${t('save')}</button>
                        </div>
                    </div>
                </div>
            `;

            document.body.appendChild(modal);
            
            // Initialize Sortable
            new Sortable(document.getElementById('sortableList'), {
                animation: 150,
                handle: '.bi-grip-vertical',
                onEnd: function(evt) {
                    // Update data-order attributes after drag
                    const items = evt.target.querySelectorAll('.list-group-item');
                    items.forEach((item, index) => {
                        item.dataset.order = index;
                    });
                }
            });

            const modalInstance = new bootstrap.Modal(modal);
            modalInstance.show();
            
            // Clean up when modal is hidden
            modal.addEventListener('hidden.bs.modal', function () {
                modal.remove();
            });
        });
}

function saveSortOrder() {
    const items = document.querySelectorAll('#sortableList .list-group-item');
    const updates = Array.from(items).map((item, index) => ({
        id: item.dataset.id,
        sortOrder: index
    }));

    Promise.all(updates.map(update => 
        fetchWithAuth(`/shifttypes/${update.id}/sort`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ sortOrder: update.sortOrder }),
        })
    ))
    .then(() => {
        bootstrap.Modal.getInstance(document.getElementById('sortModal')).hide();
        loadShiftTypes();
    })
    .catch(error => {
        console.error(t('error_updating_sort_order'), error);
        showError('error_updating_sort_order');
    });
}

function showNewShiftForType(typeId) {
    const form = document.getElementById('newShiftForm');
    form.reset(); // Reset form first
    form.elements.shiftType.value = typeId;
    form.elements.repeatCount.value = 1;
    
    new bootstrap.Modal(document.getElementById('newShiftModal')).show();
}

function duplicateShift(typeId, shiftId) {
    // Find the shift data
    fetch('/shifttypes')
        .then(response => response.json())
        .then(shiftTypes => {
            const shiftType = shiftTypes[typeId];
            const shift = shiftType.shifts.find(s => s.id === shiftId);
            
            if (shift) {
                const form = document.getElementById('newShiftForm');
                form.reset(); // Reset form first
                
                form.elements.shiftType.value = typeId;
                form.elements.startTime.value = shift.startTime.slice(0, 16);
                form.elements.duration.value = shift.duration;
                form.elements.required.value = shift.required;
                form.elements.repeatCount.value = 1;
                
                new bootstrap.Modal(document.getElementById('newShiftModal')).show();
            }
        });
}

function applyDayDelta() {
    const dayDelta = parseInt(document.getElementById('dayDelta').value);
    const dateText = document.querySelector('#newShiftForm small.text-muted').textContent;
    const match = dateText.match(/\d{2}\.\d{2}\.\d{4}/);
    
    if (!isNaN(dayDelta) && match) {
        const [day, month, year] = match[0].split('.');
        // Create date strings that explicitly avoid timezone conversion
        const dayZeroStr = `${year}-${month.padStart(2, '0')}-${day.padStart(2, '0')}T00:00`;
        const targetDate = new Date(dayZeroStr);
        targetDate.setDate(targetDate.getDate() + dayDelta);
        
        // Format as YYYY-MM-DDT00:00
        const targetDateStr = targetDate.toISOString().slice(0, 16);
        document.getElementById('startTime').value = targetDateStr;
    }
}