/**
 * Lab logic (prj/coordination-improvements-plan.md, 5.5 and 6.3). Pure functions without Alpine,
 * DOM or fetch.
 *
 * A lab draft is a list of target statuses {shift_id, name, status} (the format of
 * POST /coordination/apply). The lab model is always applyOpsLocally(liveModel, ops), so new
 * sign-ups of other people show up by themselves and ops that change nothing any more drop out
 * (normalizeOps).
 *
 * @typedef {{shift_id:number, name:string, status:'assigned'|'signed_up'|'declined'|'remove'}} LabOp
 */
import { personKey } from '../stats/compute.js';
import { assignOverlapLanes } from '../timetable-view-layout.js';
import { dayStartMs, offsets, span } from './timeline.js';

/** Minutes per column of the lab grid. */
export const GRID_MINUTES = 5;

/**
 * Last op per (shift, person) wins; ops that do not change the live model are dropped (same
 * status as the live participation, or a removal of a participation that does not exist).
 * The order is the order of the surviving (last) ops.
 * @param {import('./model.js').CoordModel} model  live model
 * @param {LabOp[]} ops
 * @returns {LabOp[]}
 */
export function normalizeOps(model, ops) {
  const last = new Map();
  for (const op of ops || []) {
    const id = `${op.shift_id}|${personKey(op.name)}`;
    last.delete(id); // re-insert so that the order follows the last op
    last.set(id, op);
  }
  const result = [];
  for (const [id, op] of last) {
    const part = model.partIndex.get(id);
    if (op.status === 'remove' ? !part : part && part.status === op.status) continue;
    result.push({ shift_id: op.shift_id, name: op.name, status: op.status });
  }
  return result;
}

/**
 * Ops to remove a person from a shift: 'declined' if the person signed up themselves in the live
 * model (the wish stays visible as declined), else 'remove' (made by the coordination, or only
 * created in the lab).
 * @param {import('./model.js').CoordModel} liveModel
 * @param {string} key  person key
 * @param {string} name
 * @param {number} shiftId
 * @returns {LabOp[]}
 */
export function removeOps(liveModel, key, name, shiftId) {
  const part = liveModel.partIndex.get(`${shiftId}|${key}`);
  if (part && !part.addedByAdmin) return [{ shift_id: shiftId, name: part.name, status: 'declined' }];
  return [{ shift_id: shiftId, name: part ? part.name : name, status: 'remove' }];
}

/**
 * Ops to move a person from shift A to shift B (one undo step). fromShiftId null (or equal to B)
 * means "assign" (side panel, or wish to Einsatz). An existing wish in B is turned into the
 * Einsatz (same participation, no duplicate); nothing is added if the person is assigned in B
 * already. Nothing is removed from A if the lab has no live entry there.
 * @param {import('./model.js').CoordModel} liveModel
 * @param {import('./model.js').CoordModel} labModel  live model with the current ops applied
 * @param {string} key
 * @param {string} name
 * @param {number|null} fromShiftId
 * @param {number} toShiftId
 * @returns {LabOp[]}
 */
export function moveOps(liveModel, labModel, key, name, fromShiftId, toShiftId) {
  const ops = [];
  if (fromShiftId !== null && fromShiftId !== undefined && fromShiftId !== toShiftId) {
    const there = labModel.partIndex.get(`${fromShiftId}|${key}`);
    if (there && there.status !== 'declined') ops.push(...removeOps(liveModel, key, name, fromShiftId));
  }
  const here = labModel.partIndex.get(`${toShiftId}|${key}`);
  if (!here || here.status !== 'assigned') {
    ops.push({ shift_id: toShiftId, name: here ? here.name : name, status: 'assigned' });
  }
  return ops;
}

/**
 * Grid of one logical day for the lab: rows per shift type, lanes for simultaneous shifts, boxes
 * with their 5-minute grid columns and chips.
 * @param {import('./model.js').CoordModel} liveModel
 * @param {import('./model.js').CoordModel} labModel
 * @param {string} day  logical day key YYYY-MM-DD
 * @param {Set<string>} [changedPairs]  "shiftId|key" touched by the lab ops (chip marker)
 * @returns {{ columns:number, startMin:number, endMin:number,
 *             hours: Array<{hour:number, gridStart:number, gridSpan:number}>,
 *             rows: Array<{ type:Object, lanes: Array<Array<Box>> }> }}
 *   columns: number of 5-minute columns; gridStart is 0-based (CSS: gridStart + 1, span gridSpan).
 *   Box: { cs (lab shift), live (live shift), gridStart, gridSpan, einsaetze, required (-1 unlimited),
 *          chips: Array<{ key, name, kind:'einsatz'|'wish', level, changed, overlapIds:number[] }> }
 *   Chips: Einsätze first, then wishes, each by name. overlapIds are the other shifts at the same
 *   time in which the person has an Einsatz (only for Einsatz chips; ⚠ if not empty).
 *   Declined participations are not shown. Rows are in type order (sort_order), only types with shifts.
 */
