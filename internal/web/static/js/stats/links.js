/**
 * Deep links from the statistics page into the plan view.
 * The plan view reads these parameters once on load (see shift-plan-view.js).
 */

/**
 * @param {string} token plan token
 * @param {{ type?: number|number[], day?: string, status?: string, q?: string, shift?: number, view?: string }} params
 * @returns {string}
 */
export function planUrl(token, { type, day, status, q, shift, view = 'list' } = {}) {
  const sp = new URLSearchParams();
  if (view) sp.set('view', view);
  const types = Array.isArray(type) ? type : type != null ? [type] : [];
  if (types.length) sp.set('type', types.join(','));
  if (day) sp.set('day', day);
  if (status) sp.set('status', status);
  if (q) sp.set('q', q);
  if (shift != null) sp.set('shift', String(shift));
  const query = sp.toString();
  return `/plans/${encodeURIComponent(token)}${query ? `?${query}` : ''}`;
}
