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
import { timetableViewState } from './timetable-view-state.js';
import {
    parseDateTime, serializeDateTime, addMinutes as addWallMinutes, dateKey as toDateKey, logicalDay,
    formatTime as formatWallTime, formatDate as formatWallDate
} from './datetime.js';
import { parseBoundaryTime } from './timetable-view-layout.js';
import { PlanRealtime } from './realtime.js';

/** Adds the properties of `mixin` to `view`, keeping getters and setters (a spread would evaluate them). */
function composeView(view, mixin) {
    return Object.defineProperties(view, Object.getOwnPropertyDescriptors(mixin));
}

document.addEventListener('alpine:init', () => {
    Alpine.data('shiftPlanView', () => composeView({
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

        // Collision detection state
        participantCollisions: {}, // Map: participant name -> array of colliding shift IDs

        // URL deep link state (read once on first load)
        urlFiltersPending: true,
        pendingUrlShiftId: null,

        // Realtime state
        realtime: null,
        realtimeStatus: null, // null until the first status report, then live|connecting|offline
        realtimeEverLive: false,
        viewers: 0,
        recentlyChanged: new Set(), // shift ids whose participants changed, highlighted for 1.5 s

        get realtimeVisible() {
            // Hidden when the endpoint is disabled (offline without ever having been live)
            return this.realtimeStatus !== null && (this.realtimeStatus !== 'offline' || this.realtimeEverLive);
        },

        get realtimeStatusText() {
            return this.realtimeStatus ? this.$t(`realtime.status_${this.realtimeStatus}`) : '';
        },

        get viewersText() {
            return this.viewers === 1
                ? this.$t('realtime.viewers_one')
                : this.$t('realtime.viewers_other', { count: this.viewers });
        },

        // Computed property für den Ladezustand
        get isReady() {
            return !this.isLoading && !this.error && this.plan;
        },

        get isTimetableView() {
            return this.viewMode === 'timetable';
        },

        get isChronoView() {
            return this.viewMode === 'chrono';
        },

        /** Views offered by the switcher; labels shown only for the active one. */
        viewModes: [
            { id: 'list', labelKey: 'shift_plans.view.view_mode_list', icon: 'M4 6h16M4 12h16M4 18h16' },
            { id: 'chrono', labelKey: 'shift_plans.view.view_mode_chrono', icon: 'M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12' },
            { id: 'timetable', labelKey: 'shift_plans.view.view_mode_timetable', icon: 'M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z' },
        ],

        isValidViewMode(mode) {
            return this.viewModes.some(v => v.id === mode);
        },

        /**
         * All filtered shifts of all types in one chronological sequence, grouped by logical day.
         * Same shape as type.shiftsByDate ({ date, shifts }), so shift-row can render it unchanged.
         * Equal start times keep the order of the shift types.
         */
        get chronoDays() {
            const boundaryMinutes = parseBoundaryTime(this.plan?.day_boundary_time || '00:00');
            const entries = [];
            (this.filteredShiftTypes || []).forEach(type => {
                (type.shifts || []).forEach(shift => entries.push({ shift, order: type.sort_order ?? 0 }));
            });
            // Floating ISO strings without offset sort lexicographically
            entries.sort((a, b) => a.shift.start_time.localeCompare(b.shift.start_time) || a.order - b.order);

            const days = [];
            entries.forEach(({ shift }) => {
                const start = parseDateTime(shift.start_time);
                if (!start) return;
                const date = toDateKey(logicalDay(start, boundaryMinutes));
                let day = days[days.length - 1];
                if (!day || day.date !== date) {
                    day = { date, shifts: [] };
                    days.push(day);
                }
                day.shifts.push(shift);
            });
            return days;
        },

        /** Title of a shift type in the current language. */
        typeTitle(type) {
            if (!type) return '';
            return this.plan?.is_multilingual && this.$locale() === 'de' ? type.title_de : type.title_en;
        },

        /** The (unfiltered) shift type a shift belongs to. */
        typeOfShift(shift) {
            return this.shiftTypes.find(t => t.id === shift.type_id);
        },

        // Computed property für verfügbare Tage
        get availableDays() {
            this.filterManager.setDayBoundary(this.plan?.day_boundary_time);
            return this.filterManager.getAvailableDays(this.shiftTypes);
        },

        // Computed property für gefilterte Schichttypen
        get filteredShiftTypes() {
            this.filterManager.setDayBoundary(this.plan?.day_boundary_time);
            return this.filterManager.filterShiftTypes(this.shiftTypes);
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
            this.initTimetableModel();

            // Filters and target shift from the URL, read once; the URL is not written back
            const urlParams = new URLSearchParams(window.location.search);
            this.filterManager.applyUrlFilters(urlParams);
            const shiftParam = urlParams.get('shift') || '';
            this.pendingUrlShiftId = /^\d+$/.test(shiftParam) ? shiftParam : null;

            this.shiftModal = new ShiftModal(
                this.plan.start_date,
                this.shiftTypes,
                token,
                this.adminToken,
                this.handleShiftsUpdate.bind(this),
                this.plan.day_boundary_time
            );
            this.shiftTypeModal = new ShiftTypeModal(this.plan.is_multilingual, token, this.adminToken, this.handleShiftTypeSuccess.bind(this));
            this.deleteTypeModal = new DeleteTypeModal(token, this.adminToken, this.handleDeleteTypeSuccess.bind(this));
            this.deleteShiftModal = new DeleteShiftModal(token, this.adminToken, this.handleDeleteShiftSuccess.bind(this));
            this.joinModal = new JoinModal(token, this.handleShiftUpdate.bind(this), this.adminToken, () => this.refreshData('participants'));
            this.removeModal = new RemoveModal(token, this.handleShiftUpdate.bind(this), this.adminToken);
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

            // Resolve the server-configured default from the rendered attribute
            const el = document.querySelector('[data-default-view]');
            const raw = el?.dataset?.defaultView;
            const configDefault = this.isValidViewMode(raw) ? raw : 'timetable';

            // Cache-busting: if the server default changed since the preference was saved,
            // discard the saved preference so the new default takes effect.
            const storedDefaultKey = `shift_view_mode_default_${token}`;
            const storedDefault = localStorage.getItem(storedDefaultKey);
            if (storedDefault !== configDefault) {
                localStorage.removeItem(`shift_view_mode_${token}`);
                localStorage.setItem(storedDefaultKey, configDefault);
            }

            if (this.isValidViewMode(requested)) {
                this.viewMode = requested;
                localStorage.setItem(`shift_view_mode_${token}`, requested);
                return;
            }

            const stored = localStorage.getItem(`shift_view_mode_${token}`);
            if (this.isValidViewMode(stored)) {
                this.viewMode = stored;
                this.syncViewModeQuery(stored);
                return;
            }

            this.viewMode = configDefault;
            this.syncViewModeQuery(configDefault);
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
            if (!this.isValidViewMode(mode)) return;
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
                    this.detectAllShiftOverflows();
                }
            });
        },

        // Gruppiere Schichten nach Datum
        groupShiftsByDate(shifts) {
            const groups = {};

            // Sortiere Schichten nach Startzeit
            // Floating ISO strings without offset sort lexicographically
            shifts.sort((a, b) => a.start_time.localeCompare(b.start_time));

            // Group by logical day (times before the day boundary belong to the previous day)
            const boundaryMinutes = parseBoundaryTime(this.plan?.day_boundary_time || '00:00');
            shifts.forEach(shift => {
                const start = parseDateTime(shift.start_time);
                if (!start) return;
                const dateKey = toDateKey(logicalDay(start, boundaryMinutes));

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

        // Realtime: connect once after the first successful load
        startRealtime(token) {
            if (this.realtime) return;
            this.realtime = new PlanRealtime(token, {
                onChange: (scope) => this.handleRealtimeChange(scope),
                onStatus: (status) => {
                    this.realtimeStatus = status;
                    if (status === 'live') this.realtimeEverLive = true;
                },
                onViewers: (count) => { this.viewers = count; }
            });
            this.realtime.start();
            window.addEventListener('pagehide', () => this.realtime?.stop());
            window.addEventListener('pageshow', (e) => { if (e.persisted) this.realtime?.start(); });
        },

        handleRealtimeChange(scope) {
            if (scope === 'deleted') {
                window.location.href = '/plan-deleted';
                return;
            }
            return this.refreshData(scope);
        },

        /**
         * Re-fetch only what changed and swap the data in place. Modals are not
         * recreated; filters, scroll position and open dialogs stay as they are.
         * scope 'plan' also reloads the plan itself, everything else only the types.
         */
        async refreshData(scope) {
            const token = this.planToken;
            const headers = {};
            if (this.isAdmin && this.adminToken) {
                headers['X-Admin-Token'] = this.adminToken;
            }

            let plan = null;
            if (scope === 'plan') {
                const response = await fetch(`/api/plans/${token}`, { headers });
                if (!response.ok) throw new Error('Failed to load plan');
                plan = await response.json();
            }
            const typesResponse = await fetch(`/api/plans/${token}/types`, { headers });
            if (!typesResponse.ok) throw new Error('Failed to load shift types');
            const types = await typesResponse.json();

            const before = this.participantSignatures();

            if (plan) {
                this.plan = { ...plan, day_boundary_time: plan.day_boundary_time || '00:00' };
                if (this.shiftTypeModal) this.shiftTypeModal.isMultilingual = this.plan.is_multilingual;
            }
            this.shiftTypes = this.prepareTypes(types);
            // Modals read shiftTypes when they are used; point the shift modal at the new array
            if (this.shiftModal) this.shiftModal.shiftTypes = this.shiftTypes;

            this.markChangedShifts(before);
            this.rebindShiftDialogs();
            this.detectCollisions();
            this.$nextTick(() => this.detectAllShiftOverflows());
        },

        findShift(shiftId) {
            for (const type of this.shiftTypes) {
                const found = type.shifts?.find(s => s.id === shiftId);
                if (found) return found;
            }
            return null;
        },

        /**
         * Prepares API types for the view. Declined participations are moved out of
         * shift.participants into shift.declinedParticipants, so everything that counts
         * or lists participants (occupancy, filters, collisions, timetable) sees only the ones
         * that occupy a place. Declined ones are shown on request (filterManager.showDeclined).
         */
        prepareTypes(types) {
            return types.map(type => {
                const shifts = (type.shifts || []).map(shift => {
                    const all = shift.participants || [];
                    const declinedParticipants = all.filter(p => p.status === 'declined');
                    return declinedParticipants.length
                        ? { ...shift, participants: all.filter(p => p.status !== 'declined'), declinedParticipants }
                        : { ...shift, declinedParticipants: [] };
                });
                return { ...type, shifts, shiftsByDate: this.groupShiftsByDate(shifts) };
            });
        },

        /** Label of a person field or of a choice option in the current language. */
        fieldLabel(f) {
            const de = window.AlpineI18n?.locale !== 'en';
            return (de ? f.label_de : f.label_en) || f.label_en || f.label_de || '';
        },

        /** Options of the plan's experience field (empty if none is set). */
        experienceOptions() {
            const field = (this.plan?.fields || []).find(f => f.id === this.plan.experience_field_id);
            return field ? field.options : [];
        },

        optionLabel(o) {
            const de = window.AlpineI18n?.locale !== 'en';
            return (de ? o.de : o.en) || o.en || o.de || '';
        },

        /** Whether any participation has a status other than signed up (status icons appear only then). */
        get planHasStatuses() {
            return this.shiftTypes.some(t => t.shifts.some(s =>
                (s.declinedParticipants?.length || 0) > 0 || (s.participants || []).some(p => p.status === 'assigned')));
        },

        /** Whether declined participations are loaded at all (the filter switch appears only then). */
        get hasDeclined() {
            return this.shiftTypes.some(t => t.shifts.some(s => (s.declinedParticipants?.length || 0) > 0));
        },

        // Map: shift id -> participant signature
        participantSignatures() {
            const map = new Map();
            for (const type of this.shiftTypes) {
                for (const shift of type.shifts || []) {
                    map.set(shift.id, (shift.participants || []).map(p => `${p.id}:${p.name}`).sort().join('|'));
                }
            }
            return map;
        },

        markChangedShifts(before) {
            const changed = [];
            for (const [id, signature] of this.participantSignatures()) {
                if (before.has(id) && before.get(id) !== signature) changed.push(id);
            }
            if (changed.length === 0) return;
            this.recentlyChanged = new Set([...this.recentlyChanged, ...changed]);
            setTimeout(() => {
                const next = new Set(this.recentlyChanged);
                changed.forEach(id => next.delete(id));
                this.recentlyChanged = next;
            }, 1500);
        },

        isRecentlyChanged(shiftId) {
            return this.recentlyChanged.has(shiftId);
        },

        // Open join/remove dialogs keep working on the current shift object
        rebindShiftDialogs() {
            for (const modal of [this.joinModal, this.removeModal]) {
                const fresh = modal?.currentShift ? this.findShift(modal.currentShift.id) : null;
                if (fresh) modal.currentShift = fresh;
            }
        },

        /** 'deleted' | 'full' | null: why the shift of the open dialog can no longer be used. */
        shiftDialogNotice(modal, checkFull) {
            const current = modal?.currentShift;
            if (!current) return null;
            const shift = this.findShift(current.id);
            if (!shift) return 'deleted';
            if (checkFull && !this.plan?.collaborative_mode
                && typeof shift.required === 'number' && shift.required !== -1
                && (shift.participants?.length || 0) >= shift.required) {
                return 'full';
            }
            return null;
        },

        get joinNotice() {
            return this.shiftDialogNotice(this.joinModal, true);
        },

        get removeNotice() {
            return this.shiftDialogNotice(this.removeModal, false);
        },

        noticeText(notice) {
            return notice === 'deleted' ? this.$t('realtime.shift_deleted') : this.$t('realtime.shift_now_full');
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
                this.shiftTypes = this.prepareTypes(types);

                // First load only: drop URL type ids that do not exist in this plan
                if (this.urlFiltersPending) {
                    const knownTypeIds = this.shiftTypes.map(t => t.id);
                    this.filterManager.selectedTypes = this.filterManager.selectedTypes.filter(id => knownTypeIds.includes(id));
                    this.urlFiltersPending = false;
                }

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
                    this.handleShiftsUpdate.bind(this),
                    this.plan.day_boundary_time
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
                    this.handleShiftUpdate.bind(this),
                    this.adminToken,
                    () => this.refreshData('participants')
                );
                this.removeModal = new RemoveModal(
                    this.plan.token,
                    this.handleShiftUpdate.bind(this),
                    this.adminToken
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

                this.startRealtime(token);

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
                this.$nextTick(() => this.detectAllShiftOverflows());
                if (this.pendingUrlShiftId && !this.error) {
                    this.$nextTick(() => this.focusUrlShift());
                }
            }
        },

        // Scroll to the shift from ?shift= and focus it; runs once after the first load
        focusUrlShift() {
            const id = this.pendingUrlShiftId;
            this.pendingUrlShiftId = null;
            if (!id) return;

            // Timetable cells and list rows both carry data-shift-id; take the visible one.
            const el = [...document.querySelectorAll(`[data-shift-id="${id}"]`)]
                .find(e => e.getClientRects().length > 0);
            if (!el) return;

            if (!el.hasAttribute('tabindex')) el.setAttribute('tabindex', '-1');
            const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
            el.scrollIntoView({ block: 'center', behavior: reduceMotion ? 'auto' : 'smooth' });
            el.focus({ preventScroll: true });
        },

        // Modal functions
        openJoinModal(shift) {
            if (!shift) return;
            this.joinModal.openWith(shift, this.plan);
        },

        openRemoveModal(shift, participant) {
            if (!shift || !participant) return;
            if (shift.is_locked && !this.isAdmin) return;
            this.removeModal.openWith(shift, participant);
        },

        // Formatierungshelfer mit Null-Checks
        formatDate(dateStr) {
            if (!dateStr) return '';
            const formatted = formatWallDate(dateStr, this.$locale(), {
                weekday: 'long',
                year: 'numeric',
                month: 'long',
                day: 'numeric'
            });

            // For German: Capitalize first letter
            if (formatted && this.$locale() === 'de') {
                return formatted.charAt(0).toUpperCase() + formatted.slice(1);
            }
            return formatted;
        },

        formatTime(dateStr, duration) {
            const start = parseDateTime(dateStr);
            if (!start) return '';
            const end = addWallMinutes(start, duration || 0);
            return `${formatWallTime(start, this.$locale())} – ${formatWallTime(end, this.$locale())}`;
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

        // time: API string "YYYY-MM-DDTHH:MM:SS" or floating Date
        formatTimeOnly(time) {
            const date = time instanceof Date ? time : parseDateTime(time);
            return date ? formatWallTime(date, this.$locale()) : '';
        },

        // Returns the API string moved by the given minutes
        addMinutes(time, minutes) {
            const date = parseDateTime(time);
            return date ? serializeDateTime(addWallMinutes(date, minutes)) : '';
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

            const date1 = parseDateTime(shift1.start_time);
            const date2 = parseDateTime(shift2.start_time);
            if (!date1 || !date2) return false;

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

        // True only if this participant collides in this specific shift
        hasCollision(participantName, shiftId) {
            if (!participantName) return false;
            const list = this.participantCollisions[participantName];
            return Array.isArray(list) && list.includes(shiftId);
        },

        // True if any participant has an overlap in this shift
        shiftHasCollision(shiftId) {
            return Object.values(this.participantCollisions).some(
                list => Array.isArray(list) && list.includes(shiftId)
            );
        },

        getCollidingShifts(participantName) {
            const list = this.participantCollisions[participantName];
            return Array.isArray(list) ? list : [];
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
                this.handleShiftsUpdate.bind(this),
                this.plan.day_boundary_time
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
            if (this.joinModal) this.joinModal.adminToken = token;
            if (this.removeModal) this.removeModal.adminToken = token;

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
            if (this.joinModal) this.joinModal.adminToken = '';
            if (this.removeModal) this.removeModal.adminToken = '';

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
            
            this.$nextTick(() => this.detectAllShiftOverflows());
        },

        // Add new method to handle shift updates
        handleShiftsUpdate(shifts) {
            const touchedTypeIndices = new Set();
            // Update or add the shifts to their respective types
            shifts.forEach(shift => {
                const typeIndex = this.shiftTypes.findIndex(type => type.id === shift.type_id);
                if (typeIndex === -1) return;
                touchedTypeIndices.add(typeIndex);

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
                type.// Floating ISO strings without offset sort lexicographically
            shifts.sort((a, b) => a.start_time.localeCompare(b.start_time));
            });

            touchedTypeIndices.forEach(typeIndex => {
                const t = this.shiftTypes[typeIndex];
                t.shiftsByDate = this.groupShiftsByDate(t.shifts || []);
            });

            // Recalculate collisions after shifts are updated
            this.detectCollisions();

            this.$nextTick(() => this.detectAllShiftOverflows());
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
                    this.shiftTypes = this.prepareTypes(types);

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
                this.handleShiftsUpdate.bind(this),
                this.plan.day_boundary_time
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
    }, timetableViewState()));
}); 