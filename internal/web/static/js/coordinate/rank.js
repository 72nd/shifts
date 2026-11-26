/**
 * Ranking of candidates for a shift (prj/coordination-plan.md, sections 4.2 and 4.3).
 *
 * Pure functions. Reasons and warnings are returned as { id, params } and rendered by the UI
 * with the i18n keys coord.reason_<id> and coord.warn_<id>; no text is built here.
 */
import { findShiftWarnings } from '../stats/compute.js';
import { round1 } from './model.js';

const MS_MINUTE = 60 * 1000;
const DEFAULT_ADJACENT_MIN = 60;

/**
 * @typedef {Object} Candidate
 * @property {string} key
 * @property {string} name
 * @property {boolean} interested  has an open wish for this shift
 * @property {boolean} declined    was declined for this shift (only in the source "all")
 * @property {Array<{id:string, params:Object}>} reasons  why the person stands where they stand
 * @property {Array<{id:string, params:Object}>} warnings soft rule violations (rest, day, over_target)
 * @property {number[]} alsoInterested  other shifts at the same time with an open wish of this person
 * @property {number} level       experience level of the person (-1 = none)
 * @property {Array<{id:'needed_elsewhere', params:{shiftId:number}}>} impacts
 *           soft hint, not a warning (autodistribute still may pick the person): the person is
 *           indispensable for the open experience need of a concurrent shift (earliest one)
 *
 * @typedef {Object} Unavailable
 * @property {string} key
 * @property {string} name
 * @property {'overlap'|'absent'|'declined'} reason
 * @property {Object} params   overlap: { shiftId } (the first overlapping shift)
 * @property {{fromShiftId:number, fromLocked:boolean, ops:Array}|undefined} move  rebooking, ready-made
 *                             operations (overlap with exactly one Einsatz)
 */

/** Whether two shifts overlap in time. */
export function overlaps(a, b) {
  return a.start < b.end && b.start < a.end;
}

/** Soft rule violations if the person took the shift in addition to their Einsätze. */
function softWarnings(model, person, cs) {
  const warnings = [];
  const { restMin, dayMin } = model.limits;
  const items = [...person.items, cs].sort((a, b) => a.start - b.start || a.id - b.id);
  for (const w of findShiftWarnings(items, restMin, person.prefersLongDays ? null : dayMin)) {
    if (!w.shiftIds.includes(cs.id) || w.kind === 'overlap') continue;
    if (w.kind === 'rest') warnings.push({ id: 'rest', params: { minutes: w.minutes } });
    else warnings.push({ id: 'day', params: { hours: round1(w.minutes / 60) } });
  }
  if (person.targetMin !== null && person.targetMin > 0 && (person.hours * 60 + cs.minutes) > person.targetMin * 1.25) {
    warnings.push({ id: 'over_target', params: {} });
  }
  return warnings;
}

/** Hard rules for the person and the shift: the first one that applies, or null. */
export function hardRule(model, person, cs) {
  const overlapping = person.items.filter(s => s.id !== cs.id && overlaps(s, cs));
  if (overlapping.length) {
    const from = overlapping[0];
    const entry = { reason: 'overlap', params: { shiftId: from.id } };
    if (overlapping.length === 1) {
      const part = model.parts.find(p => p.shiftId === from.id && p.key === person.key);
      entry.move = {
        fromShiftId: from.id,
        fromLocked: from.locked,
        // The coordination's own entry is removed; a self-signed one becomes a declined wish.
        ops: [
          { shift_id: cs.id, name: person.name, status: 'assigned' },
          { shift_id: from.id, name: person.name, status: part?.addedByAdmin ? 'remove' : 'declined' },
        ],
      };
    }
    return entry;
  }
  const { from, until } = person.attendance;
  if ((from && cs.start < from) || (until && cs.end > until)) return { reason: 'absent', params: {} };
  return null;
}

/**
 * Shifts that overlap the given one in time (itself excluded), sorted by start, then type sort order.
 * Unlimited shifts are included.
 * @param {import('./model.js').CoordModel} model
 * @param {number} shiftId
 * @returns {import('./model.js').CoordShift[]}
 */
export function concurrentShifts(model, shiftId) {
  const cs = model.shifts.get(shiftId);
  if (!cs) return [];
  const result = [];
  for (const other of model.shifts.values()) {
    if (other.id !== cs.id && overlaps(other, cs)) result.push(other);
  }
  return result.sort((a, b) => a.start - b.start
    || (a.type.sort_order ?? 0) - (b.type.sort_order ?? 0) || a.id - b.id);
}

