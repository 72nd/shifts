/**
 * Statistics core: pure functions without Alpine or DOM access.
 *
 * All statistics shown in the web UI are computed here from the same data the
 * plan view loads. The server endpoint GET /api/plans/{token}/stats is only a
 * summary for API clients and is not used by the UI.
 *
 * Times: shift times are floating wall-clock values (see datetime.js).
 * participants[].created_at is a real instant (RFC3339 in UTC, e.g.
 * "2024-12-01T10:00:00Z"); it is converted to the plan's time zone with
 * instantInZone, so sign-up days match what the organisers see on the clock.
 */
import {
  parseDate, parseDateTime, addMinutes, addDays, logicalDay, dateKey,
  minutesOfDay, instantInZone,
} from '../datetime.js';

/** Default warning thresholds; the plan's min_rest_minutes / max_day_minutes replace them (see thresholds). */
export const REST_WARN_MIN = 8 * 60;
export const DAY_LOAD_WARN_MIN = 10 * 60;
export const NIGHT_START_MIN = 22 * 60;
export const NIGHT_END_MIN = 6 * 60;
export const DISTRIBUTION_MIN_PEOPLE = 5;

const MS_MINUTE = 60 * 1000;
const MS_DAY = 24 * 60 * MS_MINUTE;

/**
 * @typedef {{ typeIds: number[], days: string[] }} StatsFilter  empty array = all
 *
 * @typedef {Object} StatsModel
 * @property {Object} options  unfiltered choices for the filters: { days: string[], types: [{typeId, title}] }
 * @property {Object} totals   shifts, openShifts, unlimitedShifts, required, filledCapped, open,
 *   percent (0–100 or null), requiredMinutes, filledMinutes, doneMinutes, plannedMinutes,
 *   people, avgMinutesPerPerson, medianMinutesPerPerson
 * @property {Array}  days     [{ day, shifts, required, filledCapped, open, percent, filledMinutes }] ascending
 * @property {Array}  types    [{ typeId, title:{de,en}, shifts, required, filledCapped, open, percent, filledMinutes }] plan order
 * @property {Object} heatmap  { days: string[], hours: number[] (clock hour of each column),
 *   cells: [[{required, filled, ratio, openPeak}|null]] } (openPeak: most places missing at once in that hour)
 * @property {Object} typeHeatmap  { days, hours (as heatmap), rows: [{ typeId, title:{de,en}, cells: [[cell|null]] }] }
 *   one row per type with shifts, plan order; cells[dayIdx][hourIdx] like heatmap.cells
 * @property {Array}  people   [{ key, name, shifts, minutes, doneMinutes, plannedMinutes, nightMinutes, days, typeCount,
 *   maxDayMinutes, longestShiftMinutes, firstSignup, lastSignup, warnings:[{kind, shiftIds, minutes}], items:[...] }]
 * @property {Array}  possibleDuplicates  [[key, key]]
 * @property {Object} distribution { buckets:[{fromHours, toHours|null, count}], top20SharePercent|null }
 * @property {Object} signups  { points:[{at: Date, cumulative}], requiredTotal, planStart: Date,
 *   forecast: { date: Date|null, remainingAtStart: number|null } | null }
 * @property {Recap}  recap    figures and highlights for the recap tab (see buildRecap)
 *
 * @typedef {Object} Recap
 * @property {number} people          people.length (assignments only)
 * @property {number} minutes         sum of people[].minutes
 * @property {number} shifts          shifts with at least one assignment
 * @property {number} types           distinct areas among those shifts
 * @property {number} signups         participations (declined excluded)
 * @property {number} nightMinutes    sum of people[].nightMinutes
 * @property {number} avgShiftMinutes mean length of shifts with an assignment, rounded; 0 if none
 * @property {number|null} repeatPercent  % of people with at least two shifts; null if fewer than 3 people
 * @property {{value:number, at:Date, day:string}|null} peak  most people on assignment at once
 * @property {Array<RecapFact>} facts
 *
 * @typedef {Object} RecapFact
 * @property {string} id      most_shifts, most_days, longest_shift, early_bird, night_owl, all_rounder,
 *   dream_team, first_signup, fastest_full, last_minute, best_signup_day
 * @property {'people'|'signups'} group
 * @property {number} value
 * @property {string[]} names  shown names (1, or 2 for dream_team; empty for counts)
 * @property {number} others   further people (or pairs) with the same value, 0 if none
 * @property {number} [shiftId]  (longest_shift)
 * @property {Date} [at]       floating sign-up instant (first_signup) or shift start (fastest_full)
 * @property {Date} [end]      shift end (fastest_full)
 * @property {string} [day]    day key
 * @property {{de:string,en:string}} [typeTitle]  (fastest_full)
 */

