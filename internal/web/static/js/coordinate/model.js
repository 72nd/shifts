/**
 * Coordination model: pure functions without Alpine, DOM or fetch.
 *
 * buildModel turns the plan, its shift types (with shifts and participations of
 * every status), the stored person rows and the lottery numbers into a CoordModel.
 * rank.js and autodistribute.js read it; the UI only reads the model and never
 * contains fairness or rule logic.
 *
 * Terms (see prj/coordination-plan.md, section 4.1):
 * - Einsatz: assigned, or signed up in a shift that is not popular or is locked.
 * - Open wish: signed up in a popular, unlocked shift (not decided yet).
 * - Declined: declined by the coordination (counts as a declined wish for fairness).
 *
 * @typedef {Object} CoordShift
 * @property {Object} shift     API shift
 * @property {Object} type      API shift type
 * @property {number} id
 * @property {Date} start       floating wall clock (datetime.js)
 * @property {Date} end
 * @property {number} minutes
 * @property {string} day       logical day key YYYY-MM-DD
 * @property {number} factor    load factor (shift override, else type, default 1)
 * @property {boolean} popular
 * @property {boolean} overbooked  required > 0 and occupancy > required
 * @property {boolean} locked
 * @property {boolean} unlimited   required === -1
 * @property {number} required
 * @property {number} occupancy    participations that are not declined
 * @property {number} interest     participations of any status that people made themselves (not added by the coordination)
 * @property {number|null} open    free places for new assignments (null if unlimited):
 *                                 required minus Einsätze
 * @property {{count:number, levelId:string|null, level:number, have:number, met:boolean}} expNeed
 *                                 count 0 = no need; level = index of levelId (−1 = none)
 * @property {string[]} assigned   person keys with an Einsatz
 * @property {string[]} wishes     person keys with an open wish
 * @property {string[]} declined   person keys with a declined participation
 *
 * @typedef {Object} CoordPerson
 * @property {string} key
 * @property {string} name
 * @property {Object|null} row         stored person row (API) or null
 * @property {number} level            experience level (−1 = none)
 * @property {{from:Date|null, until:Date|null}} attendance
 * @property {number|null} targetMin   own target, else plan default
 * @property {boolean} prefersLongDays
 * @property {number} hours            hours of all Einsätze
 * @property {number} load             hours × load factor
 * @property {number} popularReceived  Einsätze in popular shifts
 * @property {number} wishesOpen
 * @property {number} wishesDeclined
 * @property {number} assignedUnwished Einsätze added by the coordination
 * @property {Map<string, number>} perDayMin  minutes per logical day
 * @property {boolean} missingRequired required participant field without value
 * @property {Array<{kind:'overlap'|'rest'|'day'|'over_target'|'under_target'|'absent', shiftIds:number[], minutes?:number}>} warnings
 * @property {CoordShift[]} items      Einsätze sorted by start
 *
 * @typedef {Object} CoordModel
 * @property {Map<number, CoordShift>} shifts  sorted by start
 * @property {Map<string, CoordPerson>} persons
 * @property {{field:Object|null, options:Array<{id:string, de:string, en:string, level:number}>}} experience
 * @property {{restMin:number|null, dayMin:number|null}} limits  plan rules (null = not set)
 * @property {Array} parts     all participations {shiftId, key, name, status, addedByAdmin} (basis of applyOpsLocally)
 * @property {Map<string, Object>} partIndex  the same by "shiftId|key"
 * @property {Object} ctx      static input shared by rebuilt models
 */
import {
  parseDateTime, addMinutes, dateKey, logicalDay,
} from '../datetime.js';
import {
  personKey, boundaryMinutes, isPopular, isEinsatz, findShiftWarnings,
} from '../stats/compute.js';

const MS_MINUTE = 60 * 1000;

/** Collapses whitespace like the stats page does. */
function cleanName(name) {
  return String(name || '').trim().replace(/\s+/g, ' ');
}

/**
 * Builds the model.
 * @param {Object} plan   GET /api/plans/{token} (with fields)
 * @param {Array} types   GET /api/plans/{token}/types as admin (participations of every status)
 * @param {Array} people  GET /api/plans/{token}/people (stored person rows)
 * @param {Map<string, number>|Object} lotteryByKey  random number per person key
 * @returns {CoordModel}
 */
