/**
 * Coordination page (admin only): shared state of the three tabs.
 *
 * This component loads the plan, the shift types with participations of every status and the stored
 * person data, builds the CoordModel (compute.js) and applies changes through POST
 * /api/plans/{token}/coordination/apply. The tabs have their own nested components
 * (distribute.js, auto.js, people.js, settings.js) that read this scope and call apply().
 *
 * No fairness or rule logic lives here or in the templates; everything is read from `model`.
 */
import { formatDate, formatTime, parseDate } from '../datetime.js';
import { PlanRealtime } from '../realtime.js';
import { personKey } from '../stats/compute.js';
import { ShiftModal } from '../shift-modal.js';
import { ShiftTypeModal } from '../shift-type-modal.js';
import {
    buildModel, rankCandidates, concurrentShifts, buildTimeline, buildPersonTimeline,
    applyOpsLocally, invertOps, newLottery,
} from './compute.js';

const TABS = ['distribute', 'people', 'lab', 'settings'];
const VIEWS = ['list', 'timeline', 'people'];
const WIDE_QUERY = '(min-width: 1024px)'; // Tailwind lg: timeline and people views need the room
const STATUS_FILTERS = ['overbooked', 'open', 'full', 'experience'];
const TOAST_MS = 10000;

document.addEventListener('alpine:init', () => {
    Alpine.data('coordinationPage', () => ({
        token: '',
        adminToken: '',
        plan: null,
        types: [],
        shiftModal: null, // edit dialogs of the plan page, opened from the lab
        shiftTypeModal: null,
        people: [],
        model: null,
        draws: new Map(),      // shiftId → Map<personKey, number>: the lot drawn for a tie in that shift (not stored)
        isLoading: true,
        denied: false,
        error: null,
        realtime: null,

        tabs: TABS,
        tabPrefix: 'coord',
        tab: 'distribute',

        // Distribute tab (shared with its nested components)
        filter: { typeIds: [], days: [], status: '' },
        selectedShiftId: null,
        source: 'interested',
        candidates: { ranked: [], unavailable: [] }, // ranked[].impacts / .level come from rank.js
        concurrent: [], // shifts overlapping the selected one (CoordShift[]), for the "concurrent" bar
        shiftList: [],

        // Distribute view: 'list' | 'timeline' | 'people' (the latter two only from lg up)
        view: 'list',
        timelineDay: '', // logical day (YYYY-MM-DD) of the two time views; '' = derived (see currentTimelineDay)
        isWide: false,

        // Lab tab (state of the lab itself lives in lab-page.js): logical day shown by the lab grid, '' = derived
        labDay: '',

        // Changes in flight and undo
        busy: false,
        /** Inline level saving per person key: { saving: boolean, error: string } */
        levelState: {},
        pendingKey: null, // person key or other id of the action in flight (for the loading state)
        actionError: null,
        // Error of the open modal; same shape as the other modals so the error-display partial works
        modalError: {
            hasError: false,
            errorTitle: '',
            errorDetails: null,
            showTechnicalDetails: false,
            copyErrorDetails() {
                navigator.clipboard?.writeText(JSON.stringify(this.errorDetails, null, 2));
            },
        },
        toast: null,
        toastTimer: null,
        lastUndo: null,

        // Auto distribution preview (set by auto.js); stale when the plan changed meanwhile
        preview: null,
        previewStale: false,

        init() {
            const parts = window.location.pathname.split('/');
            this.token = parts[parts.length - 2];
            try {
                this.adminToken = localStorage.getItem(`admin_token_${this.token}`) || '';
            } catch (e) {
                this.adminToken = '';
            }
            this.readUrl();
            const mq = window.matchMedia(WIDE_QUERY);
            this.isWide = mq.matches;
            mq.addEventListener('change', (e) => { this.isWide = e.matches; });
            if (!this.adminToken) {
                this.isLoading = false;
                this.denied = true;
                return;
            }
            this.load();
        },

        get isAdmin() {
            return !!this.adminToken && !this.denied;
        },

        // --- URL state (?tab=&type=&day=&status=&shift=&view=&tday=&lday=) ---

        readUrl() {
            const sp = new URLSearchParams(window.location.search);
            const tab = sp.get('tab');
            if (TABS.includes(tab)) this.tab = tab;
            const type = (sp.get('type') || '').split(',').map(Number).filter(n => Number.isInteger(n) && n > 0);
            const day = (sp.get('day') || '').split(',').filter(d => /^\d{4}-\d{2}-\d{2}$/.test(d));
            const status = sp.get('status');
            this.filter = { typeIds: type, days: day, status: STATUS_FILTERS.includes(status) ? status : '' };
            const shift = Number(sp.get('shift'));
            this.selectedShiftId = Number.isInteger(shift) && shift > 0 ? shift : null;
            const view = sp.get('view');
            this.view = VIEWS.includes(view) ? view : 'list';
            const tday = sp.get('tday') || '';
            this.timelineDay = /^\d{4}-\d{2}-\d{2}$/.test(tday) ? tday : '';
            const lday = sp.get('lday') || '';
            this.labDay = /^\d{4}-\d{2}-\d{2}$/.test(lday) ? lday : '';
        },

        writeUrl() {
            const sp = new URLSearchParams();
            if (this.tab !== 'distribute') sp.set('tab', this.tab);
            if (this.filter.typeIds.length) sp.set('type', this.filter.typeIds.join(','));
            if (this.filter.days.length) sp.set('day', this.filter.days.join(','));
            if (this.filter.status) sp.set('status', this.filter.status);
            if (this.selectedShiftId) sp.set('shift', String(this.selectedShiftId));
            if (this.view !== 'list') sp.set('view', this.view);
            if (this.view !== 'list' && this.timelineDay) sp.set('tday', this.timelineDay);
            if (this.tab === 'lab' && this.labDay) sp.set('lday', this.labDay);
            const q = sp.toString();
            history.replaceState(null, '', window.location.pathname + (q ? `?${q}` : ''));
        },

        // --- Server communication ---

        /**
         * Saves the experience level of a person inline (full person record, so nothing else is lost).
         * A person without stored row gets a new record with name and level only. Used by the people
         * tab and the person column of the distribute tab.
         */
        async setLevel(p, event) {
            const select = event.target;
            const fieldId = this.model.experience.field.id;
            const stored = p.level >= 0 ? this.model.experience.options[p.level].id : '';
            if (select.value === stored || this.busy) {
                select.value = stored;
                return;
            }
            const r = p.row;
            const values = { ...(r?.values || {}) };
            if (select.value) values[fieldId] = select.value;
            else delete values[fieldId];
            const body = {
                name: p.name,
                available_from: r?.available_from ?? null,
                available_until: r?.available_until ?? null,
                target_minutes: r?.target_minutes ?? null,
                prefers_long_days: !!r?.prefers_long_days,
                values,
            };
            this.levelState[p.key] = { saving: true, error: '' };
            this.busy = true;
            try {
                await this.api('PUT', '/people', body);
                this.levelState[p.key] = { saving: false, error: '' };
                await this.load(true);
            } catch (e) {
                select.value = stored;
                this.levelState[p.key] = { saving: false, error: e.message || this.$t('coord.error_action') };
            } finally {
                this.busy = false;
            }
        },

        /** Calls the API as admin. Throws an Error with .status on a failed response. */
        async api(method, path, body) {
            const res = await fetch(`/api/plans/${encodeURIComponent(this.token)}${path}`, {
                method,
                headers: {
                    'X-Admin-Token': this.adminToken,
                    ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
                },
                body: body !== undefined ? JSON.stringify(body) : undefined,
            });
            if (!res.ok) {
                const err = new Error((await res.text()).trim() || `HTTP ${res.status}`);
                err.status = res.status;
                throw err;
            }
            const type = res.headers.get('Content-Type') || '';
            return type.includes('application/json') ? res.json() : null;
        },

        /**
         * Loads plan, types and people and rebuilds the model.
         * silent: background reload (realtime): no spinner, selection and tab stay.
         */
        async load(silent = false) {
            if (!silent) {
                this.isLoading = true;
                this.error = null;
            }
            try {
                const [plan, types, people] = await Promise.all([
                    this.api('GET', ''),
                    this.api('GET', '/types'),
                    this.api('GET', '/people'),
                ]);
                this.plan = plan;
                this.types = types;
                this.people = people;
                this.rebuild();
                if (!this.realtime) this.startRealtime();
            } catch (e) {
                console.error('coordination: loading failed', e);
                if (e.status === 401) {
                    this.denied = true;
                } else if (!silent) {
                    this.error = e.message || String(e);
                }
            } finally {
                if (!silent) this.isLoading = false;
            }
        },

        /** Live updates: reload silently; an open preview becomes stale. */
        startRealtime() {
            this.realtime = new PlanRealtime(this.token, {
                onChange: (scope) => {
                    if (scope === 'deleted') {
                        this.realtime.stop();
                        window.location.href = '/plan-deleted';
                        return;
                    }
                    if (this.preview && !this.busy) this.previewStale = true;
                    return this.load(true);
                },
            });
            this.realtime.start();
        },

        // --- Model ---

        /** Rebuilds the model from the loaded data (and everything derived from it). */
        rebuild() {
            this.model = buildModel(this.plan, this.types, this.people);
            this.syncModals();
            if (this.selectedShiftId && !this.model.shifts.has(this.selectedShiftId)) this.selectedShiftId = null;
            this.refresh();
            // On a phone the shift is chosen with a select: start with the first one
            if (!this.selectedShiftId && this.shiftList.length && window.matchMedia('(max-width: 767px)').matches) {
                this.selectShift(this.shiftList[0].id);
            }
        },

        /** Creates the shift and shift type dialogs once and keeps their data current; saving reloads the page data. */
        syncModals() {
            if (!this.shiftModal) {
                const reload = () => this.load(true);
                this.shiftModal = new ShiftModal(this.plan.start_date, this.types, this.token, this.adminToken, reload, this.plan.day_boundary_time);
                this.shiftTypeModal = new ShiftTypeModal(this.plan.is_multilingual, this.token, this.adminToken, reload);
            }
            this.shiftModal.shiftTypes = this.types;
            this.shiftTypeModal.isMultilingual = this.plan.is_multilingual;
        },

        get shiftTypes() {
            return this.types;
        },

        openShiftModal(cs) {
            this.shiftModal.openWith(cs.shift, cs.type);
        },

        openTypeModal(type) {
            this.shiftTypeModal.openWith(type);
        },

        /** Options of the plan's experience field (empty if none is set), for the coordination part of the dialogs. */
        experienceOptions() {
            const field = (this.plan?.fields || []).find(f => f.id === this.plan.experience_field_id);
            return field ? field.options : [];
        },

        optionLabel(o) {
            return (this.lang() === 'en' ? o.en : o.de) || o.en || o.de || '';
        },

        /** Recomputes the filtered shift list and the candidates of the selected shift. */
        refresh() {
            if (!this.model) return;
            const { typeIds, days, status } = this.filter;
            this.shiftList = [...this.model.shifts.values()].filter(cs =>
                (!typeIds.length || typeIds.includes(cs.type.id)) &&
                (!days.length || days.includes(cs.day)) &&
                this.matchesStatus(cs, status));
            this.candidates = this.selectedShiftId
                ? this.rank(this.selectedShiftId)
                : { ranked: [], unavailable: [], tie: null };
            this.concurrent = this.selectedShiftId ? concurrentShifts(this.model, this.selectedShiftId) : [];
        },

        /** Ranking of a shift with its drawn lot; a lot that no longer fits the tie group is dropped. */
        rank(shiftId) {
            const draw = this.draws.get(shiftId) || null;
            let result = rankCandidates(this.model, shiftId, this.source, draw);
            if (draw && !result.tie?.drawn) {
                this.draws.delete(shiftId);
                result = rankCandidates(this.model, shiftId, this.source);
            }
            return result;
        },

        matchesStatus(cs, status) {
            switch (status) {
                case 'overbooked': return cs.overbooked || (cs.popular && cs.wishes.length > 0);
                case 'open': return !cs.unlimited && cs.open > 0;
                case 'full': return !cs.unlimited && cs.open === 0;
                case 'experience': return cs.expNeed.count > 0 && !cs.expNeed.met;
                default: return true;
            }
        },

        get selectedShift() {
            return this.model && this.selectedShiftId ? this.model.shifts.get(this.selectedShiftId) || null : null;
        },

        /** Days (YYYY-MM-DD) that have shifts, ascending. */
        get dayOptions() {
            return this.model ? [...new Set([...this.model.shifts.values()].map(cs => cs.day))].sort() : [];
        },

        /** Persons as a list for the tables. */
        get personList() {
            return this.model ? [...this.model.persons.values()] : [];
        },

        // --- Views (list, timeline, people) ---

        /** The view in effect: below lg only the list is available. */
        get activeView() {
            return this.isWide ? this.view : 'list';
        },

        setView(view) {
            if (!VIEWS.includes(view)) return;
            this.view = view;
            this.writeUrl();
        },

        /**
         * Day shown by the time views: the chosen one if it has shifts, else the day of the selected
         * shift, else the first day with shifts ('' without shifts).
         */
        get currentTimelineDay() {
            const days = this.dayOptions;
            if (this.timelineDay && days.includes(this.timelineDay)) return this.timelineDay;
            return this.selectedShift?.day || days[0] || '';
        },

        setTimelineDay(day) {
            this.timelineDay = day;
            this.writeUrl();
        },

        /** Moves the time views by n days among the days that have shifts (‹ ›); stays put at the ends. */
        shiftTimelineDay(n) {
            const days = this.dayOptions;
            const i = days.indexOf(this.currentTimelineDay);
            const next = days[Math.min(days.length - 1, Math.max(0, i + n))];
            if (next) this.setTimelineDay(next);
        },

        get hasPrevDay() {
            return this.dayOptions.indexOf(this.currentTimelineDay) > 0;
        },

        get hasNextDay() {
            const days = this.dayOptions;
            const i = days.indexOf(this.currentTimelineDay);
            return i >= 0 && i < days.length - 1;
        },

        /** Data of the timeline view for the current day and filter (see timeline.js). */
        get timeline() {
            if (!this.model) return { startMin: 0, endMin: 0, hours: [], rows: [] };
            return buildTimeline(this.model, this.shiftList.map(cs => cs.id), this.currentTimelineDay);
        },

        /** Data of the people view for the current day (rows in model order; the template sorts them). */
        get personTimeline() {
            if (!this.model) return { startMin: 0, endMin: 0, hours: [], selection: null, rows: [] };
            return buildPersonTimeline(this.model, this.currentTimelineDay, this.selectedShiftId);
        },

        selectShift(id) {
            this.selectedShiftId = id;
            this.writeUrl();
            this.refresh();
        },

        setSource(source) {
            this.source = source === 'all' ? 'all' : 'interested';
            this.refresh();
        },

        /** Draws lots among the people of a tie in the shift (again, if it was drawn before). */
        drawTie(shiftId, keys) {
            this.draws.set(shiftId, newLottery(keys));
            this.refresh();
        },

        setFilter(patch) {
            this.filter = { ...this.filter, ...patch };
            this.writeUrl();
            this.refresh();
        },

        resetFilter() {
            this.setFilter({ typeIds: [], days: [], status: '' });
        },

        get hasFilter() {
            return this.filter.typeIds.length > 0 || this.filter.days.length > 0 || !!this.filter.status;
        },

        // --- Coordination switch ---

        /** Coordination is switched off for this plan (plan setting coordination_enabled). */
        get coordinationOff() {
            return !!this.plan && !this.plan.coordination_enabled;
        },

        /** Switches coordination on (plan update with all plan fields), then reloads. */
        async enableCoordination() {
            if (this.busy || !this.plan) return;
            const p = this.plan;
            this.busy = true;
            this.pendingKey = 'enable';
            this.clearErrors();
            try {
                await this.api('PUT', '', {
                    title_de: p.title_de,
                    title_en: p.title_en,
                    description_de: p.description_de,
                    description_en: p.description_en,
                    start_date: p.start_date,
                    end_date: p.end_date,
                    is_multilingual: p.is_multilingual,
                    day_boundary_time: p.day_boundary_time,
                    collaborative_mode: p.collaborative_mode,
                    show_declined: p.show_declined,
                    experience_field_id: p.experience_field_id ?? null,
                    attendance_mode: p.attendance_mode,
                    default_target_minutes: p.default_target_minutes ?? null,
                    min_rest_minutes: p.min_rest_minutes ?? null,
                    max_day_minutes: p.max_day_minutes ?? null,
                    coordination_enabled: true,
                });
                await this.load(true);
            } catch (e) {
                this.fail(e, false);
            } finally {
                this.busy = false;
                this.pendingKey = null;
            }
        },

        // --- Tabs ---

        setTab(tab) {
            if (!TABS.includes(tab)) return;
            this.tab = tab;
            this.writeUrl();
        },

        /** Moves focus between tabs with the arrow keys (WAI-ARIA tabs pattern). */
        tabKey(event) {
            const i = TABS.indexOf(this.tab);
            let next = null;
            if (event.key === 'ArrowRight') next = TABS[(i + 1) % TABS.length];
            else if (event.key === 'ArrowLeft') next = TABS[(i + TABS.length - 1) % TABS.length];
            else if (event.key === 'Home') next = TABS[0];
            else if (event.key === 'End') next = TABS[TABS.length - 1];
            if (!next) return;
            event.preventDefault();
            this.setTab(next);
            this.$nextTick(() => document.getElementById(`${this.tabPrefix}-tab-${next}`)?.focus());
        },

        tabLabel(name) {
            return this.$t(`coord.tab_${name}`);
        },

        tabCount(name) {
            return name === 'people' && this.model ? this.model.persons.size : 0;
        },

        // --- Changes ---

        /** Shows an error in the open modal (error-display partial) or, without a modal, as a page alert. */
        fail(e, modal) {
            console.error('coordination: action failed', e);
            if (modal) {
                this.modalError.hasError = true;
                this.modalError.errorTitle = e.message || this.$t('coord.error_action');
                this.modalError.errorDetails = { message: e.message, status: e.status };
                this.modalError.showTechnicalDetails = false;
            } else {
                this.actionError = e.message || this.$t('coord.error_action');
            }
        },

        clearErrors() {
            this.actionError = null;
            this.modalError.hasError = false;
        },

        /** The participation of a person in a shift, or undefined. */
        part(shiftId, key) {
            return this.model?.partIndex.get(`${shiftId}|${key}`);
        },

        /**
         * Applies operations on the server (one transaction) and offers undo.
         * @param {Array<{shift_id:number, name:string, status:string}>} ops
         * @param {{lock?: number[], message?: string, modal?: boolean, pending?: string|null}} options
         *   message defaults to "n changes applied"; modal: show errors in the open modal;
         *   pending: id shown as loading while the request runs (pendingKey)
         * @returns {Promise<boolean>} whether it worked
         */
        async apply(ops, { lock = [], message, modal = false, pending = null } = {}) {
            if (this.busy) return false;
            this.busy = true;
            this.pendingKey = pending;
            this.clearErrors();
            const before = this.model;
            try {
                const lockNow = lock.filter(id => !before.shifts.get(id)?.locked);
                const result = await this.api('POST', '/coordination/apply', { ops, lock: lockNow });
                this.lastUndo = { ops: invertOps(ops, before), unlock: lockNow };
                this.showToast(message || this.$t('coord.applied', { count: result.applied }), true);
                await this.load(true);
                return true;
            } catch (e) {
                this.fail(e, modal);
                return false;
            } finally {
                this.busy = false;
                this.pendingKey = null;
            }
        },

        /** Takes back the last apply: restores the old statuses and unlocks what was locked. */
        async undo() {
            const last = this.lastUndo;
            if (!last || this.busy) return;
            this.busy = true;
            this.clearErrors();
            try {
                await this.api('POST', '/coordination/apply', { ops: last.ops, lock: [] });
                for (const id of last.unlock) await this.api('POST', `/shifts/${id}/unlock`);
                this.lastUndo = null;
                this.showToast(this.$t('coord.undone'), false);
                await this.load(true);
            } catch (e) {
                this.fail(e, false);
            } finally {
                this.busy = false;
            }
        },

        // --- Auto distribution preview ---

        /** Shows a computed auto-distribution result instead of the workbench. */
        openPreview(result, options) {
            this.preview = { result, options };
            this.previewStale = false;
        },

        discardPreview() {
            this.preview = null;
            this.previewStale = false;
        },

        /** Applies the previewed operations (not possible once the plan changed meanwhile). */
        async applyPreview() {
            if (!this.preview || this.previewStale) return;
            const { ops, lock } = this.preview.result;
            if (await this.apply(ops, { lock, pending: 'preview' })) this.discardPreview();
        },

        /** Preview of what ops would do (no server call). */
        simulate(ops) {
            return applyOpsLocally(this.model, ops);
        },

        showToast(message, undoable) {
            clearTimeout(this.toastTimer);
            this.toast = { message, undoable };
            this.toastTimer = setTimeout(() => { this.toast = null; }, TOAST_MS);
        },

        closeToast() {
            clearTimeout(this.toastTimer);
            this.toast = null;
        },

        // --- Formatting helpers for all partials ---

        lang() {
            return window.AlpineI18n?.locale === 'en' ? 'en' : 'de';
        },

        /** Localised title from { de, en } or from an API object with title_de / title_en. */
        t(title) {
            if (!title) return '';
            return title[this.lang()] || title.en || title.de || '';
        },

        typeTitle(type) {
            return this.lang() === 'de' ? (type.title_de || type.title_en) : (type.title_en || type.title_de);
        },

        /** Localised label of a person field or option ({ label_de, label_en } or { de, en }). */
        fieldLabel(f) {
            if (!f) return '';
            if (f.label_en !== undefined) return (this.lang() === 'de' ? f.label_de : f.label_en) || f.label_en || f.label_de || '';
            return this.t(f);
        },

        /** Minutes → hours with one decimal in the UI language. */
        hours(minutes) {
            return new Intl.NumberFormat(this.lang() === 'en' ? 'en-GB' : 'de-DE', { maximumFractionDigits: 1 }).format((minutes || 0) / 60);
        },

        number(n, digits = 1) {
            return new Intl.NumberFormat(this.lang() === 'en' ? 'en-GB' : 'de-DE', { maximumFractionDigits: digits }).format(n || 0);
        },

        /** Minutes → "1:30". */
        duration(minutes) {
            const m = Math.max(0, Math.round(minutes || 0));
            return `${Math.floor(m / 60)}:${String(m % 60).padStart(2, '0')}`;
        },

        dayLabel(day) {
            return formatDate(typeof day === 'string' ? parseDate(day) : day, this.lang(), { weekday: 'short', day: 'numeric', month: 'numeric' });
        },

        time(d) {
            return d ? formatTime(d, this.lang()) : '';
        },

        /** Load factor of a shift as "×1.5". */
        factorLabel(cs) {
            return cs ? `×${this.number(cs.factor, 2)}` : '';
        },

        /** Average load factor (load per hour) of a person's numbers { hours, load }; "–" without hours. */
        avgFactor(s) {
            return s && s.hours > 0 ? this.number(s.load / s.hours, 2) : '–';
        },

        /** "Bar, Sa 4.7. 20:00–24:00, ×1.5" for a model shift. */
        shiftLabel(cs) {
            if (!cs) return '';
            return `${this.typeTitle(cs.type)}, ${this.dayLabel(cs.start)} ${this.time(cs.start)}–${this.time(cs.end)}, ${this.factorLabel(cs)}`;
        },

        /** "Bar 20:00" for a model shift (type and start time). */
        shortLabel(cs) {
            return cs ? `${this.typeTitle(cs.type)} ${this.time(cs.start)}` : '';
        },

        /** Text of an impact { id, params } of a candidate (long: the explanation for the tooltip). */
        impactText(imp, long = false) {
            const shift = this.shortLabel(this.model?.shifts.get(imp.params.shiftId));
            return this.$t(`coord.impact_${imp.id}${long ? '_long' : ''}`, { shift });
        },

        /** Warnings and impacts of a candidate as one text (tooltip and screen readers). */
        candidateNotes(c) {
            return [...c.warnings.map(w => this.reasonText('warn', w)), ...c.impacts.map(i => this.impactText(i))].join(', ');
        },

        /** "3/4" or "3/∞" for a model shift. */
        occupancy(cs) {
            return `${cs.occupancy}/${cs.unlimited ? '∞' : cs.required}`;
        },

        /** Text of a warning from model.persons[].warnings ({ kind, minutes }). */
        warningText(w) {
            const params = {};
            if (w.kind === 'rest') params.duration = this.duration(w.minutes);
            if (w.kind === 'day') params.hours = this.number((w.minutes || 0) / 60);
            return this.$t(`coord.warn_${w.kind}`, params);
        },

        /** Name of the level an experience need refers to ("" without need). */
        levelLabel(cs) {
            const o = this.model?.experience.options[cs.expNeed.level];
            return o ? this.t(o) : '';
        },

        /** Name of an experience level (number, -1 = none) for badges and dots. */
        levelName(level) {
            const o = this.model?.experience.options[level];
            return o ? this.t(o) : '–';
        },

        /**
         * Tone classes of an experience level (level 0..count-1, -1 = none). Written out so the Tailwind CLI finds
         * them. The dots (coord-level-dots) and the name carry the meaning, the tone only grades it.
         */
        levelTone(level, count) {
            if (level < 0) return 'text-base-content/60';
            const r = level / Math.max(1, count - 1);
            if (count === 1) return 'bg-primary text-primary-content';
            if (r < 0.34) return 'border border-primary/40';
            if (r < 0.67) return 'bg-primary/20';
            return 'bg-primary text-primary-content';
        },

        /** Text of a reason or warning { id, params } with the given key prefix (reason | warn). */
        reasonText(kind, r) {
            const params = { ...r.params };
            if (params.minutes !== undefined) params.duration = this.duration(params.minutes);
            for (const k of ['hours', 'target', 'load']) if (params[k] !== undefined) params[k] = this.number(params[k]);
            const very = r.id === 'popular_received' && params.very > 0 ? '_very' : '';
            return this.$t(`coord.${kind}_${r.id}${very}`, params);
        },

        planTitle() {
            if (!this.plan) return '';
            return this.lang() === 'de' ? (this.plan.title_de || this.plan.title_en) : (this.plan.title_en || this.plan.title_de);
        },

        backUrl() {
            return `/plans/${encodeURIComponent(this.token)}`;
        },
    }));
});