/** Normalised person key: trimmed, single spaces, lower case, NFC. */
export function personKey(name) {
  return String(name || '').normalize('NFC').trim().replace(/\s+/g, ' ').toLocaleLowerCase();
}

/** Key without punctuation, spaces and diacritics, used for duplicate hints. */
function looseKey(key) {
  return key.normalize('NFD').replace(/\p{M}/gu, '').replace(/[\p{P}\p{S}\s]/gu, '');
}

/** "HH:MM" → minutes after midnight. */
export function boundaryMinutes(str) {
  const m = /^(\d{1,2}):(\d{2})/.exec(str || '');
  return m ? +m[1] * 60 + +m[2] : 0;
}

/**
 * Warning thresholds of the statistics: the plan's settings, else the defaults.
 * @returns {{restMin: number, dayMin: number}}
 */
export function thresholds(plan) {
  return {
    restMin: plan?.min_rest_minutes ?? REST_WARN_MIN,
    dayMin: plan?.max_day_minutes ?? DAY_LOAD_WARN_MIN,
  };
}

/**
 * Whether a shift counts as popular: the shift's own popularity setting, else the type's.
 * "auto" means more interested people (all statuses) than places.
 * @param {Object} shift  API shift (popularity may be null/undefined)
 * @param {Object} type   API shift type (popularity: auto|popular|normal)
 * @param {number} interest  participations of any status that people made themselves
 *   (entries added by the coordination are not interest)
 */
export function isPopular(shift, type, interest) {
  const mode = shift.popularity || type?.popularity || 'auto';
  if (mode === 'popular') return true;
  if (mode === 'normal') return false;
  return shift.required > 0 && interest > shift.required;
}

/**
 * Whether a participation is an assignment ("Einsatz") rather than an open wish or a decline:
 * assigned, or signed up in a shift that is not popular or is locked.
 */
export function isEinsatz(status, popular, locked) {
  if (status === 'assigned') return true;
  return status === 'signed_up' && (!popular || locked);
}

/**
 * Finds overlap, rest and day load warnings in one person's shifts.
 * @param {Array<{id:number, start:Date, end:Date, minutes:number, day:string}>} items  sorted by start
 * @param {number|null} restMin  minimum break between shifts (null = no check)
 * @param {number|null} dayMin   maximum minutes per logical day (null = no check)
 * @returns {Array<{kind:'overlap'|'rest'|'dayload', shiftIds:number[], minutes:number}>}
 */
export function findShiftWarnings(items, restMin, dayMin) {
  const warnings = [];
  let prev = null; // shift with latest end so far
  for (const s of items) {
    if (prev) {
      if (prev.end > s.start) {
        const overlap = (Math.min(prev.end, s.end) - s.start) / MS_MINUTE;
        warnings.push({ kind: 'overlap', shiftIds: [prev.id, s.id], minutes: Math.round(overlap) });
      } else if (restMin !== null) {
        const gap = (s.start - prev.end) / MS_MINUTE;
        if (gap < restMin) warnings.push({ kind: 'rest', shiftIds: [prev.id, s.id], minutes: Math.round(gap) });
      }
    }
    if (!prev || s.end > prev.end) prev = s;
  }
  if (dayMin !== null) {
    const perDay = new Map();
    for (const s of items) perDay.set(s.day, (perDay.get(s.day) || 0) + s.minutes);
    for (const [day, m] of perDay) {
      if (m > dayMin) warnings.push({ kind: 'dayload', shiftIds: items.filter(s => s.day === day).map(s => s.id), minutes: m });
    }
  }
  return warnings;
}

const pct = (filled, required) => (required > 0 ? Math.round((filled / required) * 100) : null);

/** Minutes of [start, end) that fall between NIGHT_START_MIN and NIGHT_END_MIN. */
function nightMinutes(start, end) {
  let total = 0;
  const first = addDays(new Date(Date.UTC(start.getUTCFullYear(), start.getUTCMonth(), start.getUTCDate())), -1);
  for (let d = first; d < end; d = addDays(d, 1)) {
    const ns = addMinutes(d, NIGHT_START_MIN).getTime();
    const ne = addMinutes(addDays(d, 1), NIGHT_END_MIN).getTime();
    const overlap = Math.min(ne, end.getTime()) - Math.max(ns, start.getTime());
    if (overlap > 0) total += overlap / MS_MINUTE;
  }
  return Math.round(total);
}

