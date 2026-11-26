---
paths:
  - "internal/web/templates/**"
  - "internal/web/static/css/**"
  - "internal/web/static/js/**"
  - "internal/web/static/i18n/**"
---

# UI rules

Participants open a link on a phone or laptop, often once and without any introduction. The UI must work without explanation: mobile first, calm, one main thing per page. We keep the daisyUI look; the brand colour (`#422ad5`) is the only distinctive element. No new typefaces, effects or decoration.

For a new page or a larger component run the `ui-plan` skill first and wait for approval. Small changes to an existing page need no plan. Read the neighbouring partials first so the change matches what exists.

## Stack

- daisyUI components (`btn`, `modal`, `badge`, `alert`, …) before custom CSS. Tailwind utilities before `static/css/timetable.css`. New CSS only if utilities cannot do it; mind selector specificity there.
- Colours only through daisyUI theme tokens (`base-100`, `primary`, `error`, …), never raw colour classes or hex values. Light and dark theme (`data-theme`) must both work.
- Alpine renders (templating in the HTML), JS models hold state and talk to the server. No hand-built DOM in JS.
- No npm, no new vendored asset without asking.

## Layout

- Mobile first (check 390 px), then widen with `sm:`/`md:`/`lg:`. Verify again at 1280 px.
- Never a horizontal scrollbar on the page: `min-w-0` on flex/grid children with text, `flex-wrap` on button groups. The timetable scrolls inside its own container only.
- Touch targets at least 44 px high. At most one primary action per view.
- No cards in cards. Do not give everything the same radius and shadow. Separate with spacing or a 1 px line.
- A change must not make a page taller or show less information per screen. Compare before and after.

## Modals

- Pattern: `<dialog class="modal">` as in `partials/shift-delete-modal.html`, errors through the `error-display` partial.
- Modals for short inputs and confirmations. A large form is a page section, not a modal.
- Destructive action: labelled error button inside a confirmation dialog.

## States

Every view and component considers: empty (say what belongs here and offer the action), loading (page stays usable), error (at the place of the problem, what happened and what to do), overlong (truncate with `…` and a tooltip; layouts survive 30 % longer text, German is longer than English), disabled (only when the reason is clear).

## Accessibility

- Semantic elements first (`button`, `dialog`, `label`, `nav`, headings in order), ARIA only where HTML cannot express it.
- Icon-only buttons need `aria-label`. Keyboard focus stays visible. Colour is never the only carrier of meaning (add icon or text).
- Motion only as the answer to an action, short, and disabled under `prefers-reduced-motion`.

## Texts and i18n

- Every visible text goes through `$t(...)` with keys in both `static/i18n/de.json` and `en.json` (flat). No hard-coded strings, including `aria-label` and `title`.
- German: gender-neutral, otherwise colon (`Teilnehmer:innen`), informal "Du". Fields that need both languages (titles) are filled in both.
- Name actions after what happens ("Delete shift", not "OK") and keep the name through the whole flow (button "Delete", toast "Deleted"). Sentence case. Errors say what happened and what to do, never apologise. Plain words from the user's view, not system terms.
- If help text is affected, update `internal/web/docs/help-{de,en}.md` and the feature list in `README.md` (see CLAUDE.md).

## Forbidden (they appear on their own when nobody decides)

Cards in cards · one radius and shadow everywhere · gradients · fade-ins and hover effects on surfaces · uppercase labels and eyebrows above self-explanatory content · middle-dot chains and `→` in button text as decoration · colour as the only signal · hard-coded texts · raw colours · explanatory sentences where a clear label would do · permanent "all fine" indicators.

## Building blocks

Reusable pieces are Go partials in `internal/web/templates/partials/`: `control-bar`, `filters`, `error-display`, `shift-row`, `timetable-view`, `type-card` and the `*-modal` partials. A pattern needed a second time becomes a partial. If a block is missing, propose it instead of copy-pasting markup.

## Check before reporting

`go vet ./...`, `task css` (when Tailwind classes changed), `task build`, run `./shifts`, then look at the change at 390 px and 1280 px, in light and dark theme, in German and English. Take a screenshot if the environment allows it, and ask yourself which element can go.
