---
name: ui-plan
description: Plan new Shifts UI before any code is written, for a new page, a larger component or a form with many fields: flow, function inventory, mobile and desktop sketch, two variants, states and texts in German and English. Use for every new page and larger component; small changes to an existing page need no plan.
---

# Plan new UI

Rules: `.claude/rules/ui.md`. Read the neighbouring templates and partials first (`internal/web/templates/`) so the plan matches what exists. Output one plan in the chat, one short block per step. Build only after the developer approves it.

1. **Flow.** What the participant or organiser wants to reach, in which steps, where they come from and where they go next. The flow comes before the elements.
2. **Function inventory.** Table: function, frequency, importance, place (control bar, inside the page, modal, settings). Frequent and important means visible and directly reachable; rare or set-once goes where it belongs, not into a hidden menu. Exactly one primary action per view.
3. **Sketch.** ASCII of the target state at 390 px (phone) and at 1280 px (desktop). For forms list the fields with their widths and which are required.
4. **Two variants.** For every non-obvious decision (layout, modal versus page section, grouping) name at least two approaches, compare them briefly, give one reasoned recommendation. Ask via `AskUserQuestion` only about real taste questions, recommended option first. Do not ask what `.claude/rules/ui.md` already decides.
5. **States and texts.** Empty, loading, error, overlong, disabled. Give the texts in German and English with their flat i18n keys (German: gender-neutral, informal "Du"). Action names stay the same through the whole flow.
6. **Reuse.** Which existing partials and daisyUI components are used, and which new partial is justified (needed a second time).

Then list explicitly what is **not** in scope, so neighbouring ideas stay out, and name the checks: `go vet ./...`, `make build`, and what to look at in the running app (390 px and 1280 px, light and dark theme, de and en).

**Density check:** compare with the existing page (height, visible information, values shown twice). Say whether it is denser, equal or less dense; if less dense, justify it or rework it.

Self-check before presenting: would a default daisyUI page have produced the same plan? Which element can go?