function median(values) {
  if (!values.length) return 0;
  const s = [...values].sort((a, b) => a - b);
  const mid = Math.floor(s.length / 2);
  return s.length % 2 ? s[mid] : Math.round((s[mid - 1] + s[mid]) / 2);
}

/** Normalises all shifts of all types (unfiltered). */
function normaliseShifts(plan, types, now, boundary) {
  const shifts = [];
  for (const type of types || []) {
    const typeTitle = { de: type.title_de || type.title_en || '', en: type.title_en || type.title_de || '' };
    for (const s of type.shifts || []) {
      const start = parseDateTime(s.start_time);
      if (!start) continue;
      const minutes = s.duration || 0;
      const end = addMinutes(start, minutes);
      const unlimited = s.required === -1;
      // Declined participations never occupy a place (they only count as interest).
      const interest = (s.participants || []).filter(p => !p.added_by_admin).length;
      const popular = isPopular(s, type, interest);
      const locked = !!s.is_locked;
      const participants = (s.participants || [])
        .filter(p => p.status !== 'declined')
        .map(p => ({
          key: personKey(p.name),
          name: String(p.name || '').trim().replace(/\s+/g, ' '),
          createdAt: p.created_at ? instantInZone(p.created_at, plan.time_zone) : null,
          // Entries made by the coordination are not a sign-up of the person.
          byAdmin: !!p.added_by_admin,
          // Open wishes (signed up for a popular, unlocked shift) are not assignments yet.
          einsatz: isEinsatz(p.status || 'signed_up', popular, locked),
        }));
      const filled = participants.length;
      const einsatzCount = participants.filter(p => p.einsatz).length;
      const required = unlimited ? null : Math.max(0, s.required || 0);
      shifts.push({
        id: s.id,
        typeId: type.id,
        typeTitle,
        start,
        end,
        minutes,
        required,
        filled,
        einsatzCount,
        filledCapped: unlimited ? 0 : Math.min(filled, required),
        open: unlimited ? 0 : Math.max(0, required - filled),
        locked,
        day: dateKey(logicalDay(start, boundary)),
        status: end <= now ? 'past' : start > now ? 'future' : 'current',
        participants,
      });
    }
  }
  shifts.sort((a, b) => a.start - b.start || a.id - b.id);
  return shifts;
}

function aggregate(list) {
  const a = { shifts: list.length, required: 0, filledCapped: 0, open: 0, filledMinutes: 0 };
  for (const s of list) {
    if (s.required !== null) {
      a.required += s.required;
      a.filledCapped += s.filledCapped;
      a.open += s.open;
    }
    a.filledMinutes += s.filled * s.minutes;
  }
  a.percent = pct(a.filledCapped, a.required);
  return a;
}

/**
 * Hour grid for the given shifts: cells[dayIndex][hourIndex] = {required, filled, ratio, openPeak} | null.
 * days: ordered day keys that define the rows of the grid (shifts on other days are ignored).
 * required/filled are weighted by the share of the hour a shift covers (they set the colour);
 * openPeak is the highest number of places missing at the same time within the hour (whole number).
 */