export function buildLabGrid(liveModel, labModel, day, changedPairs = new Set()) {
  const base = dayStartMs(labModel, day);
  const items = [];
  for (const cs of labModel.shifts.values()) if (cs.day === day) items.push({ cs, ...offsets(cs, base) });
  const { startMin, endMin, hours } = span(labModel, items);
  const columns = Math.round((endMin - startMin) / GRID_MINUTES);

  const column = (min) => Math.round((min - startMin) / GRID_MINUTES);
  const boxOf = (it) => {
    const from = Math.max(it.start, startMin);
    const to = Math.min(it.end, endMin);
    const gridStart = Math.min(Math.max(0, Math.floor((from - startMin) / GRID_MINUTES)), Math.max(0, columns - 1));
    const gridEnd = Math.min(columns, Math.max(gridStart + 1, Math.ceil((to - startMin) / GRID_MINUTES)));
    const cs = it.cs;
    const chip = (key, kind) => {
      const person = labModel.persons.get(key);
      const overlapIds = kind !== 'einsatz' ? [] : (person?.items || [])
        .filter(o => o.id !== cs.id && o.start < cs.end && cs.start < o.end).map(o => o.id);
      return {
        key, name: person?.name || key, kind, level: person?.level ?? -1,
        changed: changedPairs.has(`${cs.id}|${key}`), overlapIds,
      };
    };
    const byName = (a, b) => a.name.localeCompare(b.name);
    const chips = [
      ...cs.assigned.map(k => chip(k, 'einsatz')).sort(byName),
      ...cs.wishes.map(k => chip(k, 'wish')).sort(byName),
    ];
    return {
      cs, live: liveModel.shifts.get(cs.id) || null,
      gridStart, gridSpan: gridEnd - gridStart,
      einsaetze: cs.assigned.length, required: cs.required, chips,
    };
  };

  const byType = new Map();
  for (const it of items) {
    if (!byType.has(it.cs.type.id)) byType.set(it.cs.type.id, []);
    byType.get(it.cs.type.id).push(it);
  }
  const rows = [...byType.values()]
    .sort((a, b) => (a[0].cs.type.sort_order ?? 0) - (b[0].cs.type.sort_order ?? 0) || a[0].cs.type.id - b[0].cs.type.id)
    .map(list => {
      const laid = list.map(it => ({ it, startMinutes: it.start, endMinutes: it.end, durationMinutes: it.end - it.start }));
      const count = assignOverlapLanes(laid);
      const lanes = Array.from({ length: count }, () => []);
      for (const l of laid.sort((a, b) => a.startMinutes - b.startMinutes || a.it.cs.id - b.it.cs.id)) {
        lanes[l.laneIndex].push(boxOf(l.it));
      }
      return { type: list[0].cs.type, lanes };
    });

  return {
    columns, startMin, endMin,
    hours: hours.map((hour, i) => ({ hour, gridStart: column(startMin + i * 60), gridSpan: 60 / GRID_MINUTES })),
    rows,
  };
}

/** Numbers of a person in one model (zeros if the person does not exist there). */
function stats(person) {
  return {
    hours: person ? person.hours : 0,
    load: person ? person.load : 0,
    popularScore: person ? person.popularScore : 0,
    warnings: person ? person.warnings : [],
    perDayMin: person ? person.perDayMin : new Map(),
  };
}

const warningId = (w) => `${w.kind}|${[...w.shiftIds].sort((a, b) => a - b).join(',')}`;

/**
 * Per person: hours, load and popular score live vs lab, minutes per day live vs lab, warnings.
 * @param {import('./model.js').CoordModel} liveModel
 * @param {import('./model.js').CoordModel} labModel
 * @returns {Map<string, { key:string, name:string, level:number, isNew:boolean,
 *   live:{hours:number, load:number, popularScore:number, warnings:Array, perDayMin:Map<string,number>},
 *   lab:{hours:number, load:number, popularScore:number, warnings:Array, perDayMin:Map<string,number>},
 *   hoursDelta:number, loadDelta:number, scoreDelta:number,
 *   perDay: Map<string, {live:number, lab:number}>,
 *   newWarnings: Array }>}
 *   isNew: not in the live model (added in the lab). newWarnings: lab warnings the live model does
 *   not have (what a change of the lab caused). Persons in the live model come first.
 */
export function personDeltas(liveModel, labModel) {
  const result = new Map();
  const keys = new Set([...liveModel.persons.keys(), ...labModel.persons.keys()]);
  for (const key of keys) {
    const lp = liveModel.persons.get(key) || null;
    const bp = labModel.persons.get(key) || null;
    const live = stats(lp);
    const lab = stats(bp);
    const perDay = new Map();
    for (const day of new Set([...live.perDayMin.keys(), ...lab.perDayMin.keys()])) {
      perDay.set(day, { live: live.perDayMin.get(day) || 0, lab: lab.perDayMin.get(day) || 0 });
    }
    const liveWarn = new Set(live.warnings.map(warningId));
    result.set(key, {
      key,
      name: (bp || lp).name,
      level: (bp || lp).level,
      isNew: !lp,
      live,
      lab,
      hoursDelta: lab.hours - live.hours,
      loadDelta: lab.load - live.load,
      scoreDelta: lab.popularScore - live.popularScore,
      perDay,
      newWarnings: lab.warnings.filter(w => !liveWarn.has(warningId(w))),
    });
  }
  return result;
}

/**
 * Summary for the apply dialog: number of ops per target status and the persons that have a
 * simultaneous double booking (overlap warning) in the lab model.
 * @param {import('./model.js').CoordModel} liveModel
 * @param {import('./model.js').CoordModel} labModel
 * @param {LabOp[]} ops  normally the normalized ops
 * @returns {{ assigned:number, declined:number, removed:number, overlaps:string[] }}
 *   overlaps: names, sorted. Every person with an overlap in the lab model is listed, also one
 *   that already existed live (the lab did not necessarily cause it).
 */
export function labSummary(liveModel, labModel, ops) {
  const count = (status) => (ops || []).filter(op => op.status === status).length;
  const overlaps = [...labModel.persons.values()]
    .filter(p => p.warnings.some(w => w.kind === 'overlap'))
    .map(p => p.name)
    .sort((a, b) => a.localeCompare(b));
  return { assigned: count('assigned'), declined: count('declined'), removed: count('remove'), overlaps };
}