/**
 * Bottlenecks among the shifts concurrent to cs: for every limited concurrent shift S' with an open
 * experience need, the people who could still cover it (level, no Einsatz or decline there, hard rules
 * ok). Only shifts whose need cannot be met without one of these people are returned (pool.size <= missing),
 * in start order. Soft rules are ignored on purpose (optimistic estimate).
 * Computed once per ranking, not per candidate.
 * @returns {Array<{shiftId:number, pool:Set<string>}>}
 */
export function bottlenecks(model, cs) {
  const result = [];
  for (const other of concurrentShifts(model, cs.id)) {
    if (other.unlimited || other.expNeed.count === 0 || other.expNeed.met) continue;
    const missing = other.expNeed.count - other.expNeed.have;
    const pool = new Set();
    for (const person of model.persons.values()) {
      if (person.level < other.expNeed.level) continue;
      if (other.assigned.includes(person.key) || other.declined.includes(person.key)) continue;
      if (hardRule(model, person, other)) continue;
      pool.add(person.key);
      if (pool.size > missing) break;
    }
    if (pool.size <= missing) result.push({ shiftId: other.id, pool });
  }
  return result;
}

/** Relative load: share of the target if there is one, else the absolute load. */
function relativeLoad(person) {
  return person.targetMin ? person.load / (person.targetMin / 60) : person.load;
}

/** Whether the person has an Einsatz on the same logical day that directly adjoins the shift. */
function adjoins(model, person, cs) {
  if (!person.prefersLongDays) return false;
  const limit = (model.limits.restMin ?? 0) > 0 ? model.limits.restMin : DEFAULT_ADJACENT_MIN;
  return person.items.some(s => {
    if (s.day !== cs.day) return false;
    const gap = s.end <= cs.start ? (cs.start - s.end) / MS_MINUTE : cs.end <= s.start ? (s.start - cs.end) / MS_MINUTE : Infinity;
    return gap >= 0 && gap <= limit;
  });
}

/** Sort keys of a candidate, in the order of the ranking rules (lower is better). */
function sortKeys(model, cs, person, interested, adjacent) {
  const needsExperience = cs.expNeed.count > 0 && !cs.expNeed.met;
  const experienced = needsExperience && person.level >= cs.expNeed.level;
  return [
    needsExperience ? (experienced ? 0 : 1) : 0,
    0, // concurrent bottleneck (impacts), filled in by the caller
    0, // soft violations, filled in by the caller
    cs.popular ? person.popularScore : 0,
    cs.popular ? -(person.wishesDeclined + person.assignedUnwished) : 0,
    interested ? 0 : person.assignedUnwished,
    relativeLoad(person),
    interested ? 0 : 1,
    adjacent ? 0 : 1,
  ];
}

export function compareKeys(a, b) {
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return a[i] - b[i];
  }
  return 0;
}

/**
 * Evaluates one person for one shift.
 * @param {Array<{shiftId:number, pool:Set<string>}>} [bottle]  result of bottlenecks(); without it no impacts
 * @returns {null | {unavailable: Unavailable} | {candidate: Candidate & {keys:number[]}}}
 *   null = the person is not a candidate for this source (no Einsatz and no interest, or already assigned)
 */
export function evaluate(model, cs, person, source, bottle = []) {
  if (cs.assigned.includes(person.key)) return null;
  const interested = cs.wishes.includes(person.key);
  const declined = cs.declined.includes(person.key);
  if (source === 'interested' && !interested && !declined) return null;
  if (source === 'interested' && declined) {
    return { unavailable: { key: person.key, name: person.name, reason: 'declined', params: {} } };
  }

  const hard = hardRule(model, person, cs);
  if (hard) return { unavailable: { key: person.key, name: person.name, ...hard } };

  const needsExperience = cs.expNeed.count > 0 && !cs.expNeed.met;
  const warnings = softWarnings(model, person, cs);
  const adjacent = adjoins(model, person, cs);
  const keys = sortKeys(model, cs, person, interested, adjacent);
  // Own experience need comes first (Q3): no impact if the person covers it here.
  const coversOwn = needsExperience && person.level >= cs.expNeed.level;
  const hit = coversOwn ? null : bottle.find(b => b.pool.has(person.key));
  const impacts = hit ? [{ id: 'needed_elsewhere', params: { shiftId: hit.shiftId } }] : [];
  keys[1] = impacts.length ? 1 : 0;
  keys[2] = warnings.length;

  const reasons = [];
  if (needsExperience && person.level >= cs.expNeed.level) reasons.push({ id: 'experience', params: {} });
  if (cs.popular) reasons.push({ id: 'popular_received', params: { count: person.popularReceived, very: person.veryPopularReceived } });
  if (cs.popular && person.wishesDeclined > 0) reasons.push({ id: 'wishes_declined', params: { count: person.wishesDeclined } });
  if (person.assignedUnwished > 0 && (cs.popular || !interested)) {
    reasons.push({ id: 'assigned_unwished', params: { count: person.assignedUnwished } });
  }
  if (person.targetMin) {
    reasons.push({ id: 'load_target', params: { hours: round1(person.hours), target: round1(person.targetMin / 60) } });
  } else {
    reasons.push({ id: 'load', params: { load: round1(person.load) } });
  }
  if (adjacent) reasons.push({ id: 'adjacent', params: {} });
  if (!interested) reasons.push({ id: 'not_signed_up', params: {} });

  const alsoInterested = [];
  for (const other of model.shifts.values()) {
    if (other.id !== cs.id && other.wishes.includes(person.key) && overlaps(other, cs)) alsoInterested.push(other.id);
  }

  return {
    candidate: {
      key: person.key,
      name: person.name,
      interested,
      declined,
      reasons,
      warnings,
      impacts,
      level: person.level,
      alsoInterested,
      keys,
    },
  };
}

