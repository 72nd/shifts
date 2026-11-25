/**
 * Distribute tab (workbench): UI state and operation builders.
 *
 * The component is nested in x-data="coordinationPage". Methods here do not use the parent scope via
 * `this`; the template passes what they need and sends the returned operations to the page's apply().
 * Everything shown (ranking, reasons, warnings, numbers) comes from the model; no rules live here.
 */

const PERSON_SORTS = ['load', 'under_target', 'declined', 'name'];

document.addEventListener('alpine:init', () => {
    Alpine.data('coordDistribute', () => ({
        personSort: 'load',
        highlightKey: null,
        lockAfter: false,
        peopleOpen: false,

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
        suggestionOps(cs, ranked) {
            if (cs.unlimited) return [];
            return ranked.filter(c => !c.declined).slice(0, cs.open).map(c => ({ shift_id: cs.id, name: c.name, status: 'assigned' }));
        },

        suggestionCount(cs, ranked) {
            return this.suggestionOps(cs, ranked).length;
        },

        /** Declines all open wishes of the shift. */
        declineRestOps(cs, persons) {
            return cs.wishes.map(key => ({ shift_id: cs.id, name: persons.get(key).name, status: 'declined' }));
        },

        // --- Person column ---

        sortedPersons(list) {
            const sort = PERSON_SORTS.includes(this.personSort) ? this.personSort : 'load';
            const under = p => (p.targetMin ? p.targetMin - p.hours * 60 : -Infinity);
            const cmp = {
                load: (a, b) => b.load - a.load,
                under_target: (a, b) => under(b) - under(a),
                declined: (a, b) => b.wishesDeclined - a.wishesDeclined,
                name: () => 0,
            }[sort];
            return [...list].sort((a, b) => cmp(a, b) || a.name.localeCompare(b.name));
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