export function buildModel(plan, types, people, lotteryByKey) {
  const boundary = boundaryMinutes(plan.day_boundary_time);
  const shifts = [];
  const parts = [];
  for (const type of types || []) {
    for (const s of type.shifts || []) {
      const start = parseDateTime(s.start_time);
      if (!start) continue;
      shifts.push({ shift: s, type, id: s.id, start, minutes: s.duration || 0, end: addMinutes(start, s.duration || 0) });
      for (const p of s.participants || []) {
        parts.push({
          shiftId: s.id,
          key: personKey(p.name),
          name: cleanName(p.name),
          status: p.status || 'signed_up',
          addedByAdmin: !!p.added_by_admin,
        });
      }
    }
  }
  shifts.sort((a, b) => a.start - b.start || a.id - b.id);

  const field = (plan.fields || []).find(f => f.id === plan.experience_field_id && f.kind === 'choice') || null;
  const experience = {
    field,
    options: (field?.options || []).map((o, level) => ({ id: o.id, de: o.de, en: o.en, level })),
  };
  const lottery = lotteryByKey instanceof Map ? lotteryByKey : new Map(Object.entries(lotteryByKey || {}));
  const rows = new Map((people || []).map(row => [personKey(row.name), row]));

  const ctx = { plan, boundary, shifts, experience, rows, lottery };
  return buildFromParts(ctx, parts);
}

/** Builds a model from static context and the list of participations. */
function buildFromParts(ctx, parts) {
  const { plan, boundary, shifts: shiftList, experience, rows, lottery } = ctx;
  const restMin = plan.min_rest_minutes ?? null;
  const dayMin = plan.max_day_minutes ?? null;
  const partsByShift = new Map();
  for (const p of parts) {
    if (!partsByShift.has(p.shiftId)) partsByShift.set(p.shiftId, []);
    partsByShift.get(p.shiftId).push(p);
  }

  const persons = new Map();
  const spellings = new Map();
  const ensurePerson = (key) => {
    let person = persons.get(key);
    if (!person) {
      const row = rows.get(key) || null;
      const level = row && experience.field ? experience.options.find(o => o.id === row.values?.[experience.field.id])?.level ?? -1 : -1;
      const targetMin = row?.target_minutes ?? plan.default_target_minutes ?? null;
      person = {
        key,
        name: row?.name || key,
        row,
        level,
        attendance: {
          from: row?.available_from ? parseDateTime(row.available_from) : null,
          until: row?.available_until ? parseDateTime(row.available_until) : null,
        },
        targetMin,
        prefersLongDays: !!row?.prefers_long_days,
        hours: 0,
        load: 0,
        popularReceived: 0,
        wishesOpen: 0,
        wishesDeclined: 0,
        assignedUnwished: 0,
        perDayMin: new Map(),
        missingRequired: false,
        warnings: [],
        items: [],
      };
      persons.set(key, person);
    }
    return person;
  };
  for (const key of rows.keys()) ensurePerson(key);

  const shifts = new Map();
  for (const entry of shiftList) {
    const { shift, type } = entry;
    const list = partsByShift.get(shift.id) || [];
    // Interest = people who asked for the shift; entries made by the coordination are not interest.
    const interest = list.filter(p => !p.addedByAdmin).length;
    const popular = isPopular(shift, type, interest);
    const locked = !!shift.is_locked;
    const unlimited = shift.required === -1;
    const required = unlimited ? -1 : Math.max(0, shift.required || 0);
    const cs = {
      shift,
      type,
      id: shift.id,
      start: entry.start,
      end: entry.end,
      minutes: entry.minutes,
      day: dateKey(logicalDay(entry.start, boundary)),
      factor: shift.load_factor ?? type.load_factor ?? 1,
      popular,
      locked,
      unlimited,
      required,
      occupancy: 0,
      interest,
      overbooked: false,
      open: null,
      expNeed: { count: 0, levelId: null, level: -1, have: 0, met: true },
      assigned: [],
      wishes: [],
      declined: [],
    };
    for (const p of list) {
      const person = ensurePerson(p.key);
      let names = spellings.get(p.key);
      if (!names) spellings.set(p.key, names = new Map());
      names.set(p.name, (names.get(p.name) || 0) + 1);
      if (p.status === 'declined') {
        cs.declined.push(p.key);
        person.wishesDeclined++;
        continue;
      }
      cs.occupancy++;
      if (isEinsatz(p.status, popular, locked)) {
        cs.assigned.push(p.key);
        person.items.push(cs);
        person.hours += cs.minutes / 60;
        person.load += (cs.minutes / 60) * cs.factor;
        person.perDayMin.set(cs.day, (person.perDayMin.get(cs.day) || 0) + cs.minutes);
        if (popular) person.popularReceived++;
        if (p.addedByAdmin) person.assignedUnwished++;
      } else {
        cs.wishes.push(p.key);
        person.wishesOpen++;
      }
    }
    cs.overbooked = !unlimited && required > 0 && cs.occupancy > required;
    cs.open = unlimited ? null : Math.max(0, required - cs.assigned.length);

    // Experience need: shift setting, else type setting; only if an experience field is set
    const count = shift.exp_min_count ?? type.exp_min_count ?? 0;
    const levelId = shift.exp_min_level ?? type.exp_min_level ?? null;
    const level = experience.options.find(o => o.id === levelId)?.level ?? -1;
    if (experience.field && count > 0 && level >= 0) {
      const have = cs.assigned.filter(k => persons.get(k).level >= level).length;
      cs.expNeed = { count, levelId, level, have, met: have >= count };
    }
    shifts.set(cs.id, cs);
  }

  // Person names, required fields and warnings
  const requiredFields = (plan.fields || []).filter(f => f.required && f.editable_by === 'participant');
  for (const person of persons.values()) {
    if (!person.row) {
      const names = spellings.get(person.key);
      if (names) person.name = [...names.entries()].sort((a, b) => b[1] - a[1])[0][0];
    }
    person.missingRequired = requiredFields.some(f => {
      const v = person.row?.values?.[f.id];
      return v === undefined || v === null || v === '';
    });
    person.items.sort((a, b) => a.start - b.start || a.id - b.id);
    person.warnings = personWarnings(person, plan, restMin, dayMin);
  }

  return {
    shifts,
    persons,
    experience,
    limits: { restMin, dayMin },
    parts,
    partIndex: new Map(parts.map(p => [`${p.shiftId}|${p.key}`, p])),
    ctx,
  };
}

