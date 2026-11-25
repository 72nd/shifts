// Recap tab: turns model.recap into display rows and a plain text for copying.
// Presentation only; all numbers come from compute.js (buildRecap). The copy text is
// built from the same rows the tab renders, so there is only one source of text.

const COPIED_MS = 2000;

document.addEventListener('alpine:init', () => {
    Alpine.data('statsRecap', () => ({
        copied: false,
        copyFailed: false,
        copiedTimer: null,

        destroy() {
            clearTimeout(this.copiedTimer);
        },

        get recap() {
            return this.model.recap;
        },

        /** Big numbers at the top. */
        get figures() {
            const r = this.recap;
            return [
                { id: 'people', label: this.$t('stats.recap_people'), value: this.number(r.people) },
                { id: 'hours', label: this.$t('stats.recap_hours'), value: this.hours(r.minutes) },
                { id: 'shifts', label: this.$t('stats.recap_shifts'), value: this.number(r.shifts) },
                { id: 'signups', label: this.$t('stats.recap_signups'), value: this.number(r.signups) },
            ];
        },

        /** Only in the copy text; the page shows the same numbers as figures. */
        get heroSentence() {
            const r = this.recap;
            if (r.people < 2 || r.shifts < 2) return '';
            return this.$t('stats.recap_sentence', {
                people: this.number(r.people), hours: this.hours(r.minutes), shifts: this.number(r.shifts),
            });
        },

        get daysSentence() {
            const r = this.recap;
            if (r.minutes < 1440) return '';
            return this.$t('stats.recap_sentence_days', {
                days: this.decimal(r.minutes / 1440), workdays: this.number(Math.round(r.minutes / 480)),
            });
        },

        /** Sections with rows { id, label, value, detail }; empty sections are left out. */
        get sections() {
            const r = this.recap;
            const numbers = [];
            const hoursValue = (m) => this.$t('stats.hours_unit', { n: this.hours(m) });
            if (r.nightMinutes > 0) numbers.push(this.row('night', 'stats.recap_night', hoursValue(r.nightMinutes)));
            if (r.avgShiftMinutes > 0) numbers.push(this.row('avg_shift', 'stats.recap_avg_shift', hoursValue(r.avgShiftMinutes)));
            if (r.repeatPercent !== null) {
                numbers.push(this.row('repeat', 'stats.recap_repeat',
                    this.$t('stats.recap_repeat_value', { p: this.number(r.repeatPercent) })));
            }
            if (r.peak) {
                numbers.push(this.row('peak', 'stats.fact_peak_hour', this.number(r.peak.value),
                    `${this.dayLabel(r.peak.day)}, ${this.time(r.peak.at)}`));
            }
            if (r.types >= 2) numbers.push(this.row('types', 'stats.recap_types', this.number(r.types)));

            const facts = (group) => r.facts.filter(f => f.group === group).map(f => this.factRow(f));
            return [
                { id: 'numbers', title: this.$t('stats.recap_numbers'), rows: numbers },
                { id: 'people', title: this.$t('stats.recap_people_title'), rows: facts('people') },
                { id: 'signups', title: this.$t('stats.recap_signups_title'), rows: facts('signups') },
            ].filter(s => s.rows.length);
        },

        row(id, labelKey, value, detail = '') {
            return { id, label: this.$t(labelKey), value, detail };
        },

        factRow(f) {
            let value, detail = this.names(f);
            switch (f.id) {
                case 'most_days':
                    value = this.$t('stats.days_unit', { n: this.number(f.value) });
                    break;
                case 'longest_shift':
                case 'night_owl':
                    value = this.$t('stats.hours_unit', { n: this.hours(f.value) });
                    break;
                case 'early_bird':
                    value = this.$t('stats.fact_early_bird_value', { time: this.clock(f.value) });
                    break;
                case 'dream_team':
                    value = this.$t('stats.fact_dream_team_value', { n: this.number(f.value) });
                    break;
                case 'first_signup':
                    value = `${this.dayLabel(f.at)}, ${this.time(f.at)}`;
                    break;
                case 'fastest_full':
                    value = this.$t('stats.fact_fastest_full_value', { duration: this.duration(f.value) });
                    detail = `${this.t(f.typeTitle)}, ${this.dayLabel(f.day)} ${this.timeRange(f.at, f.end)}`;
                    break;
                case 'last_minute':
                    value = this.number(f.value);
                    detail = this.$t('stats.fact_last_minute_detail');
                    break;
                case 'best_signup_day':
                    value = this.number(f.value);
                    detail = this.dayLabel(f.day);
                    break;
                default: // most_shifts, all_rounder
                    value = this.number(f.value);
            }
            return { id: f.id, label: this.$t(`stats.fact_${f.id}`), value, detail };
        },

        /** "Anna", "Cem und Dora", plus "+2 weitere" for ties. */
        names(f) {
            if (!f.names.length) return '';
            let s = f.names.length === 2 ? this.$t('stats.recap_and', { a: f.names[0], b: f.names[1] }) : f.names[0];
            if (f.others > 0) s += ' ' + this.$t('stats.recap_others', { n: this.number(f.others) });
            return s;
        },

        /** Minutes after midnight → "05:00". */
        clock(minutes) {
            return `${String(Math.floor(minutes / 60)).padStart(2, '0')}:${String(minutes % 60).padStart(2, '0')}`;
        },

        /** Duration for fastest_full: minutes below 1 h, hours below 48 h, else days. */
        duration(minutes) {
            if (minutes < 60) return this.$t('stats.duration_min', { n: this.number(minutes) });
            if (minutes < 48 * 60) return this.$t('stats.hours_unit', { n: this.hours(minutes) });
            return this.$t('stats.days_unit', { n: this.number(Math.round(minutes / 1440)) });
        },

        decimal(n) {
            return new Intl.NumberFormat(this.lang() === 'en' ? 'en-GB' : 'de-DE', { maximumFractionDigits: 1 }).format(n);
        },

        /** Active filters as one line, e.g. "Bereich: Bar, Tag: Sa 4.7."; empty without filter. */
        filterLine() {
            const parts = [];
            if (this.filter.typeIds.length) {
                const names = this.filter.typeIds.map(id => this.t(this.model.options.types.find(o => o.typeId === id)?.title));
                parts.push(`${this.$t('stats.filter_type')}: ${names.join(', ')}`);
            }
            if (this.filter.days.length) {
                parts.push(`${this.$t('stats.filter_day')}: ${this.filter.days.map(d => this.dayLabel(d)).join(', ')}`);
            }
            return parts.join(', ');
        },

        copyText() {
            const blocks = [];
            blocks.push([`${this.planTitle()} – ${this.$t('stats.tab_recap')}`, this.filterLine()].filter(Boolean).join('\n'));
            const sentences = [this.heroSentence, this.daysSentence].filter(Boolean);
            if (sentences.length) blocks.push(sentences.join('\n'));
            for (const s of this.sections) {
                const lines = s.rows.map(r => `- ${r.label}: ${r.value}${r.detail ? ` (${r.detail})` : ''}`);
                blocks.push([s.title, ...lines].join('\n'));
            }
            return blocks.join('\n\n');
        },

        async copy() {
            this.copyFailed = false;
            try {
                await navigator.clipboard.writeText(this.copyText());
            } catch (e) {
                // No clipboard access (e.g. no HTTPS): show the text to copy by hand.
                console.error('stats: copying the recap failed', e);
                this.copyFailed = true;
                return;
            }
            this.copied = true;
            clearTimeout(this.copiedTimer);
            this.copiedTimer = setTimeout(() => { this.copied = false; }, COPIED_MS);
        },
    }));
});
