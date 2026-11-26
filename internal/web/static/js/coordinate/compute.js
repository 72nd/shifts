/**
 * Coordination core: the public entry point. Pure functions without Alpine, DOM or fetch.
 *
 *   buildModel(plan, types, people)                 → CoordModel         (model.js, typedefs there)
 *   rankCandidates(model, shiftId, source, draw)    → ranked + unavailable + tie (rank.js)
 *   autoDistribute(model, shiftIds, options)         → ops, lock, unfilled … (autodistribute.js)
 *   concurrentShifts(model, shiftId)                 → shifts overlapping in time (rank.js)
 *   buildTimeline(model, shiftIds, day) / buildPersonTimeline(model, day, shiftId)  (timeline.js)
 *   applyOpsLocally(model, ops) / invertOps(ops, model)                    (model.js)
 *
 * Reasons and warnings are { id, params } and are shown with the i18n keys coord.reason_<id>
 * and coord.warn_<id>; no text is built in this layer. The shared helpers of the statistics
 * (personKey, overlap, rest and day checks, popularity) live in ../stats/compute.js.
 */
export { buildModel, applyOpsLocally, invertOps } from './model.js';
export { rankCandidates, cutTie, concurrentShifts } from './rank.js';
export { buildTimeline, buildPersonTimeline } from './timeline.js';
export { autoDistribute } from './autodistribute.js';
export { personKey } from '../stats/compute.js';

/**
 * Random numbers for a lot, one per person key (the draw of a tie, or the whole run of the auto
 * distribution). Nothing in the ranking is random without one.
 * @param {Iterable<string>} keys
 * @returns {Map<string, number>}
 */
export function newLottery(keys) {
  const lottery = new Map();
  for (const key of keys) lottery.set(key, Math.random());
  return lottery;
}
