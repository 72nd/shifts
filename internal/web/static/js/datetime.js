/**
 * Floating local time helpers.
 *
 * Shift times and plan dates are "floating" wall-clock values without offset
 * (API format "YYYY-MM-DDTHH:MM:SS" and "YYYY-MM-DD"). They are held as Date
 * objects whose UTC fields contain the wall clock, e.g. "2025-03-30T01:30:00"
 * is Date.UTC(2025, 2, 30, 1, 30). Formatting always uses
 * Intl.DateTimeFormat with timeZone 'UTC'. The result is therefore identical in
 * every browser time zone and there are no daylight saving gaps.
 *
 * Outside this module no code may use: new Date(apiString), the local
 * getters/setters (getHours, getDate, setHours, ...), toISOString,
 * getTimezoneOffset, or toLocale*String without an explicit timeZone.
 */

const DATETIME_RE = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/;
const DATE_RE = /^(\d{4})-(\d{2})-(\d{2})$/;
const MS_MINUTE = 60 * 1000;
const MS_DAY = 24 * 60 * MS_MINUTE;

const pad = (n, len = 2) => String(n).padStart(len, '0');

/** Maps short language codes to the fixed 24h locales used by the app. */
function resolveLocale(locale) {
  if (!locale || locale === 'de') return 'de-DE';
  if (locale === 'en') return 'en-GB';
  return locale;
}

/** Builds a floating Date from parts; null if a part overflowed (e.g. Feb 30). */
function build(y, mo, d, h, mi, s) {
  const date = new Date(Date.UTC(y, mo - 1, d, h, mi, s));
  if (
    date.getUTCFullYear() !== y || date.getUTCMonth() !== mo - 1 || date.getUTCDate() !== d ||
    date.getUTCHours() !== h || date.getUTCMinutes() !== mi || date.getUTCSeconds() !== s
  ) {
    return null;
  }
  return date;
}

/**
 * Parses "YYYY-MM-DDTHH:MM[:SS]" into a floating Date.
 * Anything else (offsets, "Z", other formats) yields null and a console warning.
 * @param {string} str
 * @returns {Date|null}
 */
export function parseDateTime(str) {
  const m = typeof str === 'string' ? DATETIME_RE.exec(str) : null;
  const date = m ? build(+m[1], +m[2], +m[3], +m[4], +m[5], m[6] ? +m[6] : 0) : null;
  if (!date) console.warn('datetime: invalid date-time value', str);
  return date;
}

/**
 * Parses "YYYY-MM-DD" into a Date at 00:00 (UTC fields).
 * @param {string} str
 * @returns {Date|null}
 */
export function parseDate(str) {
  const m = typeof str === 'string' ? DATE_RE.exec(str) : null;
  const date = m ? build(+m[1], +m[2], +m[3], 0, 0, 0) : null;
  if (!date) console.warn('datetime: invalid date value', str);
  return date;
}

/**
 * Serializes to the API format "YYYY-MM-DDTHH:MM:SS".
 * @param {Date} d
 * @returns {string}
 */
export function serializeDateTime(d) {
  return `${serializeDate(d)}T${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}:${pad(d.getUTCSeconds())}`;
}

/**
 * Serializes to "YYYY-MM-DD".
 * @param {Date} d
 * @returns {string}
 */
export function serializeDate(d) {
  return `${pad(d.getUTCFullYear(), 4)}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())}`;
}

/**
 * Value for <input type="datetime-local">: "YYYY-MM-DDTHH:MM".
 * @param {Date} d
 * @returns {string}
 */
export function toInputValue(d) {
  return `${serializeDate(d)}T${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}`;
}

/**
 * Reads a <input type="datetime-local"> value; same as parseDateTime.
 * @param {string} str
 * @returns {Date|null}
 */
export function fromInputValue(str) {
  return parseDateTime(str);
}

/**
 * Returns a new Date moved by the given minutes (pure wall-clock arithmetic).
 * @param {Date} d
 * @param {number} minutes
 * @returns {Date}
 */
export function addMinutes(d, minutes) {
  return new Date(d.getTime() + minutes * MS_MINUTE);
}

/**
 * Returns a new Date moved by the given days (pure wall-clock arithmetic).
 * @param {Date} d
 * @param {number} days
 * @returns {Date}
 */
export function addDays(d, days) {
  return new Date(d.getTime() + days * MS_DAY);
}

/**
 * Minutes since midnight of the wall clock.
 * @param {Date} d
 * @returns {number}
 */
export function minutesOfDay(d) {
  return d.getUTCHours() * 60 + d.getUTCMinutes();
}

/**
 * Calendar day key "YYYY-MM-DD" (same as serializeDate).
 * @param {Date} d
 * @returns {string}
 */
export function dateKey(d) {
  return serializeDate(d);
}

/**
 * The logical day (Date at 00:00) a time belongs to: times before the daily
 * boundary (minutes after midnight) count towards the previous day.
 * @param {Date} d
 * @param {number} boundaryMinutes
 * @returns {Date}
 */
export function logicalDay(d, boundaryMinutes) {
  const midnight = new Date(Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate()));
  return minutesOfDay(d) < boundaryMinutes ? addDays(midnight, -1) : midnight;
}

/**
 * Whole days from a to b (b - a). Both should be at 00:00; rounds defensively.
 * @param {Date} a
 * @param {Date} b
 * @returns {number}
 */
export function daysBetween(a, b) {
  return Math.round((b.getTime() - a.getTime()) / MS_DAY);
}

/**
 * "HH:MM" in 24h notation. Locale: 'de' (de-DE, default), 'en' (en-GB) or a BCP 47 tag.
 * @param {Date} d
 * @param {string} [locale]
 * @returns {string}
 */
export function formatTime(d, locale) {
  return new Intl.DateTimeFormat(resolveLocale(locale), {
    timeZone: 'UTC', hour: '2-digit', minute: '2-digit', hourCycle: 'h23',
  }).format(d);
}

/**
 * Formats the date part with Intl ({ timeZone: 'UTC', ...options }).
 * Accepts a floating Date, or an API string ("YYYY-MM-DD" or "YYYY-MM-DDTHH:MM[:SS]"),
 * which is parsed first. Invalid input returns ''.
 * @param {Date|string} d
 * @param {string} [locale]
 * @param {Intl.DateTimeFormatOptions} [options]
 * @returns {string}
 */
export function formatDate(d, locale, options) {
  if (typeof d === 'string') {
    d = d.includes('T') ? parseDateTime(d) : parseDate(d);
  }
  if (!(d instanceof Date) || isNaN(d.getTime())) return '';
  return new Intl.DateTimeFormat(resolveLocale(locale), { ...options, timeZone: 'UTC' }).format(d);
}

/**
 * Current wall clock in an IANA time zone as a floating Date (seconds precision).
 * An unknown zone logs a warning and falls back to UTC.
 * @param {string} timeZone e.g. plan.time_zone
 * @returns {Date}
 */
export function nowInZone(timeZone) {
  let fmt;
  try {
    fmt = new Intl.DateTimeFormat('en-CA', {
      timeZone, hourCycle: 'h23',
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit', second: '2-digit',
    });
  } catch (e) {
    console.warn('datetime: unknown time zone', timeZone);
    fmt = new Intl.DateTimeFormat('en-CA', {
      timeZone: 'UTC', hourCycle: 'h23',
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit', second: '2-digit',
    });
  }
  const p = {};
  for (const part of fmt.formatToParts(new Date())) p[part.type] = part.value;
  return new Date(Date.UTC(+p.year, +p.month - 1, +p.day, +p.hour % 24, +p.minute, +p.second));
}
