import { formatDate, formatTime, parseDate, nowInZone } from './datetime.js';
import { computeStats } from './stats/compute.js';
import { planUrl } from './stats/links.js';

const TABS = ['overview', 'gaps', 'people', 'history'];

document.addEventListener('alpine:init', () => {
    Alpine.data('shiftPlanStats', () => ({
        token: '',
        adminToken: '',
        plan: null,
        types: [],
        model: null,
        tab: 'overview',
        filter: { typeIds: [], days: [] },
        isLoading: true,
        error: null,

        init() {
            const parts = window.location.pathname.split('/');
            this.token = parts[parts.length - 2];
            try {
                this.adminToken = localStorage.getItem(`admin_token_${this.token}`) || '';
            } catch (e) {
                this.adminToken = '';
            }
            this.readUrl();
            this.load();
        },

        get isAdmin() {
            return !!this.adminToken;
        },

        // --- URL state (?tab=&type=&day=) ---

        readUrl() {
            const sp = new URLSearchParams(window.location.search);
            const tab = sp.get('tab');
            if (TABS.includes(tab)) this.tab = tab;
            const type = (sp.get('type') || '').split(',').map(Number).filter(n => Number.isInteger(n) && n > 0);
            const day = (sp.get('day') || '').split(',').filter(d => /^\d{4}-\d{2}-\d{2}$/.test(d));
            this.filter = { typeIds: type, days: day };
        },

        writeUrl() {
            const sp = new URLSearchParams();
            if (this.tab !== 'overview') sp.set('tab', this.tab);
            if (this.filter.typeIds.length) sp.set('type', this.filter.typeIds.join(','));
            if (this.filter.days.length) sp.set('day', this.filter.days.join(','));
            const q = sp.toString();
            history.replaceState(null, '', window.location.pathname + (q ? `?${q}` : ''));
        },

        // --- Loading and computing ---

        async load() {
            this.isLoading = true;
            this.error = null;
            try {
                const headers = this.adminToken ? { 'X-Admin-Token': this.adminToken } : {};
                const [planRes, typesRes] = await Promise.all([
                    fetch(`/api/plans/${this.token}`, { headers }),
                    fetch(`/api/plans/${this.token}/types`, { headers }),
                ]);
                if (!planRes.ok || !typesRes.ok) throw new Error(`HTTP ${planRes.status}/${typesRes.status}`);
                this.plan = await planRes.json();
                this.types = await typesRes.json();
                this.recompute();
            } catch (e) {
                console.error('stats: loading failed', e);
                this.error = e.message || String(e);
            } finally {
                this.isLoading = false;
            }
        },

        recompute() {
            if (!this.plan) return;
            this.model = computeStats(this.plan, this.types, this.filter, nowInZone(this.plan.time_zone));
        },

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
            this.$nextTick(() => document.getElementById(`stats-tab-${next}`)?.focus());
        },

        setFilter(patch) {
            this.filter = { ...this.filter, ...patch };
            this.writeUrl();
            this.recompute();
        },

        setTypeFilter(typeId) {
            this.setFilter({ typeIds: typeId ? [Number(typeId)] : [] });
        },

        setDayFilter(day) {
            this.setFilter({ days: day ? [day] : [] });
        },

        resetFilter() {
            this.setFilter({ typeIds: [], days: [] });
        },

        get hasFilter() {
            return this.filter.typeIds.length > 0 || this.filter.days.length > 0;
        },

        // --- Helpers shared by all tab partials (formatting only) ---

        /** Localised title from { de, en }. */
        t(title) {
            if (!title) return '';
            const lang = window.AlpineI18n?.locale === 'en' ? 'en' : 'de';
            return title[lang] || title.en || title.de || '';
        },

        lang() {
            return window.AlpineI18n?.locale === 'en' ? 'en' : 'de';
        },

        /** Minutes → hours with one decimal in the UI language. */
        hours(minutes) {
            return new Intl.NumberFormat(this.lang() === 'en' ? 'en-GB' : 'de-DE', {
                maximumFractionDigits: 1,
            }).format((minutes || 0) / 60);
        },

        number(n) {
            return new Intl.NumberFormat(this.lang() === 'en' ? 'en-GB' : 'de-DE').format(n || 0);
        },

        /** "Sa 4.7." for a day key or Date. */
        dayLabel(day) {
            return formatDate(typeof day === 'string' ? parseDate(day) : day, this.lang(),
                { weekday: 'short', day: 'numeric', month: 'numeric' });
        },

        /** Long date for a day key or Date. */
        dateLabel(day) {
            return formatDate(typeof day === 'string' ? parseDate(day) : day, this.lang(),
                { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });
        },

        time(d) {
            return d ? formatTime(d, this.lang()) : '';
        },

        timeRange(start, end) {
            return `${this.time(start)}–${this.time(end)}`;
        },

        planTitle() {
            if (!this.plan) return '';
            return this.lang() === 'de' ? (this.plan.title_de || this.plan.title_en) : (this.plan.title_en || this.plan.title_de);
        },

        planUrl(params) {
            return planUrl(this.token, params);
        },

        backUrl() {
            return `/plans/${encodeURIComponent(this.token)}`;
        },
    }));
});