function buildGrid(shifts, boundary, days) {
  const dayIndex = new Map(days.map((d, i) => [d, i]));
  const cells = days.map(() => new Array(24).fill(null));
  for (const s of shifts) {
    // Walk the hour slots the shift touches; slot = logical day + hour offset from boundary.
    let slotStart = new Date(Math.floor((s.start.getTime() - boundary * MS_MINUTE) / (60 * MS_MINUTE)) * 60 * MS_MINUTE + boundary * MS_MINUTE);
    while (slotStart < s.end) {
      const slotEnd = addMinutes(slotStart, 60);
      const overlap = (Math.min(slotEnd, s.end) - Math.max(slotStart, s.start)) / MS_MINUTE;
      const day = dateKey(logicalDay(slotStart, boundary));
      const di = dayIndex.get(day);
      if (overlap > 0 && di !== undefined) {
        const offset = (minutesOfDay(slotStart) - boundary + 1440) % 1440;
        const hi = Math.floor(offset / 60);
        const share = overlap / 60;
        const cell = cells[di][hi] || (cells[di][hi] = { required: 0, filled: 0, ratio: null, openPeak: 0 });
        if (s.required === null) {
          cell.filled += s.filled * share;
        } else {
          cell.required += s.required * share;
          cell.filled += s.filledCapped * share;
          if (s.open > 0) {
            (cell.openParts || (cell.openParts = [])).push([
              Math.max(slotStart.getTime(), s.start.getTime()),
              Math.min(slotEnd.getTime(), s.end.getTime()),
              s.open,
            ]);
          }
        }
      }
      slotStart = slotEnd;
    }
  }
  for (const row of cells) {
    for (const c of row) {
      if (!c) continue;
      c.required = Math.round(c.required * 10) / 10;
      c.filled = Math.round(c.filled * 10) / 10;
      c.ratio = c.required > 0 ? Math.min(1, c.filled / c.required) : null;
      c.openPeak = peakOf(c.openParts || []);
      delete c.openParts;
    }
  }
  return cells;
}

/** Highest sum of `open` over overlapping [from, to) parts (sweep, ends before starts). */
function peakOf(parts) {
  const ev = [];
  for (const [from, to, open] of parts) ev.push([from, open], [to, -open]);
  ev.sort((a, b) => a[0] - b[0] || a[1] - b[1]);
  let cur = 0, peak = 0;
  for (const [, d] of ev) { cur += d; peak = Math.max(peak, cur); }
  return peak;
}

function buildHeatmap(shifts, boundary) {
  const days = [...new Set(shifts.map(s => s.day))].sort();
  const firstHour = Math.floor(boundary / 60);
  const hours = Array.from({ length: 24 }, (_, i) => (firstHour + i) % 24);
  return { days, hours, cells: buildGrid(shifts, boundary, days) };
}

/** Same grid per type: rows in plan order, only types with at least one shift. Days and hours as in heatmap. */
function buildTypeHeatmap(shifts, boundary, days, hours, typeOptions) {
  const rows = [];
  for (const t of typeOptions) {
    const list = shifts.filter(s => s.typeId === t.typeId);
    if (!list.length) continue;
    rows.push({ typeId: t.typeId, title: t.title, cells: buildGrid(list, boundary, days) });
  }
  return { days, hours, rows };
}

function buildPeople(shifts, boundary, limits) {
  const map = new Map();
  for (const s of shifts) {
    // Hours count only assignments, not open wishes.
    for (const p of s.participants) {
      if (!p.einsatz) continue;
      let person = map.get(p.key);
      if (!person) {
        person = { key: p.key, spellings: new Map(), items: [], signups: [] };
        map.set(p.key, person);
      }
      person.spellings.set(p.name, (person.spellings.get(p.name) || 0) + 1);
      // The same person signed up twice for one shift counts once.
      if (person.items[person.items.length - 1] !== s) person.items.push(s);
      if (p.createdAt) person.signups.push(p.createdAt);
    }
  }

  const people = [];
  for (const person of map.values()) {
    const name = [...person.spellings.entries()].sort((a, b) => b[1] - a[1])[0][0];
    const items = person.items;
    let minutes = 0, doneMinutes = 0, night = 0, longest = 0;
    const perDay = new Map();
    const typeIds = new Set();
    for (const s of items) {
      minutes += s.minutes;
      if (s.status === 'past') doneMinutes += s.minutes;
      night += nightMinutes(s.start, s.end);
      longest = Math.max(longest, s.minutes);
      perDay.set(s.day, (perDay.get(s.day) || 0) + s.minutes);
      typeIds.add(s.typeId);
    }

    const warnings = findShiftWarnings(items, limits.restMin, limits.dayMin);

    const signups = person.signups.sort((a, b) => a - b);
    people.push({
      key: person.key,
      name,
      shifts: items.length,
      minutes,
      doneMinutes,
      plannedMinutes: minutes - doneMinutes,
      nightMinutes: night,
      days: perDay.size,
      typeCount: typeIds.size,
      maxDayMinutes: Math.max(0, ...perDay.values()),
      longestShiftMinutes: longest,
      firstSignup: signups[0] || null,
      lastSignup: signups[signups.length - 1] || null,
      warnings,
      items: items.map(s => ({
        shiftId: s.id, typeId: s.typeId, typeTitle: s.typeTitle, day: s.day,
        start: s.start, end: s.end, minutes: s.minutes, status: s.status,
      })),
    });
  }
  people.sort((a, b) => a.name.localeCompare(b.name));
  return people;
}

