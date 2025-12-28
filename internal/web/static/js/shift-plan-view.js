// Import the modal classes
import { ShiftModal } from './shift-modal.js';
import { ShiftTypeModal } from './shift-type-modal.js';
import { DeleteTypeModal } from './delete-type-modal.js';
import { DeleteShiftModal } from './delete-shift-modal.js';
import { JoinModal } from './join-modal.js';
import { RemoveModal } from './remove-modal.js';
import { AdminModal } from './admin-modal.js';
import { SortTypesModal } from './sort-types-modal.js';
import { SettingsModal } from './settings-modal.js';
import { FilterManager } from './filter-manager.js';
import { ExportModal } from './export-modal.js';
import { buildTimetableModel } from './timetable-view-model.js';

document.addEventListener('alpine:init', () => {
    Alpine.data('shiftPlanView', () => ({
        // State
        plan: null,
        shiftTypes: [],
        isLoading: true,
        error: null,

        // Modal state
        currentShift: null,
        currentParticipant: null,

        // Admin state
        isAdmin: false,
        adminToken: '',

        // Initialize managers
        filterManager: null,
        shiftModal: null,
        shiftTypeModal: null,
        deleteTypeModal: null,
        deleteShiftModal: null,
        joinModal: null,
        removeModal: null,
        adminModal: null,
        sortTypesModal: null,
        settingsModal: null,
        exportModal: null,
        exportFormat: 'json',
        isExporting: false,
        exportError: null,

        // View state
        viewMode: 'timetable',
        planToken: '',
        timetableSlotHeight: 16,
        expandedTimetableShifts: {},
        shiftOverflowState: {},
        activeDayKey: null,
        scrollTimeout: null,

        // Collision detection state
        participantCollisions: {}, // Map: participant name -> array of colliding shift IDs

        // Computed property für den Ladezustand
        get isReady() {
            return !this.isLoading && !this.error && this.plan;
        },

        get isTimetableView() {
            return this.viewMode === 'timetable';
        },

        get timetableMinuteHeight() {
            return this.timetableSlotHeight / 15;
        },

        // Computed property für verfügbare Tage
        get availableDays() {
            return this.filterManager.getAvailableDays(this.shiftTypes);
        },

        // Computed property für gefilterte Schichttypen
        get filteredShiftTypes() {
            return this.filterManager.filterShiftTypes(this.shiftTypes);
        },

        get timetableModel() {
            return buildTimetableModel({
                shiftTypes: this.filteredShiftTypes || [],
                plan: this.plan || {},
                locale: this.$locale()
            });
        },

        // Filter functions
        filterShiftsByDate(shifts) {
            return shifts;
        },

        // Filter actions
        toggleType(typeId) {
            this.filterManager.toggleType(typeId);
        },

        clearFilters() {
            this.filterManager.clearFilters();
        },

        activeFilterCount() {
            return this.filterManager.getActiveFilterCount();
        },

        selectedTypeNames() {
            return this.filterManager.getSelectedTypeNames(this.shiftTypes, this.plan.is_multilingual, this.$locale());
        },

        toggleAllTypes() {
            this.filterManager.toggleAllTypes(this.shiftTypes.map(t => t.id));
        },

        // Initialisierung
        init() {
            // Setze Standardwerte
            this.plan = {
                is_multilingual: false,
                title_en: '',
                title_de: '',
                description_en: '',
                description_de: '',
                start_date: '',
                end_date: '',
                token: ''
            };

            // Hole Token aus URL
            const pathParts = window.location.pathname.split('/');
            const token = pathParts[pathParts.length - 1];
            this.planToken = token;
            this.initializeViewMode(token);

            // Versuche Admin-Token aus localStorage zu laden
            const storedToken = localStorage.getItem(`admin_token_${token}`);
            if (storedToken) {
                this.isAdmin = true;
                this.adminToken = storedToken;
            }

            // Initialize managers
            this.filterManager = new FilterManager();
            this.shiftModal = new ShiftModal(
                this.plan.start_date,
                this.shiftTypes,
                token,
                this.adminToken,
                this.handleShiftsUpdate.bind(this)
            );
            this.shiftTypeModal = new ShiftTypeModal(this.plan.is_multilingual, token, this.adminToken, this.handleShiftTypeSuccess.bind(this));
            this.deleteTypeModal = new DeleteTypeModal(token, this.adminToken, this.handleDeleteTypeSuccess.bind(this));
            this.deleteShiftModal = new DeleteShiftModal(token, this.adminToken, this.handleDeleteShiftSuccess.bind(this));
            this.joinModal = new JoinModal(token, this.handleShiftUpdate.bind(this));
            this.removeModal = new RemoveModal(token, this.handleShiftUpdate.bind(this));
            this.adminModal = new AdminModal(token, this.handleAdminSuccess.bind(this));
            this.sortTypesModal = new SortTypesModal(
                token,
                this.adminToken,
                this.handleSortTypesSuccess.bind(this)
            );
            this.settingsModal = new SettingsModal(
                token,
                this.adminToken,
                this.handleSettingsSuccess.bind(this)
            );
            this.exportModal = new ExportModal(token, this.adminToken, this.filterManager);

            // Load initial data
            this.loadPlan(token);
        },

        initializeViewMode(token) {
            const params = new URLSearchParams(window.location.search);
            const requested = params.get('view');
            const stored = localStorage.getItem(`shift_view_mode_${token}`);

            if (requested === 'list' || requested === 'timetable') {
                this.viewMode = requested;
                localStorage.setItem(`shift_view_mode_${token}`, requested);
                return;
            }

            if (stored === 'list' || stored === 'timetable') {
                this.viewMode = stored;
                this.syncViewModeQuery(stored);
                return;
            }

            this.viewMode = 'timetable';
            this.syncViewModeQuery('timetable');
        },

        syncViewModeQuery(mode) {
            const url = new URL(window.location.href);
            if (mode === 'timetable') {
                url.searchParams.delete('view');
            } else {
                url.searchParams.set('view', mode);
            }
            const newUrl = `${url.pathname}${url.search ? `?${url.searchParams.toString()}` : ''}`;
            window.history.replaceState({}, '', newUrl);
        },

        setViewMode(mode) {
            if (mode !== 'list' && mode !== 'timetable') return;
            if (this.viewMode === mode) return;
            this.viewMode = mode;
            if (this.planToken) {
                localStorage.setItem(`shift_view_mode_${this.planToken}`, mode);
            }
            this.syncViewModeQuery(mode);
            this.$nextTick(() => {
                this.$dispatch('shift-view-mode-changed', { mode });
                // Initialize active day when switching to timetable view
                if (mode === 'timetable') {
                    this.initTimetableActiveDay();
                }
            });
        },

        getTimetableColumnTemplate(day) {
            const columnCount = day?.columns?.length || 0;
            const count = Math.max(columnCount, 1);
            return `repeat(${count}, minmax(220px, 1fr))`;
        },

        getTimetableDayShiftCount(day) {
            if (!day?.columns) return 0;
            return day.columns.reduce((acc, column) => acc + (column.shifts?.length || 0), 0);
        },

        getTimetableShiftStyle(shift) {
            if (!shift) return {};
            const minuteHeight = this.timetableMinuteHeight;
            const top = Math.max(0, (shift.startMinutes || 0) * minuteHeight);
            const height = Math.max(this.timetableSlotHeight, (shift.durationMinutes || this.timetableSlotHeight) * minuteHeight);
            const laneCount = Math.max(shift.laneCount || 1, 1);
            const laneIndex = Math.min(shift.laneIndex || 0, laneCount - 1);
            const widthPercent = 100 / laneCount;
            const leftPercent = widthPercent * laneIndex;
            const horizontalPadding = laneCount > 1 ? 6 : 4;

            return {
                top: `${top}px`,
                height: `${height}px`,
                left: `calc(${leftPercent}% + 2px)`,
                width: `calc(${widthPercent}% - ${horizontalPadding}px)`,
                '--shift-color': shift.typeColor || 'oklch(var(--p))'
            };
        },

        isTimetableShiftExpanded(shiftId) {
            return Boolean(this.expandedTimetableShifts?.[shiftId]);
        },

        toggleTimetableShift(shiftId) {
            if (!shiftId) return;
            // Only toggle if there's overflow
            if (!this.hasShiftOverflow(shiftId)) return;
            const nextState = { ...this.expandedTimetableShifts };
            nextState[shiftId] = !nextState[shiftId];
            this.expandedTimetableShifts = nextState;
        },

        // Check if a shift cell has overflow
        hasShiftOverflow(shiftId) {
            return Boolean(this.shiftOverflowState?.[shiftId]);
        },

        // Detect overflow by comparing content height to container height
        detectShiftOverflow(shiftId, containerEl) {
            if (!shiftId || !containerEl) return;
            
            // Use requestAnimationFrame to ensure layout is complete, especially for wrapped participants
            requestAnimationFrame(() => {
                if (!containerEl.isConnected) return; // Element may have been removed
                
                // Find the content wrapper inside the container
                const contentEl = containerEl.querySelector('.timetable-shift-content');
                if (!contentEl) return;
                
                // Get the container's available height
                const containerHeight = containerEl.clientHeight;
                const contentHeight = contentEl.scrollHeight;
                
                // Account for padding and expand hint space
                // When has-overflow class is present, there's extra padding-bottom (1.25rem = 20px)
                // We need to check if content would overflow without that padding
                const computedStyle = getComputedStyle(containerEl);
                const paddingTop = parseFloat(computedStyle.paddingTop) || 0;
                const paddingBottom = parseFloat(computedStyle.paddingBottom) || 0;
                const availableHeight = containerHeight - paddingTop - paddingBottom;
                
                // Consider overflow if content is taller than available space
                // Add a small buffer (3px) to account for rounding and sub-pixel rendering
                const hasOverflow = contentHeight > (availableHeight - 3);
                
                // Only update state if it changed
                if (this.shiftOverflowState[shiftId] !== hasOverflow) {
                    this.shiftOverflowState = {
                        ...this.shiftOverflowState,
                        [shiftId]: hasOverflow
                    };
                }
            });
        },

        // Get the day navigation bar height from CSS variable
        getDayNavHeight() {
            const container = this.$refs.timetableContainer;
            if (!container) return 40; // Fallback value
            const computedStyle = getComputedStyle(container);
            const height = computedStyle.getPropertyValue('--timetable-day-nav-height');
            return height ? parseInt(height) : 40; // Default to 40px if not found
        },

        // Scroll to a specific day in the timetable
        scrollToDay(dateKey) {
            if (!dateKey || !this.timetableModel?.days) return;
            
            const container = this.$refs.timetableContainer;
            if (!container) return;

            // Find the day section element
            const daySection = container.querySelector(`[data-day-key="${dateKey}"]`);
            if (!daySection) return;

            // Calculate the scroll position
            // We need to account for the sticky headers
            const dayNavHeight = this.getDayNavHeight();
            const headerHeight = 36; // Height of type headers
            
            // Get the position of the day section relative to the container
            const containerRect = container.getBoundingClientRect();
            const dayRect = daySection.getBoundingClientRect();
            
            // Calculate scroll position: current scroll + offset from container top - sticky header heights
            const scrollTop = container.scrollTop + (dayRect.top - containerRect.top) - dayNavHeight - headerHeight;

            // Smooth scroll to the day
            container.scrollTo({
                top: Math.max(0, scrollTop),
                behavior: 'smooth'
            });

            // Update active day immediately for better UX
            this.activeDayKey = dateKey;
        },

        // Handle scroll events to update active day
        handleTimetableScroll() {
            // Debounce scroll handling
            if (this.scrollTimeout) {
                clearTimeout(this.scrollTimeout);
            }

            this.scrollTimeout = setTimeout(() => {
                this.updateActiveDay();
            }, 100);
        },

        // Update which day is currently visible based on scroll position
        updateActiveDay() {
            const container = this.$refs.timetableContainer;
            if (!container || !this.timetableModel?.days || this.timetableModel.days.length === 0) {
                return;
            }

            const containerRect = container.getBoundingClientRect();
            const dayNavHeight = this.getDayNavHeight();
            const headerHeight = 36; // Height of type headers
            const viewportTop = container.scrollTop + dayNavHeight + headerHeight + 20; // Small offset for better UX

            // Find which day section is currently most visible in the viewport
            let activeDay = null;
            let maxVisibleArea = 0;

            this.timetableModel.days.forEach(day => {
                const daySection = container.querySelector(`[data-day-key="${day.dateKey}"]`);
                if (!daySection) return;

                const dayRect = daySection.getBoundingClientRect();
                const dayTop = dayRect.top - containerRect.top + container.scrollTop;
                const dayBottom = dayTop + daySection.offsetHeight;
                const containerBottom = container.scrollTop + container.clientHeight;

                // Calculate how much of this day is visible
                const visibleTop = Math.max(viewportTop, dayTop);
                const visibleBottom = Math.min(containerBottom, dayBottom);
                const visibleArea = Math.max(0, visibleBottom - visibleTop);

                // If this day has more visible area than the current best, make it active
                if (visibleArea > maxVisibleArea) {
                    maxVisibleArea = visibleArea;
                    activeDay = day.dateKey;
                }
            });

            // Fallback: if no day found (shouldn't happen), use the first day
            if (!activeDay && this.timetableModel.days.length > 0) {
                activeDay = this.timetableModel.days[0].dateKey;
            }

            if (activeDay && this.activeDayKey !== activeDay) {
                this.activeDayKey = activeDay;
            }
        },

        // Initialize active day when timetable is shown
        initTimetableActiveDay() {
            this.$nextTick(() => {
                if (this.timetableModel?.days && this.timetableModel.days.length > 0) {
                    // Set initial active day to the first day
                    this.activeDayKey = this.timetableModel.days[0].dateKey;
                    // Update based on scroll position after a short delay
                    setTimeout(() => {
                        this.updateActiveDay();
                    }, 100);
                }
            });
        },

        // Handle register button click in timetable view
        handleTimetableRegister(shift) {
            if (!shift) return;
            
            // Find the actual shift object from shiftTypes
            let actualShift = null;
            for (const type of this.shiftTypes) {
                const found = type.shifts?.find(s => s.id === shift.id);
                if (found) {
                    actualShift = found;
                    break;
                }
            }
            
            if (!actualShift) return;
            
            // Open join modal
            this.openJoinModal(actualShift);
        },

        // Handle right-click context menu for admins
        handleTimetableShiftContextMenu(shift, event) {
            if (!shift || !this.isAdmin) return;
            
            event.preventDefault();
            
            // Find the actual shift object and type from shiftTypes
            let actualShift = null;
            let actualType = null;
            for (const type of this.shiftTypes) {
                const found = type.shifts?.find(s => s.id === shift.id);
                if (found) {
                    actualShift = found;
                    actualType = type;
                    break;
                }
            }
            
            if (!actualShift) return;
            
            this.showTimetableShiftContextMenu(actualShift, actualType, event);
        },

        handleTimetableShiftClick(shift, event) {
            if (!shift) return;
            
            // Find the actual shift object and type from shiftTypes
            let actualShift = null;
            let actualType = null;
            for (const type of this.shiftTypes) {
                const found = type.shifts?.find(s => s.id === shift.id);
                if (found) {
                    actualShift = found;
                    actualType = type;
                    break;
                }
            }
            
            if (!actualShift) return;
            
            // For admins on right-click, show context menu options
            if (this.isAdmin && event.button === 2) {
                event.preventDefault();
                this.showTimetableShiftContextMenu(actualShift, actualType, event);
                return;
            }
            
            // Open join modal for normal click
            this.openJoinModal(actualShift);
        },

        showTimetableShiftContextMenu(shift, type, event) {
            // This will be handled by the template with x-data
            // We just dispatch an event
            this.$dispatch('show-timetable-context-menu', { 
                shift, 
                type,
                x: event.clientX, 
                y: event.clientY 
            });
        },

        handleTimetableShiftEdit(shift, column) {
            // Find the actual shift and type objects
            let actualShift = null;
            let actualType = null;
            for (const type of this.shiftTypes) {
                const found = type.shifts?.find(s => s.id === shift.id);
                if (found) {
                    actualShift = found;
                    actualType = type;
                    break;
                }
            }
            if (actualShift && actualType) {
                this.openShiftModal(actualShift, actualType, false);
            }
        },

        handleTimetableShiftDuplicate(shift, column) {
            // Find the actual shift and type objects
            let actualShift = null;
            let actualType = null;
            for (const type of this.shiftTypes) {
                const found = type.shifts?.find(s => s.id === shift.id);
                if (found) {
                    actualShift = found;
                    actualType = type;
                    break;
                }
            }
            if (actualShift && actualType) {
                this.openShiftModal(actualShift, actualType, true);
            }
        },

        handleTimetableShiftDelete(shift, column) {
            // Find the actual shift and type objects
            let actualShift = null;
            let actualType = null;
            for (const type of this.shiftTypes) {
                const found = type.shifts?.find(s => s.id === shift.id);
                if (found) {
                    actualShift = found;
                    actualType = type;
                    break;
                }
            }
            if (actualShift && actualType) {
                this.openDeleteShiftModal(actualShift, actualType);
            }
        },

        handleTimetableShiftToggleLock(shift, column) {
            // Find the actual shift and type objects
            let actualShift = null;
            let actualType = null;
            for (const type of this.shiftTypes) {
                const found = type.shifts?.find(s => s.id === shift.id);
                if (found) {
                    actualShift = found;
                    actualType = type;
                    break;
                }
            }
            if (actualShift && actualType) {
                if (actualShift.is_locked) {
                    this.unlockShift(actualShift, actualType);
                } else {
                    this.lockShift(actualShift, actualType);
                }
            }
        },

        handleTimetableParticipantRemove(shift, participant, event) {
            if (!shift || !participant) return;
            event.stopPropagation();
            
            // Find the actual shift object from shiftTypes
            let actualShift = null;
            for (const type of this.shiftTypes) {
                const found = type.shifts?.find(s => s.id === shift.id);
                if (found) {
                    actualShift = found;
                    break;
                }
            }
            
            if (!actualShift) return;
            
            // Open remove modal
            this.openRemoveModal(actualShift, participant);
        },

        // Gruppiere Schichten nach Datum
        groupShiftsByDate(shifts) {
            const groups = {};

            // Sortiere Schichten nach Startzeit
            shifts.sort((a, b) => new Date(a.start_time) - new Date(b.start_time));

            // Gruppiere nach Datum unter Berücksichtigung der Tagesgrenze
            shifts.forEach(shift => {
                const shiftDate = new Date(shift.start_time);
                const [boundaryHours, boundaryMinutes] = (this.plan?.day_boundary_time || '00:00').split(':').map(Number);
                
                const shiftHours = shiftDate.getHours();
                const shiftMinutes = shiftDate.getMinutes();
                const shiftTotalMinutes = shiftHours * 60 + shiftMinutes;
                const boundaryTotalMinutes = boundaryHours * 60 + boundaryMinutes;

                let dateKey;
                if (shiftTotalMinutes < boundaryTotalMinutes) {
                    // Wenn die Schicht zwischen 00:00 und der Tagesgrenze liegt
                    const year = shiftDate.getFullYear();
                    const month = String(shiftDate.getMonth() + 1).padStart(2, '0');
                    const day = String(shiftDate.getDate()).padStart(2, '0');
                    
                    // Berechne den vorherigen Tag
                    const prevDate = new Date(shiftDate);
                    prevDate.setDate(prevDate.getDate() - 1);
                    const prevYear = prevDate.getFullYear();
                    const prevMonth = String(prevDate.getMonth() + 1).padStart(2, '0');
                    const prevDay = String(prevDate.getDate()).padStart(2, '0');
                    dateKey = `${prevYear}-${prevMonth}-${prevDay}`;
                } else {
                    // Wenn die Schicht nach der Tagesgrenze liegt
                    const year = shiftDate.getFullYear();
                    const month = String(shiftDate.getMonth() + 1).padStart(2, '0');
                    const day = String(shiftDate.getDate()).padStart(2, '0');
                    dateKey = `${year}-${month}-${day}`;
                }

                if (!groups[dateKey]) {
                    groups[dateKey] = [];
                }
                groups[dateKey].push(shift);
            });

            // Konvertiere in Array von Gruppen und sortiere nach Datum
            return Object.entries(groups)
                .sort(([dateA], [dateB]) => dateA.localeCompare(dateB))
                .map(([date, shifts]) => ({
                    date,
                    shifts
                }));
        },

        // Daten laden
        async loadPlan(token) {
            this.isLoading = true;

            try {
                // Load plan data
                const headers = {};
                if (this.isAdmin && this.adminToken) {
                    headers['X-Admin-Token'] = this.adminToken;
                }

                const response = await fetch(`/api/plans/${token}`, { headers });
                if (!response.ok) throw new Error('Failed to load plan');
                const plan = await response.json();

                // Update plan data
                this.plan = {
                    ...plan,
                    day_boundary_time: plan.day_boundary_time || '00:00' // Ensure default value if not set
                };

                // Load shift types
                const typesResponse = await fetch(`/api/plans/${token}/types`, { headers });
                if (!typesResponse.ok) throw new Error('Failed to load shift types');
                const types = await typesResponse.json();

                // Update shift types with their shifts
                this.shiftTypes = types.map(type => ({
                    ...type,
                    shifts: type.shifts || []
                }));

                // Detect collisions after loading shifts
                this.detectCollisions();

                // Don't reinitialize filter manager - it's already created in init()
                // The templates bind directly to filterManager properties

                // Initialize modals
                this.shiftModal = new ShiftModal(
                    this.plan.start_date,
                    this.shiftTypes,
                    this.plan.token,
                    this.adminToken,
                    this.handleShiftsUpdate.bind(this)
                );
                this.shiftTypeModal = new ShiftTypeModal(
                    this.plan.is_multilingual,
                    this.plan.token,
                    this.adminToken,
                    this.handleShiftTypeSuccess.bind(this)
                );
                this.deleteTypeModal = new DeleteTypeModal(
                    this.plan.token,
                    this.adminToken,
                    this.handleDeleteTypeSuccess.bind(this)
                );
                this.deleteShiftModal = new DeleteShiftModal(
                    this.plan.token,
                    this.adminToken,
                    this.handleDeleteShiftSuccess.bind(this)
                );
                this.joinModal = new JoinModal(
                    this.plan.token,
                    this.handleShiftUpdate.bind(this)
                );
                this.removeModal = new RemoveModal(
                    this.plan.token,
                    this.handleShiftUpdate.bind(this)
                );
                this.adminModal = new AdminModal(
                    this.plan.token,
                    this.handleAdminSuccess.bind(this)
                );
                this.sortTypesModal = new SortTypesModal(
                    this.plan.token,
                    this.adminToken,
                    this.handleSortTypesSuccess.bind(this)
                );
                this.settingsModal = new SettingsModal(
                    this.plan.token,
                    this.adminToken,
                    this.handleSettingsSuccess.bind(this)
                );

            } catch (error) {
                console.error('Error loading plan:', error);
                this.error = error.message;

                // Reset to safe state
                this.plan = {
                    is_multilingual: false,
                    title_en: '',
                    title_de: '',
                    description_en: '',
                    description_de: '',
                    start_date: '',
                    end_date: '',
                    token: '',
                    day_boundary_time: '00:00'
                };
                this.shiftTypes = [];
            } finally {
                this.isLoading = false;
            }
        },

        // Modal functions
        openJoinModal(shift) {
            if (!shift) return;
            this.joinModal.openWith(shift);
        },

        openRemoveModal(shift, participant) {
            if (!shift || !participant) return;
            this.removeModal.openWith(shift, participant);
        },

        // Formatierungshelfer mit Null-Checks
        formatDate(dateStr) {
            if (!dateStr) return '';
            try {
                const date = new Date(dateStr);
                if (isNaN(date.getTime())) return '';

                const options = {
                    weekday: 'long',
                    year: 'numeric',
                    month: 'long',
                    day: 'numeric'
                };

                // Use the current language from Alpine store
                const locale = this.$locale() === 'de' ? 'de-DE' : 'en-US';
                const formatted = date.toLocaleDateString(locale, options);

                // For German: Capitalize first letter
                if (this.$locale() === 'de') {
                    return formatted.charAt(0).toUpperCase() + formatted.slice(1);
                }
                return formatted;
            } catch (error) {
                console.error('Error formatting date:', error);
                return '';
            }
        },

        formatTime(dateStr, duration) {
            if (!dateStr) return '';
            try {
                const startDate = new Date(dateStr);
                if (isNaN(startDate.getTime())) return '';

                // Format start time in 24h format
                const startTime = startDate.toLocaleTimeString(
                    this.$locale() === 'de' ? 'de-DE' : 'en-GB',
                    {
                        hour: '2-digit',
                        minute: '2-digit',
                        hour12: false
                    }
                );

                // Calculate end time
                const endDate = new Date(startDate.getTime());
                endDate.setMinutes(endDate.getMinutes() + duration);
                const endTime = endDate.toLocaleTimeString(
                    this.$locale() === 'de' ? 'de-DE' : 'en-GB',
                    {
                        hour: '2-digit',
                        minute: '2-digit',
                        hour12: false
                    }
                );

                return `${startTime} – ${endTime}`;
            } catch (error) {
                console.error('Error formatting time:', error);
                return '';
            }
        },

        formatDuration(minutes) {
            if (!minutes || isNaN(minutes)) return '';
            try {
                const hours = Math.floor(minutes / 60);
                const mins = minutes % 60;
                return `${hours.toString().padStart(2, '0')}:${mins.toString().padStart(2, '0')}`;
            } catch (error) {
                console.error('Error formatting duration:', error);
                return '';
            }
        },

        formatTimeOnly(time) {
            return new Date(time).toLocaleTimeString(
                this.$locale() === 'de' ? 'de-DE' : 'en-GB',
                { 
                    hour: '2-digit', 
                    minute: '2-digit',
                    hour12: false 
                }
            );
        },

        addMinutes(time, minutes) {
            const date = new Date(time);
            return new Date(date.getTime() + minutes * 60000).toISOString();
        },

        // Helper for radial progress styling
        getProgressStyles(shift) {
            if (!shift.participants) {
                return {
                    background: 'bg-error',
                    text: 'text-error-content'
                };
            }
            
            // For unlimited shifts, always return primary style
            if (shift.required === -1) {
                return {
                    background: 'bg-primary',
                    text: 'text-primary-content'
                };
            }
            
            const ratio = shift.participants.length / shift.required;
            
            if (shift.participants.length === 0) {
                return {
                    background: 'bg-error',
                    text: 'text-error-content'
                };
            }
            if (shift.participants.length === shift.required) {
                return {
                    background: 'bg-success',
                    text: 'text-success-content'
                };
            }
            if (ratio < 0.5) {
                return {
                    background: 'bg-warning',
                    text: 'text-warning-content'
                };
            }
            return {
                background: 'bg-primary',
                text: 'text-primary-content'
            };
        },

        // Collision detection functions
        shiftsOverlap(shift1, shift2) {
            if (!shift1 || !shift2 || !shift1.start_time || !shift2.start_time) {
                return false;
            }

            const date1 = new Date(shift1.start_time);
            const date2 = new Date(shift2.start_time);

            // Check if shifts are on the same day
            const date1Str = `${date1.getFullYear()}-${String(date1.getMonth() + 1).padStart(2, '0')}-${String(date1.getDate()).padStart(2, '0')}`;
            const date2Str = `${date2.getFullYear()}-${String(date2.getMonth() + 1).padStart(2, '0')}-${String(date2.getDate()).padStart(2, '0')}`;
            
            // If shifts are on different days, they cannot overlap
            if (date1Str !== date2Str) {
                return false;
            }

            const start1 = date1.getTime();
            const end1 = start1 + (shift1.duration || 0) * 60000; // duration in minutes, convert to ms
            const start2 = date2.getTime();
            const end2 = start2 + (shift2.duration || 0) * 60000;

            // Two shifts overlap if: start1 < end2 AND start2 < end1
            return start1 < end2 && start2 < end1;
        },

        detectCollisions() {
            // Reset collisions map
            this.participantCollisions = {};

            // Collect all shifts from all types
            const allShifts = [];
            this.shiftTypes.forEach(type => {
                if (type.shifts && Array.isArray(type.shifts)) {
                    type.shifts.forEach(shift => {
                        if (shift.participants && Array.isArray(shift.participants)) {
                            allShifts.push(shift);
                        }
                    });
                }
            });

            // Group participants by name and collect their shifts
            const participantShifts = {};
            allShifts.forEach(shift => {
                shift.participants.forEach(participant => {
                    const name = participant.name;
                    if (!participantShifts[name]) {
                        participantShifts[name] = [];
                    }
                    participantShifts[name].push(shift);
                });
            });

            // Detect collisions for each participant
            Object.keys(participantShifts).forEach(name => {
                const shifts = participantShifts[name];
                const collisions = [];

                // Check all pairs of shifts for this participant
                for (let i = 0; i < shifts.length; i++) {
                    for (let j = i + 1; j < shifts.length; j++) {
                        if (this.shiftsOverlap(shifts[i], shifts[j])) {
                            // Add both shift IDs to collisions if not already present
                            if (!collisions.includes(shifts[i].id)) {
                                collisions.push(shifts[i].id);
                            }
                            if (!collisions.includes(shifts[j].id)) {
                                collisions.push(shifts[j].id);
                            }
                        }
                    }
                }

                // Only store collisions if there are any
                if (collisions.length > 0) {
                    this.participantCollisions[name] = collisions;
                }
            });
        },

        hasCollision(participantName) {
            if (!participantName) return false;
            return Array.isArray(this.participantCollisions[participantName]) && 
                   this.participantCollisions[participantName].length > 0;
        },

        getCollidingShifts(participantName) {
            if (!this.hasCollision(participantName)) {
                return [];
            }
            return this.participantCollisions[participantName] || [];
        },

        // Helper for shift type progress calculation and styling
        getShiftTypeProgress(type) {
            if (!type.shifts || type.shifts.length === 0) {
                return {
                    openShifts: 0,
                    totalRequired: 0,
                    totalFilled: 0,
                    percentage: 0,
                    hasUnlimitedShifts: false,
                    background: 'bg-base-content/10'
                };
            }

            let totalRequired = 0;
            let totalFilled = 0;
            let openShifts = 0;
            let hasUnlimitedShifts = false;
            let limitedShiftsCount = 0;

            type.shifts.forEach(shift => {
                const participantCount = shift.participants?.length || 0;
                
                if (shift.required === -1) {
                    hasUnlimitedShifts = true;
                    totalFilled += participantCount;
                } else {
                    limitedShiftsCount++;
                    totalRequired += shift.required;
                    totalFilled += participantCount;
                    openShifts += Math.max(0, shift.required - participantCount);
                }
            });

            // Calculate percentage based only on limited shifts
            let percentage = limitedShiftsCount > 0 ? Math.round((totalFilled / totalRequired) * 100) : 100;
            
            // Color based on limited shifts only
            let background;
            if (limitedShiftsCount === 0) {
                // If there are only unlimited shifts, show as success
                background = 'progress-accent/70';
            } else if (openShifts === 0) {
                background = 'progress-success';
            } else if (percentage < 50) {
                background = 'progress-error';
            } else {
                background = 'progress-warning';
            }

            return {
                openShifts,
                totalRequired,
                totalFilled,
                percentage,
                background,
                hasUnlimitedShifts
            };
        },

        // Admin functions
        openAdminModal() {
            this.adminModal.open();
        },

        handleAdminSuccess(token) {
            // Store admin state
            this.isAdmin = true;
            this.adminToken = token;

            // Save admin token in localStorage
            localStorage.setItem(`admin_token_${this.plan.token}`, token);

            // Debug-Ausgabe für Fehlersuche
            console.log('Admin login successful, token set:', token ? '[REDACTED]' : 'empty');

            // Update all modals with the new admin token
            this.shiftModal = new ShiftModal(
                this.plan.start_date,
                this.shiftTypes,
                this.plan.token,
                this.adminToken,
                this.handleShiftsUpdate.bind(this)
            );
            this.shiftTypeModal = new ShiftTypeModal(
                this.plan.is_multilingual,
                this.plan.token,
                this.adminToken,
                this.handleShiftTypeSuccess.bind(this)
            );
            this.deleteTypeModal = new DeleteTypeModal(
                this.plan.token,
                this.adminToken,
                this.handleDeleteTypeSuccess.bind(this)
            );
            this.deleteShiftModal = new DeleteShiftModal(
                this.plan.token,
                this.adminToken,
                this.handleDeleteShiftSuccess.bind(this)
            );
            this.sortTypesModal = new SortTypesModal(
                this.plan.token,
                this.adminToken,
                this.handleSortTypesSuccess.bind(this)
            );
            this.settingsModal = new SettingsModal(
                this.plan.token,
                this.adminToken,
                this.handleSettingsSuccess.bind(this)
            );
            this.exportModal = new ExportModal(
                this.plan.token,
                this.adminToken,
                this.filterManager
            );

            // Reload data to get admin-only fields
            this.loadPlan(this.plan.token);
        },

        logoutAdmin() {
            // Ask for confirmation before logging out
            const confirmMessage = this.$locale() === 'de' 
                ? 'Möchten Sie sich wirklich als Administrator abmelden?' 
                : 'Do you really want to log out as administrator?';
            
            if (!confirm(confirmMessage)) {
                return;
            }

            this.isAdmin = false;
            this.adminToken = '';

            // Remove admin token from localStorage
            localStorage.removeItem(`admin_token_${this.plan.token}`);

            // Reload data to remove admin-only fields
            this.loadPlan(this.plan.token);
        },

        // Shift Type Modal functions
        openShiftTypeModal(type = null) {
            this.shiftTypeModal.openWith(type);
        },

        handleShiftTypeSuccess(type) {
            // If editing an existing type, update it in the list
            const existingIndex = this.shiftTypes.findIndex(t => t.id === type.id);
            if (existingIndex !== -1) {
                // Preserve the shifts when updating
                type.shifts = this.shiftTypes[existingIndex].shifts || [];
                type.shiftsByDate = this.groupShiftsByDate(type.shifts);
                this.shiftTypes.splice(existingIndex, 1, type);
            } else {
                // Add new type to the list
                type.shifts = [];
                type.shiftsByDate = [];
                this.shiftTypes.push(type);
            }
        },

        // Shift Modal functions
        openShiftModal(shift = null, type = null, isDuplicating = false) {
            this.shiftModal.openWith(shift, type, isDuplicating);
        },

        // Delete Type Modal functions
        openDeleteShiftTypeModal(type) {
            this.deleteTypeModal.openWith(type);
        },

        handleDeleteTypeSuccess(type) {
            // Remove the type from the list
            const index = this.shiftTypes.findIndex(t => t.id === type.id);
            if (index !== -1) {
                this.shiftTypes.splice(index, 1);
            }
        },

        // Delete Shift Modal functions
        openDeleteShiftModal(shift, type) {
            this.deleteShiftModal.openWith(shift, type);
        },

        handleDeleteShiftSuccess(shift, type) {
            // Remove the shift from its type
            const typeIndex = this.shiftTypes.findIndex(t => t.id === type.id);
            if (typeIndex !== -1) {
                const shiftIndex = this.shiftTypes[typeIndex].shifts.findIndex(s => s.id === shift.id);
                if (shiftIndex !== -1) {
                    this.shiftTypes[typeIndex].shifts.splice(shiftIndex, 1);
                    // Update the shiftsByDate grouping
                    this.shiftTypes[typeIndex].shiftsByDate = this.groupShiftsByDate(this.shiftTypes[typeIndex].shifts);
                }
            }

            // Recalculate collisions after shift deletion
            this.detectCollisions();
        },

        // Helper function to update a shift in the shiftTypes array
        handleShiftUpdate(updatedShift) {
            // Find the type that contains this shift
            const typeIndex = this.shiftTypes.findIndex(type =>
                type.shifts.some(shift => shift.id === updatedShift.id)
            );

            if (typeIndex !== -1) {
                // Update the shift in the type's shifts array
                const shiftIndex = this.shiftTypes[typeIndex].shifts.findIndex(
                    shift => shift.id === updatedShift.id
                );

                if (shiftIndex !== -1) {
                    this.shiftTypes[typeIndex].shifts[shiftIndex] = updatedShift;
                    // Update the shiftsByDate grouping
                    this.shiftTypes[typeIndex].shiftsByDate = this.groupShiftsByDate(this.shiftTypes[typeIndex].shifts);
                }
            }

            // Recalculate collisions after shift update (participants may have changed)
            this.detectCollisions();
            
            // Re-detect overflow after participants change (they may wrap differently now)
            this.$nextTick(() => {
                const containerEl = document.querySelector(`[data-shift-id="${updatedShift.id}"]`);
                if (containerEl) {
                    this.detectShiftOverflow(updatedShift.id, containerEl);
                }
            });
        },

        // Add new method to handle shift updates
        handleShiftsUpdate(shifts) {
            // Update or add the shifts to their respective types
            shifts.forEach(shift => {
                const typeIndex = this.shiftTypes.findIndex(type => type.id === shift.type_id);
                if (typeIndex === -1) return;

                const type = this.shiftTypes[typeIndex];
                type.shifts = type.shifts || [];

                // Find and update existing shift or add new one
                const shiftIndex = type.shifts.findIndex(s => s.id === shift.id);
                if (shiftIndex !== -1) {
                    type.shifts[shiftIndex] = { ...shift };
                } else {
                    type.shifts.push(shift);
                }

                // Sort shifts by start time
                type.shifts.sort((a, b) => new Date(a.start_time) - new Date(b.start_time));
            });

            // Recalculate collisions after shifts are updated
            this.detectCollisions();
            
            // Re-detect overflow for all updated shifts after layout is complete
            this.$nextTick(() => {
                shifts.forEach(shift => {
                    const containerEl = document.querySelector(`[data-shift-id="${shift.id}"]`);
                    if (containerEl) {
                        this.detectShiftOverflow(shift.id, containerEl);
                    }
                });
            });
        },

        // Open sort types modal
        openSortTypesModal() {
            this.sortTypesModal.openWith(this.shiftTypes);
        },

        // Handle successful type sorting
        handleSortTypesSuccess() {
            // Reload types from server to get updated sort_order
            const headers = {};
            if (this.isAdmin && this.adminToken) {
                headers['X-Admin-Token'] = this.adminToken;
            }

            fetch(`/api/plans/${this.plan.token}/types`, { headers })
                .then(response => {
                    if (!response.ok) throw new Error('Failed to load shift types');
                    return response.json();
                })
                .then(types => {
                    // Update types with their shifts
                    this.shiftTypes = types.map(type => ({
                        ...type,
                        shifts: type.shifts || [],
                        shiftsByDate: this.groupShiftsByDate(type.shifts || [])
                    }));

                    // Recalculate collisions after reloading types
                    this.detectCollisions();
                })
                .catch(error => {
                    console.error('Error reloading types:', error);
                });
        },

        openSettingsModal() {
            this.settingsModal.openWith(this.plan);
        },

        handleSettingsSuccess(updatedPlan) {
            // Update the plan data in a way that triggers Alpine.js reactivity
            Object.keys(updatedPlan).forEach(key => {
                this.plan[key] = updatedPlan[key];
            });

            // Update modals that depend on plan settings
            this.shiftModal = new ShiftModal(
                this.plan.start_date,
                this.shiftTypes,
                this.plan.token,
                this.adminToken,
                this.handleShiftsUpdate.bind(this)
            );
            this.shiftTypeModal = new ShiftTypeModal(
                this.plan.is_multilingual,
                this.plan.token,
                this.adminToken,
                this.handleShiftTypeSuccess.bind(this)
            );

            // Reload the plan data to ensure everything is up to date
            this.loadPlan(this.plan.token);
        },

        openExportModal() {
            this.exportModal.open();
        },

        closeExportModal() {
            this.exportModal.close();
        },

        async downloadExport() {
            await this.exportModal.downloadExport();
        },

        // Lock/Unlock shift functions
        async lockShift(shift, type) {
            if (!this.adminToken) {
                console.error('Cannot lock shift: No admin token');
                return;
            }

            const url = `/api/plans/${this.plan.token}/shifts/${shift.id}/lock`;
            try {
                const response = await fetch(url, {
                    method: 'POST',
                    headers: {
                        'X-Admin-Token': this.adminToken
                    }
                });

                if (!response.ok) {
                    throw new Error(AlpineI18n.t('shift_plans.view.error_locking_shift'));
                }

                // Optimistically update the UI
                shift.is_locked = true;
            } catch (error) {
                console.error('Error locking shift:', error);
                alert(AlpineI18n.t('shift_plans.view.error_locking_shift'));
            }
        },

        async unlockShift(shift, type) {
            if (!this.adminToken) {
                console.error('Cannot unlock shift: No admin token');
                return;
            }

            const url = `/api/plans/${this.plan.token}/shifts/${shift.id}/unlock`;
            try {
                const response = await fetch(url, {
                    method: 'POST',
                    headers: {
                        'X-Admin-Token': this.adminToken
                    }
                });

                if (!response.ok) {
                    throw new Error(AlpineI18n.t('shift_plans.view.error_unlocking_shift'));
                }

                // Optimistically update the UI
                shift.is_locked = false;
            } catch (error) {
                console.error('Error unlocking shift:', error);
                alert(AlpineI18n.t('shift_plans.view.error_unlocking_shift'));
            }
        }
    }));
}); 