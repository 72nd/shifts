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
import { buildModel, rankCandidates, applyOpsLocally, invertOps, newLottery } from './compute.js';

const TABS = ['distribute', 'people', 'settings'];
const STATUS_FILTERS = ['overbooked', 'open', 'full', 'experience'];
const TOAST_MS = 10000;

document.addEventListener('alpine:init', () => {
    Alpine.data('coordinationPage', () => ({
        token: '',
        adminToken: '',
        plan: null,
        types: [],
        people: [],
        model: null,
        lottery: new Map(),
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
        candidates: { ranked: [], unavailable: [] },
        shiftList: [],

        // Changes in flight and undo
        busy: false,
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

        // --- URL state (?tab=&type=&day=&status=&shift=) ---

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
        },

        writeUrl() {
            const sp = new URLSearchParams();
            if (this.tab !== 'distribute') sp.set('tab', this.tab);
            if (this.filter.typeIds.length) sp.set('type', this.filter.typeIds.join(','));
            if (this.filter.days.length) sp.set('day', this.filter.days.join(','));
            if (this.filter.status) sp.set('status', this.filter.status);
            if (this.selectedShiftId) sp.set('shift', String(this.selectedShiftId));
            const q = sp.toString();
            history.replaceState(null, '', window.location.pathname + (q ? `?${q}` : ''));
        },

        // --- Server communication ---

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
            const keys = new Set();
            for (const type of this.types) for (const s of type.shifts || []) for (const p of s.participants || []) keys.add(personKey(p.name));
            for (const row of this.people) keys.add(personKey(row.name));
            for (const [key, n] of newLottery(keys)) if (!this.lottery.has(key)) this.lottery.set(key, n);
            this.model = buildModel(this.plan, this.types, this.people, this.lottery);
            if (this.selectedShiftId && !this.model.shifts.has(this.selectedShiftId)) this.selectedShiftId = null;
            this.refresh();
            // On a phone the shift is chosen with a select: start with the first one
            if (!this.selectedShiftId && this.shiftList.length && window.matchMedia('(max-width: 767px)').matches) {
                this.selectShift(this.shiftList[0].id);
            }
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
                ? rankCandidates(this.model, this.selectedShiftId, this.source)
                : { ranked: [], unavailable: [] };
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

        selectShift(id) {
            this.selectedShiftId = id;
            this.writeUrl();
            this.refresh();
        },

        setSource(source) {
            this.source = source === 'all' ? 'all' : 'interested';
            this.refresh();
        },

        reroll() {
            this.lottery = newLottery(this.lottery.keys());
            this.rebuild();
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

        number(n) {
            return new Intl.NumberFormat(this.lang() === 'en' ? 'en-GB' : 'de-DE', { maximumFractionDigits: 1 }).format(n || 0);
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

        /** "Bar, Sa 4.7. 20:00–24:00" for a model shift. */
        shiftLabel(cs) {
            if (!cs) return '';
            return `${this.typeTitle(cs.type)}, ${this.dayLabel(cs.start)} ${this.time(cs.start)}–${this.time(cs.end)}`;
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

        /** Text of a reason or warning { id, params } with the given key prefix (reason | warn). */
        reasonText(kind, r) {
            const params = { ...r.params };
            if (params.minutes !== undefined) params.duration = this.duration(params.minutes);
            for (const k of ['hours', 'target', 'load']) if (params[k] !== undefined) params[k] = this.number(params[k]);
            return this.$t(`coord.${kind}_${r.id}`, params);
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