function findDuplicates(people) {
  const groups = new Map();
  for (const p of people) {
    const k = looseKey(p.key);
    if (!k) continue;
    if (!groups.has(k)) groups.set(k, []);
    groups.get(k).push(p.key);
  }
  const pairs = [];
  for (const keys of groups.values()) {
    for (let i = 1; i < keys.length; i++) pairs.push([keys[0], keys[i]]);
  }
  return pairs;
}

function buildDistribution(people) {
  const buckets = Array.from({ length: 11 }, (_, i) => ({ fromHours: i * 2, toHours: i < 10 ? i * 2 + 2 : null, count: 0 }));
  for (const p of people) buckets[Math.min(10, Math.floor(p.minutes / 120))].count++;
  let top20SharePercent = null;
  if (people.length >= DISTRIBUTION_MIN_PEOPLE) {
    const sorted = people.map(p => p.minutes).sort((a, b) => b - a);
    const total = sorted.reduce((a, b) => a + b, 0);
    const n = Math.max(1, Math.ceil(people.length * 0.2));
    const top = sorted.slice(0, n).reduce((a, b) => a + b, 0);
    top20SharePercent = total > 0 ? Math.round((top / total) * 100) : null;
  }
  // Drop empty buckets at the end so the chart is not mostly empty.
  while (buckets.length > 1 && buckets[buckets.length - 1].count === 0) buckets.pop();
  return { buckets, top20SharePercent };
}

function buildSignups(shifts, totals, plan, now, boundary) {
  const times = [];
  for (const s of shifts) for (const p of s.participants) if (p.createdAt) times.push(p.createdAt);
  times.sort((a, b) => a - b);
  const points = times.map((at, i) => ({ at, cumulative: i + 1 }));
  const planStartDay = parseDate(plan.start_date);
  const planStart = planStartDay ? addMinutes(planStartDay, boundary) : null;

  let forecast = null;
  if (planStart && now < planStart && totals.open > 0) {
    const weekAgo = addDays(now, -7);
    const recent = times.filter(t => t > weekAgo && t <= now).length;
    if (recent >= 3) {
      const perDay = recent / 7;
      const fullAt = new Date(now.getTime() + (totals.open / perDay) * MS_DAY);
      if (fullAt <= planStart) {
        forecast = { date: fullAt, remainingAtStart: null };
      } else {
        const daysLeft = (planStart - now) / MS_DAY;
        forecast = { date: null, remainingAtStart: Math.max(0, Math.round(totals.open - perDay * daysLeft)) };
      }
    }
  }
  return { points, requiredTotal: totals.required, filledCapped: totals.filledCapped, planStart, forecast };
}

/**
 * Person(s) with the highest (or lowest) value; null if no one qualifies.
 * people are sorted by name, so the first of a tie is the alphabetically first.
 * min applies only when looking for the highest value; null/undefined values are skipped.
 */
function leaders(people, valueOf, { min = 1, lowest = false } = {}) {
  let best = null, list = [];
  for (const p of people) {
    const v = valueOf(p);
    if (v === null || v === undefined || (!lowest && v < min)) continue;
    if (best === null || (lowest ? v < best : v > best)) { best = v; list = [p]; }
    else if (v === best) list.push(p);
  }
  return best === null ? null : { value: best, names: [list[0].name], others: list.length - 1, first: list[0] };
}

/** Most people on assignment at the same time (sweep over start/end events). */
function peakAtOnce(shifts, boundary) {
  const events = [];
  for (const s of shifts) {
    if (!s.einsatzCount) continue;
    events.push([s.start.getTime(), s.einsatzCount], [s.end.getTime(), -s.einsatzCount]);
  }
  events.sort((a, b) => a[0] - b[0] || a[1] - b[1]);
  let current = 0, peak = 0, peakAt = null;
  for (const [t, delta] of events) {
    current += delta;
    if (current > peak) { peak = current; peakAt = new Date(t); }
  }
  return peak > 0 ? { value: peak, at: peakAt, day: dateKey(logicalDay(peakAt, boundary)) } : null;
}

