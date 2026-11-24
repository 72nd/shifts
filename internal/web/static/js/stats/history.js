// History tab (prj/stats-plan.md, 6.3): SVG geometry for the sign-up curve and
// bucket labels. Presentation only; all statistics come from compute.js.

const W = 480;
const H = 200;
const PAD = { left: 34, right: 12, top: 16, bottom: 28 };

/** Alpine.data('statsHistory'): pure helpers, the model is passed in by the template. */
document.addEventListener('alpine:init', () => {
    Alpine.data('statsHistory', () => ({
        W, H, PAD,

        /** Returns SVG geometry for model.signups, or null when there is nothing to draw. */
        curve(signups) {
            if (!signups || !signups.points.length) return null;
            const pts = signups.points;
            const fc = signups.forecast;
            const required = signups.requiredTotal || 0;
            const last = pts[pts.length - 1];

            // Forecast end point (dashed extension).
            let fcEnd = null;
            if (fc && fc.date) {
                fcEnd = { at: fc.date, value: required };
            } else if (fc && fc.remainingAtStart != null && signups.planStart) {
                fcEnd = { at: signups.planStart, value: Math.max(0, required - fc.remainingAtStart) };
            }

            const t0 = pts[0].at.getTime();
            let t1 = last.at.getTime();
            if (signups.planStart) t1 = Math.max(t1, signups.planStart.getTime());
            if (fcEnd) t1 = Math.max(t1, fcEnd.at.getTime());
            if (t1 <= t0) t1 = t0 + 3600 * 1000;
            const yMax = Math.max(required, last.cumulative, 1);

            const x = (d) => PAD.left + ((d.getTime() - t0) / (t1 - t0)) * (W - PAD.left - PAD.right);
            const y = (v) => H - PAD.bottom - (v / yMax) * (H - PAD.top - PAD.bottom);
            const r = (n) => Math.round(n * 10) / 10;

            // Step line: horizontal to the new time, then vertical to the new value.
            let d = `M${r(x(pts[0].at))},${r(y(0))}`;
            let prev = 0;
            for (const p of pts) {
                d += `H${r(x(p.at))}V${r(y(p.cumulative))}`;
                prev = p.cumulative;
            }

            const lastX = x(last.at);
            const out = {
                path: d,
                requiredY: r(y(required)),
                left: PAD.left,
                right: W - PAD.right,
                baseY: H - PAD.bottom,
                topY: r(y(yMax)),
                yMax,
                required,
                lastX: r(lastX),
                lastY: r(y(prev)),
                planX: signups.planStart ? r(x(signups.planStart)) : null,
                forecastPath: fcEnd ? `M${r(lastX)},${r(y(prev))}L${r(x(fcEnd.at))},${r(y(fcEnd.value))}` : null,
                ticks: [],
                summary: { total: last.cumulative, required },
            };

            // Four evenly spaced time ticks (first one left-aligned, last one right-aligned).
            const n = 4;
            for (let i = 0; i < n; i++) {
                const date = new Date(t0 + ((t1 - t0) * i) / (n - 1));
                out.ticks.push({ x: r(x(date)), date, anchor: i === 0 ? 'start' : (i === n - 1 ? 'end' : 'middle') });
            }
            return out;
        },

        /** Range such as "0–2" or "20+"; the unit comes from stats.hours_unit. */
        bucketLabel(b) {
            return b.toHours == null ? `${b.fromHours}+` : `${b.fromHours}–${b.toHours}`;
        },

        /** Bar width in percent relative to the biggest bucket. */
        barWidth(b, buckets) {
            const max = Math.max(1, ...buckets.map(x => x.count));
            return b.count ? Math.max(2, Math.round((b.count / max) * 100)) : 0;
        },
    }));
});
