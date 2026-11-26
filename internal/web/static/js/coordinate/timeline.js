/**
 * Data for the desktop views "Zeitleiste" and "Personen" (prj/coordination-refinement-plan.md, 4.4).
 *
 * Pure functions without Alpine or DOM. One logical day at a time; positions are percentages of the
 * time span, so the UI only has to set `left`/`width`. Times are floating wall clock (datetime.js):
 * no local getters, minutes are offsets from the plan's day boundary.
 */
import { parseDate } from '../datetime.js';
import { assignOverlapLanes } from '../timetable-view-layout.js';
import { hardRule } from './rank.js';

const MS_MINUTE = 60 * 1000;
const DAY_MIN = 24 * 60;

/** Start of the logical day in ms (floating): midnight of the day key plus the boundary. */
export function dayStartMs(model, day) {
  return parseDate(day).getTime() + model.ctx.boundary * MS_MINUTE;
}

/** Minute offsets of a shift from the logical day start; the end is cut at the end of the day. */
export function offsets(cs, base) {
  const start = (cs.start.getTime() - base) / MS_MINUTE;
  const end = Math.min((cs.end.getTime() - base) / MS_MINUTE, DAY_MIN);
  return { start, end };
}

/**
 * Time span covering the given offsets, rounded to full clock hours (clamped to the day).
 * @returns {{startMin:number, endMin:number, hours:number[]}}
 *   hours: clock hour (0-23) of each column, one per full hour in the span
 */
export function span(model, list) {
  if (!list.length) return { startMin: 0, endMin: 0, hours: [] };
  const boundary = model.ctx.boundary;
  const first = Math.min(...list.map(o => o.start));
  const last = Math.max(...list.map(o => o.end));
  // Round in wall-clock terms (boundary may not be a full hour), then clamp to the day
  const startMin = Math.max(0, Math.floor((first + boundary) / 60) * 60 - boundary);
  let endMin = Math.min(DAY_MIN, Math.ceil((last + boundary) / 60) * 60 - boundary);
  if (endMin <= startMin) endMin = Math.min(DAY_MIN, startMin + 60);
  const hours = [];
  for (let m = startMin; m < endMin; m += 60) hours.push(Math.floor(((m + boundary) % DAY_MIN) / 60));
  return { startMin, endMin, hours };
}

/** left/width in percent of the span for the offsets [from, to), clipped to the span. */
function place(from, to, startMin, endMin) {
  const total = endMin - startMin;
  const a = Math.max(from, startMin);
  const b = Math.min(to, endMin);
  return { left: ((a - startMin) / total) * 100, width: Math.max(0, ((b - a) / total) * 100) };
}

/**
 * Shifts of one logical day as rows per shift type.
 * @param {import('./model.js').CoordModel} model
 * @param {number[]} shiftIds  shifts after the current filter (type, status); the day filter is ignored
 * @param {string} day         logical day key YYYY-MM-DD
 * @returns {{ startMin:number, endMin:number, hours:number[],
 *             rows: Array<{ type:Object, lanes: Array<Array<{ cs:Object, left:number, width:number }>> }> }}
 *   startMin/endMin: minutes from the day boundary, rounded to full hours, spanning all shifts of the day;
 *   left/width: percent of the span; lanes: shifts of a type that overlap each other are in separate lanes.
 *   Rows are in the order of the types (sort_order), only types with shifts on the day.
 */
export function buildTimeline(model, shiftIds, day) {
  const base = dayStartMs(model, day);
  const items = [];
  for (const id of shiftIds) {
    const cs = model.shifts.get(id);
    if (cs && cs.day === day) items.push({ cs, ...offsets(cs, base) });
  }
  const { startMin, endMin, hours } = span(model, items);

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
        lanes[l.laneIndex].push({ cs: l.it.cs, ...place(l.it.start, l.it.end, startMin, endMin) });
      }
      return { type: list[0].cs.type, lanes };
    });
  return { startMin, endMin, hours, rows };
}