/** Pair of people with the most shifts together (assignments only), or null below 2 shared shifts. */
function dreamTeam(shifts, people) {
  const nameOf = new Map(people.map(p => [p.key, p.name]));
  const counts = new Map();
  for (const s of shifts) {
    const keys = [...new Set(s.participants.filter(p => p.einsatz).map(p => p.key))].sort();
    for (let i = 0; i < keys.length; i++) {
      for (let j = i + 1; j < keys.length; j++) {
        const k = keys[i] + '\n' + keys[j];
        counts.set(k, (counts.get(k) || 0) + 1);
      }
    }
  }
  let best = null, ties = 0;
  for (const [k, n] of counts) {
    if (n < 2) continue;
    const names = k.split('\n').map(key => nameOf.get(key) || key).sort((a, b) => a.localeCompare(b));
    const label = names.join('\n');
    if (!best || n > best.value) { best = { value: n, names, label }; ties = 0; }
    else if (n === best.value) {
      ties++;
      if (label.localeCompare(best.label) < 0) best = { value: n, names, label };
    }
  }
  return best ? { value: best.value, names: best.names, others: ties } : null;
}

/** Shift that reached its required places fastest by the people's own sign-ups (at least 3 places). */
function fastestFull(shifts) {
  let best = null;
  for (const s of shifts) {
    if (s.required === null || s.required < 3 || s.filled < s.required) continue;
    // Entries of the coordination are written in bulk with one timestamp; only own sign-ups count.
    const times = s.participants.filter(p => !p.byAdmin && p.createdAt).map(p => p.createdAt.getTime()).sort((a, b) => a - b);
    if (times.length < s.required) continue;
    const value = Math.round((times[s.required - 1] - times[0]) / MS_MINUTE);
    // Shifts are sorted by start, so a tie keeps the earlier shift.
    if (!best || value < best.value) best = { value, shift: s };
  }
  if (!best) return null;
  const s = best.shift;
  return { value: best.value, names: [], others: 0, at: s.start, end: s.end, day: s.day, typeTitle: s.typeTitle };
}

/** Figures and highlights for the recap tab; hours count assignments only (as in the people tab). */
function buildRecap(shifts, people, points, boundary) {
  const withEinsatz = shifts.filter(s => s.einsatzCount > 0);
  const sumOf = (list, f) => list.reduce((a, x) => a + f(x), 0);
  const recap = {
    people: people.length,
    minutes: sumOf(people, p => p.minutes),
    shifts: withEinsatz.length,
    types: new Set(withEinsatz.map(s => s.typeId)).size,
    signups: sumOf(shifts, s => s.participants.length),
    nightMinutes: sumOf(people, p => p.nightMinutes),
    avgShiftMinutes: withEinsatz.length ? Math.round(sumOf(withEinsatz, s => s.minutes) / withEinsatz.length) : 0,
    repeatPercent: people.length >= 3 ? Math.round((people.filter(p => p.shifts >= 2).length / people.length) * 100) : null,
    peak: peakAtOnce(shifts, boundary),
    facts: [],
  };

  const facts = recap.facts;
  const add = (id, group, r, extra = {}) => {
    if (r) facts.push({ id, group, value: r.value, names: r.names, others: r.others, ...extra });
  };

  // People
  add('most_shifts', 'people', leaders(people, p => p.shifts, { min: 2 }));
  add('most_days', 'people', leaders(people, p => p.days, { min: 2 }));
  const longest = leaders(people, p => p.longestShiftMinutes);
  if (longest) {
    const item = longest.first.items.find(i => i.minutes === longest.value);
    add('longest_shift', 'people', longest, { shiftId: item?.shiftId });
  }
  add('early_bird', 'people', leaders(people, p => {
    let earliest = null;
    for (const i of p.items) {
      const m = minutesOfDay(i.start);
      if (m >= 180 && m < 720 && (earliest === null || m < earliest)) earliest = m;
    }
    return earliest;
  }, { lowest: true }));
  add('night_owl', 'people', leaders(people, p => p.nightMinutes));
  add('all_rounder', 'people', leaders(people, p => p.typeCount, { min: 2 }));
  add('dream_team', 'people', dreamTeam(shifts, people));

  // Sign-ups
  let first = null;
  let lastMinute = 0;
  for (const s of shifts) {
    const dayBefore = addMinutes(s.start, -24 * 60);
    for (const p of s.participants) {
      if (p.byAdmin || !p.createdAt) continue;
      if (!first || p.createdAt < first.createdAt) first = p;
      if (p.createdAt >= dayBefore && p.createdAt < s.start) lastMinute++;
    }
  }
  if (first) facts.push({ id: 'first_signup', group: 'signups', value: first.createdAt.getTime(), names: [first.name], others: 0, at: first.createdAt });
  const fastest = fastestFull(shifts);
  if (fastest) facts.push({ id: 'fastest_full', group: 'signups', ...fastest });
  if (lastMinute > 0) facts.push({ id: 'last_minute', group: 'signups', value: lastMinute, names: [], others: 0 });

  const perDay = new Map();
  for (const p of points) {
    const day = dateKey(p.at);
    perDay.set(day, (perDay.get(day) || 0) + 1);
  }
  let bestDay = null;
  for (const [day, count] of perDay) if (!bestDay || count > bestDay[1]) bestDay = [day, count];
  if (bestDay) facts.push({ id: 'best_signup_day', group: 'signups', value: bestDay[1], names: [], others: 0, day: bestDay[0] });
  return recap;
}

