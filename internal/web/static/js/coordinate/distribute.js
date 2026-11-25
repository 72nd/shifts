/**
 * Distribute tab (workbench): UI state and operation builders.
 *
 * The component is nested in x-data="coordinationPage". Methods here do not use the parent scope via
 * `this`; the template passes what they need and sends the returned operations to the page's apply().
 * Everything shown (ranking, reasons, warnings, numbers) comes from the model; no rules live here.
 */

const PERSON_SORTS = ['load', 'experience', 'under_target', 'declined', 'name'];
const MORE_LIMIT = 10;        // rows of "More" before "Show all"
const CONCURRENT_LIMIT = 5;   // chips of "At the same time" before "+n" (only from 6 chips on)

document.addEventListener('alpine:init', () => {
    Alpine.data('coordDistribute', () => ({
        personSort: 'load',
        highlightKey: null,
        lockAfter: false,
        onlyFree: false,       // people view: only persons free during the selected shift
        peopleOpen: false,
        showAllMore: false,    // "More": all rows instead of the first MORE_LIMIT (per shift)
        concurrentOpen: false, // "At the same time": all chips instead of the first CONCURRENT_LIMIT
        hoverShiftId: null,    // shift the pointer/focus points at (impact badge, chip); read by the time views

        init() {
            // New shift: the per-shift list state starts over
            this.$watch('selectedShiftId', () => {
                this.showAllMore = false;
                this.concurrentOpen = false;
                this.hoverShiftId = null;
            });
        },

        // --- Operations (returned, not applied) ---

        assignOps(cs, candidate) {
            return [{ shift_id: cs.id, name: candidate.name, status: 'assigned' }];
        },

        declineOps(cs, name) {
            return [{ shift_id: cs.id, name, status: 'declined' }];
        },

        /** Back to the state before the decision: a wish again, or gone if the coordination added it. */
        resetOps(cs, name, part) {
            return [{ shift_id: cs.id, name, status: part?.addedByAdmin ? 'remove' : 'signed_up' }];
        },

        /** The first free places in ranking order (people who were declined here are skipped). */
        suggestionList(cs, ranked) {
            if (cs.unlimited) return [];
            return ranked.filter(c => !c.declined).slice(0, Math.max(0, cs.open));
        },

        /** What "take suggestion" does; the "Suggestion" section shows the same list (suggestionList). */
        suggestionOps(cs, ranked) {
            return this.suggestionList(cs, ranked).map(c => ({ shift_id: cs.id, name: c.name, status: 'assigned' }));
        },

        suggestionCount(cs, ranked) {
            return this.suggestionOps(cs, ranked).length;
        },

        /** Declines all open wishes of the shift. */
        declineRestOps(cs, persons) {
            return cs.wishes.map(key => ({ shift_id: cs.id, name: persons.get(key).name, status: 'declined' }));
        },

        // --- Candidate list ---

        /**
         * Rows of the candidate list: the suggestion, then a heading, the further candidates (first
         * MORE_LIMIT, or all) and the "show all" button.
         * @returns {Array<{id:string, kind:'row'|'head'|'all', c?:Object, rank?:number, suggested?:boolean, count?:number}>}
         */
        candidateRows(cs, ranked) {
            const suggestion = this.suggestionList(cs, ranked);
            const inSuggestion = new Set(suggestion.map(c => c.key));
            const row = (c, suggested) => ({ id: c.key, kind: 'row', c, rank: ranked.indexOf(c) + 1, suggested });
            const rows = suggestion.map(c => row(c, true));
            const more = ranked.filter(c => !inSuggestion.has(c.key));
            if (!more.length) return rows;
            const shown = this.showAllMore ? more : more.slice(0, MORE_LIMIT);
            rows.push({ id: 'head-more', kind: 'head', count: more.length });
            for (const c of shown) rows.push(row(c, false));
            if (shown.length < more.length) rows.push({ id: 'show-all', kind: 'all', count: more.length });
            return rows;
        },

        /** Reasons shown in the second line; reasons that carry no information are left out. */
        shownReasons(c, suggested) {
            return c.reasons.filter(r => {
                if (r.id === 'load' || r.id === 'load_target' || r.id === 'not_signed_up') return false;
                if (r.id === 'popular_received') return r.params.count > 0;
                if (r.id === 'lottery') return suggested;
                return true;
            });
        },

        /** "3 / 8 h" or "3 h"; fmt formats a number in the UI language. */
        hoursText(person, fmt) {
            return person.targetMin
                ? this.$t('coord.hours_of_target', { hours: fmt(person.hours), target: fmt(person.targetMin / 60) })
                : this.$t('coord.hours_only', { hours: fmt(person.hours) });
        },

        /** Overbooked shifts get a "+" in front so the meaning does not rest on the colour. */
        occupancyBadge(cs, text) {
            return (cs.overbooked ? '+' : '') + text;
        },

        /** Shifts grouped by logical day for the shift list (the list is sorted by start). */
        shiftGroups(list) {
            const groups = [];
            for (const cs of list) {
                const last = groups[groups.length - 1];
                if (last && last.day === cs.day) last.shifts.push(cs);
                else groups.push({ day: cs.day, shifts: [cs] });
            }
            return groups;
        },

        /** Chips of "At the same time": all, or the first CONCURRENT_LIMIT if there are 6 or more. */
        visibleConcurrent(list) {
            return this.concurrentOpen || list.length <= CONCURRENT_LIMIT ? list : list.slice(0, CONCURRENT_LIMIT);
        },

        // --- Person column ---

        /** Grid columns of the person table: only those that carry information. */
        personCols(model) {
            const f = model.flags;
            return ['minmax(0,1fr)', model.experience.field ? '4.5rem' : null, '2.5rem', f.anyFactor ? '2.5rem' : null,
                f.anyPopular ? '2rem' : null, f.anyDeclined ? '2rem' : null, '1.25rem'].filter(Boolean).join(' ');
        },

        /**
         * Persons in groups: by level (highest first, "not stated" last) in the sort "experience", else one
         * group without heading. label(option) names a level.
         */
        personGroups(list, experience, label, noneLabel) {
            const sorted = this.sortedPersons(list, !!experience.field);
            if (this.personSort !== 'experience' || !experience.field) return [{ id: 'all', label: '', persons: sorted }];
            const groups = [];
            for (const p of sorted) {
                const id = p.level >= 0 ? String(p.level) : 'none';
                let g = groups.find(x => x.id === id);
                if (!g) {
                    g = { id, label: p.level >= 0 ? label(experience.options[p.level]) : noneLabel, persons: [] };
                    groups.push(g);
                }
                g.persons.push(p);
            }
            return groups;
        },

        sortedPersons(list, hasLevel = true) {
            let sort = PERSON_SORTS.includes(this.personSort) ? this.personSort : 'load';
            if (sort === 'experience' && !hasLevel) sort = 'load';
            const under = p => (p.targetMin ? p.targetMin - p.hours * 60 : -Infinity);
            const cmp = {
                load: (a, b) => b.load - a.load,
                experience: (a, b) => b.level - a.level || b.load - a.load,
                under_target: (a, b) => under(b) - under(a),
                declined: (a, b) => b.wishesDeclined - a.wishesDeclined,
                name: () => 0,
            }[sort];
            return [...list].sort((a, b) => cmp(a, b) || a.name.localeCompare(b.name));
        },

        // --- Time views ---

        /** Rows of the people view: sorted like the person column; with onlyFree and a selected shift only free persons. */
        timelineRows(pt, hasLevel, onlyFree) {
            const byKey = new Map(pt.rows.map(r => [r.person.key, r]));
            return this.sortedPersons(pt.rows.map(r => r.person), hasLevel)
                .map(p => byKey.get(p.key))
                .filter(r => !onlyFree || r.freeDuring !== false);
        },

        /** Fixed minimum width of a time view: the label column plus 4 rem per hour. */
        timelineWidth(hours, labelRem) {
            return `calc(${labelRem}rem + ${Math.max(1, hours.length) * 4}rem)`;
        },

        /** Arrow left/right: focus the previous/next bar of the same row (bars carry data-bar and data-left). */
        neighbourBar(event) {
            const dir = event.key === 'ArrowRight' ? 1 : event.key === 'ArrowLeft' ? -1 : 0;
            if (!dir) return;
            const row = event.currentTarget.closest('[data-row]');
            const bars = [...row.querySelectorAll('[data-bar]')].sort((a, b) => a.dataset.left - b.dataset.left);
            const next = bars[bars.indexOf(event.currentTarget) + dir];
            if (next) { event.preventDefault(); next.focus(); }
        },

        toggleHighlight(key) {
            this.highlightKey = this.highlightKey === key ? null : key;
        },

        isHighlighted(cs, key) {
            return !!key && (cs.assigned.includes(key) || cs.wishes.includes(key));
        },

        // --- Keyboard in the shift list ---

        /** Id of the shift to select for an arrow key, or null. */
        neighbourShift(event, shiftList, selectedId) {
            const i = shiftList.findIndex(cs => cs.id === selectedId);
            let next = null;
            if (event.key === 'ArrowDown') next = shiftList[Math.min(shiftList.length - 1, i + 1)];
            else if (event.key === 'ArrowUp') next = shiftList[Math.max(0, i - 1)];
            return next ? next.id : null;
        },

        /** Sends a "move" request to the confirmation modal (it listens on the window). */
        askMove(entry, toShiftId) {
            this.$dispatch('coord-move', { entry, toShiftId });
        },
    }));

    /** Confirmation dialog for rebooking a person between overlapping shifts. */
    Alpine.data('coordMove', () => ({
        entry: null,
        toShiftId: null,

        open(detail, dialog) {
            this.entry = detail.entry;
            this.toShiftId = detail.toShiftId;
            dialog.showModal();
        },

        close(dialog) {
            dialog.close();
            this.entry = null;
        },
    }));
});