/** Spreads the bars of one person over lanes (overlapping bars below each other) and flags Einsatz overlaps. */
function layoutPersonLanes(bars) {
  const einsaetze = bars.filter(b => b.kind === 'einsatz');
  for (const b of einsaetze) {
    const other = einsaetze.find(o => o !== b && b.startMin < o.endMin && o.startMin < b.endMin);
    b.conflict = !!other;
    b.conflictWith = other ? other.cs : null;
  }
  const laid = bars.map(b => ({ b, startMinutes: b.startMin, endMinutes: b.endMin, durationMinutes: b.endMin - b.startMin }));
  const count = bars.length ? assignOverlapLanes(laid) : 1;
  const lanes = Array.from({ length: count }, () => []);
  for (const l of laid.sort((a, c) => a.startMinutes - c.startMinutes || a.b.cs.id - c.b.cs.id)) lanes[l.laneIndex].push(l.b);
  return lanes;
}

/**
 * Einsätze and open wishes per person on one logical day.
 * @param {import('./model.js').CoordModel} model
 * @param {string} day
 * @param {number|null} selectedShiftId
 * @returns {{ startMin:number, endMin:number, hours:number[],
 *             selection: {left:number, width:number}|null,   // span of the selected shift (if on this day)
 *             rows: Array<{ person:Object,
 *                           lanes: Array<Array<{ cs:Object, kind:'einsatz'|'wish', conflict:boolean, left:number, width:number }>>,
 *                             // simultaneous bars (e.g. two wishes at the same time) are stacked in separate lanes;
 *                             // conflict: two Einsätze of the person overlap (a wish next to something else is a choice)
 *                           absent: Array<{ left:number, width:number }>,   // outside the attendance
 *                           freeDuring: boolean|null }> }}                  // free in the selected shift (null without selection)
 *   Rows are in model.persons order; the UI sorts them like the person column. The span covers all
 *   shifts of the day (and is empty without any).
 */
export function buildPersonTimeline(model, day, selectedShiftId) {
  const base = dayStartMs(model, day);
  const items = [];
  for (const cs of model.shifts.values()) if (cs.day === day) items.push({ cs, ...offsets(cs, base) });
  const { startMin, endMin, hours } = span(model, items);

  const barsByKey = new Map();
  const push = (key, bar) => {
    let list = barsByKey.get(key);
    if (!list) barsByKey.set(key, list = []);
    list.push(bar);
  };
  for (const it of items) {
    const pos = place(it.start, it.end, startMin, endMin);
    const times = { startMin: it.start, endMin: it.end };
    for (const key of it.cs.assigned) push(key, { cs: it.cs, kind: 'einsatz', conflict: false, conflictWith: null, ...pos, ...times });
    for (const key of it.cs.wishes) push(key, { cs: it.cs, kind: 'wish', conflict: false, conflictWith: null, ...pos, ...times });
  }

  const selected = selectedShiftId ? model.shifts.get(selectedShiftId) || null : null;
  const selOff = selected && selected.day === day ? offsets(selected, base) : null;
  const selection = selOff && endMin > startMin ? place(selOff.start, selOff.end, startMin, endMin) : null;

  const rows = [];
  for (const person of model.persons.values()) {
    const absent = [];
    if (endMin > startMin) {
      const { from, until } = person.attendance;
      if (from) {
        const f = (from.getTime() - base) / MS_MINUTE;
        if (f > startMin) absent.push(place(startMin, f, startMin, endMin));
      }
      if (until) {
        const u = (until.getTime() - base) / MS_MINUTE;
        if (u < endMin) absent.push(place(u, endMin, startMin, endMin));
      }
    }
    let freeDuring = null;
    if (selected) {
      freeDuring = !selected.assigned.includes(person.key) && !selected.wishes.includes(person.key)
        && hardRule(model, person, selected) === null;
    }
    rows.push({ person, lanes: layoutPersonLanes(barsByKey.get(person.key) || []), absent, freeDuring });
  }
  return { startMin, endMin, hours, selection, rows };
}
