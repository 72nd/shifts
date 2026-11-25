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
function overlaps(a, b) {
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
function hardRule(model, person, cs) {
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
    0, // soft violations, filled in by the caller
    cs.popular ? person.popularReceived : 0,
    cs.popular ? -(person.wishesDeclined + person.assignedUnwished) : 0,
    interested ? 0 : person.assignedUnwished,
    relativeLoad(person),
    interested ? 0 : 1,
    adjacent ? 0 : 1,
  ];
}

function compareKeys(a, b) {
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return a[i] - b[i];
  }
  return 0;
}

/**
 * Evaluates one person for one shift.
 * @returns {null | {unavailable: Unavailable} | {candidate: Candidate & {keys:number[], lot:number}}}
 *   null = the person is not a candidate for this source (no Einsatz and no interest, or already assigned)
 */
export function evaluate(model, cs, person, source) {
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
  keys[1] = warnings.length;

  const reasons = [];
  if (needsExperience && person.level >= cs.expNeed.level) reasons.push({ id: 'experience', params: {} });
  if (cs.popular) reasons.push({ id: 'popular_received', params: { count: person.popularReceived } });
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
      alsoInterested,
      keys,
      lot: model.ctx.lottery.get(person.key) ?? 0,
    },
  };
}

/**
 * Candidates for a shift, ranked, with reasons and warnings, plus the people who are not available.
 * Candidates are people without an Einsatz in the shift. With source 'interested' only people with
 * an open wish; with 'all' everybody (people who were declined in this shift too, flagged as such).
 * @param {import('./model.js').CoordModel} model
 * @param {number} shiftId
 * @param {'interested'|'all'} source
 * @returns {{ranked: Candidate[], unavailable: Unavailable[]}}
 */
export function rankCandidates(model, shiftId, source) {
  const cs = model.shifts.get(shiftId);
  if (!cs) return { ranked: [], unavailable: [] };

  const ranked = [];
  const unavailable = [];
  for (const person of model.persons.values()) {
    const ev = evaluate(model, cs, person, source);
    if (!ev) continue;
    if (ev.unavailable) unavailable.push(ev.unavailable);
    else ranked.push(ev.candidate);
  }

  ranked.sort((a, b) => compareKeys(a.keys, b.keys) || a.lot - b.lot || a.name.localeCompare(b.name));
  // The lottery only shows as a reason where it decided between neighbours.
  ranked.forEach((c, i) => {
    const tied = (ranked[i - 1] && !compareKeys(ranked[i - 1].keys, c.keys)) || (ranked[i + 1] && !compareKeys(ranked[i + 1].keys, c.keys));
    if (tied) c.reasons.push({ id: 'lottery', params: {} });
  });
  for (const c of ranked) {
    delete c.keys;
    delete c.lot;
  }

  unavailable.sort((a, b) => a.name.localeCompare(b.name));
  return { ranked, unavailable };
}