/**
 * Computes all statistics for the stats page.
 * @param {Object} plan  response of GET /api/plans/{token}
 * @param {Array} types  response of GET /api/plans/{token}/types (types with shifts and participants)
 * @param {StatsFilter} filter
 * @param {Date} now  floating Date, nowInZone(plan.time_zone)
 * @returns {StatsModel}
 */
export function computeStats(plan, types, filter, now) {
  const boundary = boundaryMinutes(plan.day_boundary_time);
  const all = normaliseShifts(plan, types, now, boundary);

  const options = {
    days: [...new Set(all.map(s => s.day))].sort(),
    types: (types || []).map(t => ({ typeId: t.id, title: { de: t.title_de || t.title_en, en: t.title_en || t.title_de } })),
  };

  const typeIds = new Set(filter?.typeIds || []);
  const days = new Set(filter?.days || []);
  const shifts = all.filter(s => (!typeIds.size || typeIds.has(s.typeId)) && (!days.size || days.has(s.day)));

  const sum = aggregate(shifts);
  const people = buildPeople(shifts, boundary, thresholds(plan));
  let requiredMinutes = 0, filledMinutes = 0, doneMinutes = 0;
  for (const s of shifts) {
    if (s.required !== null) requiredMinutes += s.required * s.minutes;
    filledMinutes += s.filled * s.minutes;
    if (s.status === 'past') doneMinutes += s.filled * s.minutes;
  }
  const personMinutes = people.map(p => p.minutes);
  const totals = {
    shifts: sum.shifts,
    openShifts: shifts.filter(s => s.open > 0).length,
    unlimitedShifts: shifts.filter(s => s.required === null).length,
    required: sum.required,
    filledCapped: sum.filledCapped,
    open: sum.open,
    percent: sum.percent,
    requiredMinutes,
    filledMinutes,
    doneMinutes,
    plannedMinutes: filledMinutes - doneMinutes,
    people: people.length,
    avgMinutesPerPerson: people.length ? Math.round(personMinutes.reduce((a, b) => a + b, 0) / people.length) : 0,
    medianMinutesPerPerson: median(personMinutes),
  };

  const byDay = new Map();
  for (const s of shifts) {
    if (!byDay.has(s.day)) byDay.set(s.day, []);
    byDay.get(s.day).push(s);
  }
  const dayStats = [...byDay.keys()].sort().map(day => {
    const { shifts: n, required, filledCapped, open, percent, filledMinutes: fm } = aggregate(byDay.get(day));
    return { day, shifts: n, required, filledCapped, open, percent, filledMinutes: fm };
  });

  const typeStats = [];
  for (const t of options.types) {
    const list = shifts.filter(s => s.typeId === t.typeId);
    if (!list.length) continue;
    const { shifts: n, required, filledCapped, open, percent, filledMinutes: fm } = aggregate(list);
    typeStats.push({ typeId: t.typeId, title: t.title, shifts: n, required, filledCapped, open, percent, filledMinutes: fm });
  }

  const signups = buildSignups(shifts, totals, plan, now, boundary);
  const heatmap = buildHeatmap(shifts, boundary);

  return {
    options,
    totals,
    days: dayStats,
    types: typeStats,
    heatmap,
    typeHeatmap: buildTypeHeatmap(shifts, boundary, heatmap.days, heatmap.hours, options.types),
    people,
    possibleDuplicates: findDuplicates(people),
    distribution: buildDistribution(people),
    signups,
    recap: buildRecap(shifts, people, signups.points, boundary),
  };
}
