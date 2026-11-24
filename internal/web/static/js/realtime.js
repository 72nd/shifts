/**
 * Realtime client for one shift plan (Server-Sent Events).
 *
 * The server only sends invalidations; this module coalesces them and calls
 * onChange(scope, rev) so the caller can reload data. onChange may return a
 * Promise: reloads are serialised on it (one at a time, at most one follow-up).
 *
 * Server events on /api/plans/{token}/events:
 *   hello    {"instance","rev","viewers"}
 *   change   {"rev","scope","viewers"}
 *   presence {"viewers"}
 */

const JITTER_MAX_MS = 400;
const HIDDEN_CLOSE_MS = 60 * 1000;
// Strongest scope wins. 'deleted' is handled immediately and never queued.
const SCOPE_RANK = { participants: 1, shifts: 2, types: 3, plan: 4 };

export class PlanRealtime {
  /**
   * @param {string} token plan token
   * @param {{onChange?: (scope: string, rev: number) => (void|Promise<void>),
   *          onStatus?: (status: 'live'|'connecting'|'offline') => void,
   *          onViewers?: (count: number) => void}} handlers
   */
  constructor(token, { onChange, onStatus, onViewers } = {}) {
    this.token = token;
    this.onChange = onChange || (() => {});
    this.onStatus = onStatus || (() => {});
    this.onViewers = onViewers || (() => {});
    this.status = null;
    this.source = null;
    this.instance = null;
    this.rev = null;
    this.stopped = true;
    this.helloSeen = false; // false until hello arrives on the current connection
    this.pendingScope = null;
    this.running = false;
    this.timer = null;
    this.hiddenTimer = null;
    this._onVisibility = () => this.handleVisibility();
  }

  /** Opens the connection. Safe to call again after stop(). */
  start() {
    if (!this.stopped) return;
    this.stopped = false;
    document.addEventListener('visibilitychange', this._onVisibility);
    if (document.hidden) {
      this.handleVisibility();
    } else {
      this.connect();
    }
  }

  /** Closes the connection and cancels all pending work. */
  stop() {
    this.stopped = true;
    document.removeEventListener('visibilitychange', this._onVisibility);
    clearTimeout(this.timer);
    clearTimeout(this.hiddenTimer);
    this.timer = this.hiddenTimer = null;
    this.pendingScope = null;
    this.close();
  }

  close() {
    if (this.source) {
      this.source.close();
      this.source = null;
    }
  }

  setStatus(status) {
    if (this.status === status) return;
    this.status = status;
    this.onStatus(status);
  }

  connect() {
    if (this.stopped || this.source) return;
    this.setStatus('connecting');
    this.helloSeen = false;
    const es = new EventSource(`/api/plans/${encodeURIComponent(this.token)}/events`);
    this.source = es;

    es.addEventListener('hello', (e) => {
      const data = parse(e);
      if (!data) return;
      const restarted =
        (this.instance !== null && data.instance !== this.instance) ||
        (this.rev !== null && data.rev < this.rev);
      // After any (re)connect something may have been missed: reload once.
      const reconnect = this.helloSeen === false && this.instance !== null;
      this.instance = data.instance;
      this.rev = data.rev;
      this.helloSeen = true;
      this.setStatus('live');
      this.onViewers(data.viewers);
      if (restarted) {
        this.schedule('plan');
      } else if (reconnect) {
        this.schedule('participants');
      }
    });

    es.addEventListener('change', (e) => {
      const data = parse(e);
      if (!data) return;
      const restarted = this.rev !== null && data.rev < this.rev;
      this.rev = data.rev;
      if (typeof data.viewers === 'number') this.onViewers(data.viewers);
      if (data.scope === 'deleted') {
        // Plan is gone: no jitter, nothing to coalesce.
        clearTimeout(this.timer);
        this.timer = null;
        this.pendingScope = null;
        this.fire('deleted', data.rev);
        return;
      }
      this.schedule(restarted ? 'plan' : data.scope);
    });

    es.addEventListener('presence', (e) => {
      const data = parse(e);
      if (data && typeof data.viewers === 'number') this.onViewers(data.viewers);
    });

    es.onerror = () => {
      if (this.stopped || this.source !== es) return;
      if (es.readyState === EventSource.CLOSED) {
        // Browser gave up (e.g. HTTP 404 when realtime is disabled).
        this.close();
        this.setStatus('offline');
        return;
      }
      // Browser retries on its own. Probe once to tell "disabled" from "down".
      this.setStatus('connecting');
      if (!this.probing) this.probe(es);
    };
  }

  /** One fetch to see whether the endpoint exists at all (404 = disabled). */
  async probe(es) {
    this.probing = true;
    try {
      const ctrl = new AbortController();
      const res = await fetch(`/api/plans/${encodeURIComponent(this.token)}/events`, {
        signal: ctrl.signal,
        headers: { Accept: 'text/event-stream' },
      });
      ctrl.abort(); // only the status matters; do not hold a stream open
      if (res.status === 404 && this.source === es && !this.stopped) {
        this.close();
        this.setStatus('offline');
      }
    } catch (_) {
      // Network down: EventSource keeps retrying.
    } finally {
      this.probing = false;
    }
  }

  handleVisibility() {
    if (this.stopped) return;
    if (document.hidden) {
      clearTimeout(this.hiddenTimer);
      this.hiddenTimer = setTimeout(() => {
        this.close();
        this.setStatus('offline');
      }, HIDDEN_CLOSE_MS);
      return;
    }
    clearTimeout(this.hiddenTimer);
    this.hiddenTimer = null;
    if (!this.source) {
      this.connect(); // hello after reconnect triggers the reload
    } else if (this.source) {
      this.schedule('participants'); // short absence: connection kept, refresh once
    }
  }

  /** Queues a reload after random jitter; the strongest scope wins. */
  schedule(scope) {
    if (this.stopped) return;
    if (!this.pendingScope || (SCOPE_RANK[scope] || 0) > (SCOPE_RANK[this.pendingScope] || 0)) {
      this.pendingScope = scope;
    }
    if (this.running || this.timer) return; // picked up after the current run/timer
    this.timer = setTimeout(() => {
      this.timer = null;
      this.flush();
    }, Math.random() * JITTER_MAX_MS);
  }

  /** Runs one reload; events arriving meanwhile yield exactly one follow-up. */
  async flush() {
    if (this.stopped || this.running || !this.pendingScope) return;
    const scope = this.pendingScope;
    this.pendingScope = null;
    this.running = true;
    try {
      await this.onChange(scope, this.rev);
    } catch (err) {
      console.error('realtime: reload failed', err);
    } finally {
      this.running = false;
    }
    if (this.pendingScope && !this.stopped) {
      this.timer = setTimeout(() => {
        this.timer = null;
        this.flush();
      }, Math.random() * JITTER_MAX_MS);
    }
  }

  async fire(scope, rev) {
    try {
      await this.onChange(scope, rev);
    } catch (err) {
      console.error('realtime: handler failed', err);
    }
  }
}

function parse(e) {
  try {
    return JSON.parse(e.data);
  } catch (_) {
    return null;
  }
}