/**
 * The group of equal sort keys that spans the boundary of the free places, or null.
 * Only candidates that were not declined count. Needs the candidates' `keys` (rankCandidates keeps them).
 * @param {Array<{key:string, declined:boolean, keys:number[]}>} ranked  in ranking order
 * @param {number} places  free places
 * @returns {null | {keys: string[], places: number, before: number}}
 *   keys = person keys of the group, before = candidates clearly ahead of it, places = places left for the group
 */
export function cutTie(ranked, places) {
  const list = ranked.filter(c => !c.declined);
  if (!(places > 0) || places >= list.length) return null;
  const edge = list[places - 1].keys;
  if (compareKeys(edge, list[places].keys) !== 0) return null;
  let from = places - 1;
  while (from > 0 && compareKeys(list[from - 1].keys, edge) === 0) from--;
  let to = places;
  while (to + 1 < list.length && compareKeys(list[to + 1].keys, edge) === 0) to++;
  return { keys: list.slice(from, to + 1).map(c => c.key), places: places - from, before: from };
}

/**
 * Candidates for a shift, ranked, with reasons and warnings, plus the people who are not available.
 * Candidates are people without an Einsatz in the shift. With source 'interested' only people with
 * an open wish; with 'all' everybody (people who were declined in this shift too, flagged as such).
 * Equal sort keys are ordered by `draw` (random number per person key, missing = 0), then by name.
 * Without a draw nothing is random.
 * @param {import('./model.js').CoordModel} model
 * @param {number} shiftId
 * @param {'interested'|'all'} source
 * @param {Map<string, number>|null} [draw]
 * @returns {{ranked: Candidate[], unavailable: Unavailable[], tie: null | {keys:string[], places:number, before:number, drawn:boolean}}}
 *   tie = tie across the free places (cutTie); drawn = the draw holds exactly the group, so it is decided
 *   and the places won by lot carry the reason 'lottery'
 */
export function rankCandidates(model, shiftId, source, draw = null) {
  const cs = model.shifts.get(shiftId);
  if (!cs) return { ranked: [], unavailable: [], tie: null };

  const ranked = [];
  const unavailable = [];
  const bottle = bottlenecks(model, cs);
  for (const person of model.persons.values()) {
    const ev = evaluate(model, cs, person, source, bottle);
    if (!ev) continue;
    if (ev.unavailable) unavailable.push(ev.unavailable);
    else ranked.push(ev.candidate);
  }

  const lot = c => draw?.get(c.key) ?? 0;
  ranked.sort((a, b) => compareKeys(a.keys, b.keys) || lot(a) - lot(b) || a.name.localeCompare(b.name));
  unavailable.sort((a, b) => a.name.localeCompare(b.name));

  const found = cs.unlimited ? null : cutTie(ranked, cs.open);
  let tie = null;
  if (found) {
    const drawn = !!draw && draw.size === found.keys.length && found.keys.every(k => draw.has(k));
    tie = { ...found, drawn };
    if (drawn) {
      // The first `places` of the group in draw order won the lot
      const group = new Set(found.keys);
      ranked.filter(c => group.has(c.key) && !c.declined).slice(0, found.places)
        .forEach(c => c.reasons.push({ id: 'lottery', params: {} }));
    }
  }
  return { ranked, unavailable, tie };
}
