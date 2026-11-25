/**
 * Auto distribution: options dialog and preview (prj/coordination-plan.md, 4.4 and 9.3).
 *
 * The run itself is the pure autoDistribute() of the coordination core. These components only hold
 * the options and shape the result for display; the page applies it (openPreview, applyPreview).
 */
import { autoDistribute } from './compute.js';

document.addEventListener('alpine:init', () => {
    Alpine.data('coordAuto', () => ({
        source: 'interested',
        declineRest: true,
        lock: false,

        open(dialog) {
            dialog.showModal();
        },

        close(dialog) {
            dialog.close();
        },

        /** Runs the distribution over the given shifts; the caller shows the result. */
        run(model, shiftList) {
            const options = { source: this.source, declineRest: this.declineRest, lock: this.lock };
            return { result: autoDistribute(model, shiftList.map(cs => cs.id), options), options };
        },
    }));

    Alpine.data('coordPreview', () => ({
        /**
         * One row per shift that changes or stays open: names to assign and to decline, and what stays free.
         * @returns {Array<{cs: Object, assigned: string[], declined: string[], unfilled: Object|null}>}
         */
        rows(model, result) {
            const byShift = new Map();
            const row = (id) => {
                if (!byShift.has(id)) byShift.set(id, { cs: model.shifts.get(id), assigned: [], declined: [], unfilled: null });
                return byShift.get(id);
            };
            for (const op of result.ops) row(op.shift_id)[op.status === 'assigned' ? 'assigned' : 'declined'].push(op.name);
            for (const u of result.unfilled) row(u.shiftId).unfilled = u;
            return [...byShift.values()].sort((a, b) => a.cs.start - b.cs.start || a.cs.id - b.cs.id);
        },

        /** People whose hours change, sorted by name. */
        deltas(result) {
            return [...result.personDelta.values()].sort((a, b) => a.name.localeCompare(b.name));
        },

        /** Notes of the run: people who were assigned although another shift needs them (result.impacts). */
        impacts(result) {
            return (result.impacts || []).map(i => ({ ...i, id: `${i.shiftId}|${i.name}` }));
        },

        isEmpty(result) {
            return result.ops.length === 0 && result.unfilled.length === 0;
        },
    }));
});