/** Warnings of a person's Einsätze (shared checks from the statistics plus coordination rules). */
function personWarnings(person, plan, restMin, dayMin) {
  const items = person.items;
  const warnings = [];
  for (const w of findShiftWarnings(items, restMin, person.prefersLongDays ? null : dayMin)) {
    warnings.push({ kind: w.kind === 'dayload' ? 'day' : w.kind, shiftIds: w.shiftIds, minutes: w.minutes });
  }
  const { from, until } = person.attendance;
  for (const s of items) {
    if ((from && s.start < from) || (until && s.end > until)) warnings.push({ kind: 'absent', shiftIds: [s.id] });
  }
  if (person.targetMin !== null && person.targetMin > 0) {
    const minutes = person.hours * 60;
    if (minutes > person.targetMin * 1.25) warnings.push({ kind: 'over_target', shiftIds: [] });
    else if (minutes < person.targetMin) warnings.push({ kind: 'under_target', shiftIds: [] });
  }
  return warnings;
}

/**
 * Applies operations like the server does and returns a new model; the input model is unchanged.
 * An existing participation gets the new status (or is removed); otherwise one is created
 * with addedByAdmin set. For preview and undo.
 * @param {CoordModel} model
 * @param {Array<{shift_id:number, name:string, status:string}>} ops
 * @returns {CoordModel}
 */
export function applyOpsLocally(model, ops) {
  const parts = model.parts.map(p => ({ ...p }));
  const index = new Map(parts.map((p, i) => [`${p.shiftId}|${p.key}`, i]));
  const removed = new Set();
  for (const op of ops) {
    const key = personKey(op.name);
    const id = `${op.shift_id}|${key}`;
    const at = index.get(id);
    if (at !== undefined && !removed.has(at)) {
      if (op.status === 'remove') removed.add(at);
      else parts[at].status = op.status;
    } else if (op.status !== 'remove') {
      parts.push({ shiftId: op.shift_id, key, name: cleanName(op.name), status: op.status, addedByAdmin: true });
      index.set(id, parts.length - 1);
      removed.delete(parts.length - 1);
    }
  }
  const kept = parts.filter((_, i) => !removed.has(i));
  return buildFromParts(model.ctx, kept);
}

/**
 * Operations that restore the state before ops (for undo). The model must be the one the ops were
 * computed from. A participation that did not exist is removed again; an existing one gets its old
 * status back; a removed one is created again (it was created by the coordination, so it comes back
 * as such). Locks are not part of the ops; the caller unlocks the shifts it locked.
 * @param {Array<{shift_id:number, name:string, status:string}>} ops
 * @param {CoordModel} before
 * @returns {Array<{shift_id:number, name:string, status:string}>}
 */
export function invertOps(ops, before) {
  const state = new Map(before.parts.map(p => [`${p.shiftId}|${p.key}`, p.status]));
  const inverse = [];
  for (const op of [...ops].reverse()) {
    const id = `${op.shift_id}|${personKey(op.name)}`;
    const old = state.get(id);
    inverse.push({ shift_id: op.shift_id, name: op.name, status: old === undefined ? 'remove' : old });
  }
  return inverse;
}

/** Hours with one decimal (number, not text; formatting is the UI's job). */
export function round1(n) {
  return Math.round(n * 10) / 10;
}

export { MS_MINUTE };
