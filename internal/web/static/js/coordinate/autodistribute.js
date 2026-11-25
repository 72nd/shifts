/**
 * Auto distribution (prj/coordination-plan.md, section 4.4): a greedy run over the given shifts.
 *
 * Pure function. It never touches existing assigned or declined participations and never moves
 * people between shifts. Soft rules count as hard here, so a shift stays short rather than break one.
 */
import { personKey } from '../stats/compute.js';
import { applyOpsLocally } from './model.js';
import { evaluate, rankCandidates } from './rank.js';

/**
 * @typedef {Object} AutoResult
 * @property {Array<{shift_id:number, name:string, status:string}>} ops
 * @property {number[]} lock  shifts to lock (option lock)
 * @property {Array<{shiftId:number, free:number, reason:'none'|'move_only', movable:string[]}>} unfilled
 *           places that stay free; movable = names who could only be added by rebooking
 * @property {number[]} skipped  unlimited shifts (nothing to distribute)
 * @property {Map<string, {name:string, hoursBefore:number, hoursAfter:number, loadBefore:number, loadAfter:number}>} personDelta
 * @property {Array<{shiftId:number, name:string, neededIn:number}>} impacts
 *           assignments that were made although the person is needed in the concurrent shift neededIn
 *           (which still lacks experienced people after the run)
 * @property {{assigned:number, declined:number, free:number}} summary
 */

/**
 * @param {import('./model.js').CoordModel} model
 * @param {number[]} shiftIds  shifts to distribute (the current filter)
 * @param {{source?:'interested'|'all', declineRest?:boolean, lock?:boolean}} options
 * @returns {AutoResult}
 */
export function autoDistribute(model, shiftIds, { source = 'interested', declineRest = true, lock = false } = {}) {
  const selected = [];
  const skipped = [];
  for (const id of shiftIds) {
    const cs = model.shifts.get(id);
    if (!cs) continue;
    if (cs.unlimited) skipped.push(id);
    else selected.push(cs);
  }
  selected.sort((a, b) => a.start - b.start || a.id - b.id);

  let current = model;
  const ops = [];
  const impacts = [];
  const unfilled = [];
  const closed = new Set();

  // Eligible people per shift: no hard rule, no soft rule, not declined. Kept up to date per person.
  const eligible = new Map();
  const isEligible = (m, cs, person) => {
    const ev = evaluate(m, cs, person, source);
    return !!ev?.candidate && !ev.candidate.declined && ev.candidate.warnings.length === 0;
  };
  for (const cs of selected) {
    const set = new Set();
    for (const person of current.persons.values()) if (isEligible(current, cs, person)) set.add(person.key);
    eligible.set(cs.id, set);
  }

  const movableNames = (shiftId) => rankCandidates(current, shiftId, source).unavailable
    .filter(u => u.move).map(u => u.name);

  for (;;) {
    // The most constrained open shift: fewest eligible people per free place; earlier start wins a tie.
    // Shifts with an open experience need and a qualified eligible person go first (scarcest first),
    // so that a qualified person is not used up elsewhere (prj/coordination-refinement-plan.md, 4.3).
    let best = null;
    let bestExp = null;
    for (const sel of selected) {
      if (closed.has(sel.id)) continue;
      const cs = current.shifts.get(sel.id);
      if (cs.open <= 0) {
        closed.add(sel.id);
        continue;
      }
      const count = eligible.get(sel.id).size;
      if (count === 0) {
        closed.add(sel.id);
        const movable = movableNames(sel.id);
        unfilled.push({ shiftId: sel.id, free: cs.open, reason: movable.length ? 'move_only' : 'none', movable });
        continue;
      }
      const ratio = count / cs.open;
      if (!best || ratio < best.ratio) best = { id: sel.id, ratio };
      if (cs.expNeed.count > 0 && !cs.expNeed.met) {
        let qualified = 0;
        for (const key of eligible.get(sel.id)) if (current.persons.get(key).level >= cs.expNeed.level) qualified++;
        if (qualified > 0) {
          const expRatio = qualified / (cs.expNeed.count - cs.expNeed.have);
          if (!bestExp || expRatio < bestExp.ratio) bestExp = { id: sel.id, ratio: expRatio };
        }
      }
    }
    if (bestExp) best = bestExp;
    if (!best) break;

    const pick = rankCandidates(current, best.id, source).ranked
      .find(c => !c.declined && c.warnings.length === 0);
    if (!pick) {
      // Cannot happen (eligible is non-empty); close the shift instead of looping.
      closed.add(best.id);
      continue;
    }
    const op = { shift_id: best.id, name: current.persons.get(pick.key).name, status: 'assigned' };
    ops.push(op);
    if (pick.impacts.length) impacts.push({ shiftId: best.id, name: op.name, neededIn: pick.impacts[0].params.shiftId });
    current = applyOpsLocally(current, [op]);

    // Only this person's state changed: refresh their eligibility in all open shifts.
    const person = current.persons.get(pick.key);
    for (const sel of selected) {
      if (closed.has(sel.id)) continue;
      const set = eligible.get(sel.id);
      if (isEligible(current, current.shifts.get(sel.id), person)) set.add(pick.key);
      else set.delete(pick.key);
    }
  }

  // Decline the rest in every selected shift that is full now
  const lockIds = [];
  for (const sel of selected) {
    const cs = current.shifts.get(sel.id);
    if (cs.open > 0) continue;
    if (declineRest) {
      for (const key of cs.wishes) {
        ops.push({ shift_id: sel.id, name: current.persons.get(key).name, status: 'declined' });
      }
    }
    if (lock && !cs.locked && ops.some(o => o.shift_id === sel.id)) lockIds.push(sel.id);
  }

  const final = applyOpsLocally(model, ops.filter(o => o.status !== 'declined'));
  // Only report impacts that still hold at the end (the other shift lacks experienced people)
  const remainingImpacts = impacts.filter(i => {
    const need = final.shifts.get(i.neededIn)?.expNeed;
    return need && need.count > 0 && !need.met;
  });
  const personDelta = new Map();
  for (const op of ops) {
    if (op.status !== 'assigned') continue;
    const key = personKey(op.name);
    if (personDelta.has(key)) continue;
    const before = model.persons.get(key);
    const after = final.persons.get(key);
    personDelta.set(key, {
      name: after.name,
      hoursBefore: before ? before.hours : 0,
      hoursAfter: after.hours,
      loadBefore: before ? before.load : 0,
      loadAfter: after.load,
    });
  }

  return {
    ops,
    lock: lockIds,
    unfilled,
    skipped,
    personDelta,
    impacts: remainingImpacts,
    summary: {
      assigned: ops.filter(o => o.status === 'assigned').length,
      declined: ops.filter(o => o.status === 'declined').length,
      free: unfilled.reduce((sum, u) => sum + u.free, 0),
    },
  };
}
