# Test scenarios

Creates complete shift plans (types, shifts, sign-ups, people, fields) on a running instance via the API. Python 3 only, no packages.

```
./scenario.py                                  # list scenarios
./scenario.py summer-festival                  # next Saturday as start
./scenario.py summer-festival --start 2026-07-18 --host http://127.0.0.1:8080
./scenario.py summer-festival --dry-run        # send nothing
```

It prints the plan URL and the admin password.

## Scenarios

| Name | Content |
|------|---------|
| `summer-festival` | Two-day volunteer festival made for trying the coordination (switched on, day boundary 04:00, attendance by participants, default target 6 h). 7 types, 59 shifts, 154 sign-ups, one locked shift. Required select field "Experience" (New / Experienced / Pro; the option order is the level) is the experience field; 34 of 36 people with a record have a level, everyone else shows "Experience missing". **Bar** (1x Experienced; 2x on Sat 18:00 as per-shift override) and **Stage Tech** (popular, 1x Experienced) need experience. Load factors: Cleanup 2, Bar 1.5, Stage Tech 1.2. Bar shifts are overbooked (10 wishes for 4 places). Targets: Emma 600, Jonas 480, Hannah 240, Tariq 120 min. Absent: Mike/Tom/Yusuf leave Sat 18:00, Omar/Zoe arrive Sun 08:00, Elif/Ravi Sun 08:00-18:00, Lena/Rosa/Sven/Tilda/Uwe leave Sun 12:00. |

## Adding a scenario

Copy a file in `definitions/`. All days are offsets from the plan start (`0` = first day), `weekday` (0 = Monday) picks the default start. In the plan title and description `{{year}}`, `{{dates_de}}` and `{{dates_en}}` are replaced by the real dates. Person fields are referenced by their index in `fields`.

Optional keys: a shift may have `assigned` (names from its participants that get status "assigned") and the overrides `load_factor`, `popularity`, `exp_min_count`, `exp_min_level`. The choice field value of a person and `exp_min_level` are option indices (0 = lowest level). `coordination` holds the plan's coordination settings, the field with `"experience": true` becomes the experience field. People need no record: without one they have no field values.

Not part of a scenario: shift type passwords.

## Test cases in `summer-festival`

Times are for the default start (Sat/Sun), look at the coordination page, tab Distribute.

| Case | Where | What to see |
|------|-------|-------------|
| Bottleneck "needed elsewhere" | Sat 22:00 Bar vs. Sat 22:00 Stage Tech | Lena is assigned to the Bar, so its need is met. **Jonas** is the only experienced person who is free for Stage Tech, so in the Bar he carries the hint "needed in Stage Tech". |
| Auto-distribute, smaller pool | Sun 18:00 Bar vs. Sun 20:00 Stage Tech | Both need one experienced person, **Omar** wants both. Zoe could do the Bar only (she works Sun 22:00), so auto-distribute gives Omar to Stage Tech and Zoe to the Bar. Elif and Ravi are away at that time. |
| Plenty of experienced people | Sun 14:00 Bar and Stage Tech | Five experienced people free (Elif, Jonas, Omar, Ravi, Zoe), no hint. |
| Need already met | Sat 14:00 Stage Tech (Sven assigned), Sat 22:00 Bar (Lena assigned) | No experience warning. |
| Overlapping wishes | Jonas (Bar and Stage Tech Sat 22:00), Omar (Bar Sun 18:00, Stage Tech Sun 20:00) | Wishes that overlap in time. |
| Absence | people view: Mike, Tom, Yusuf, Omar, Zoe, Elif, Ravi, Lena, Rosa, Sven, Tilda, Uwe | Unavailable bars/columns. |
| Across the day boundary | Sat 22:00 Bar/Stage Tech (past midnight), Sun 02:00-05:00 Stage Tech (over 04:00) | Timeline across midnight and across the logical day change. |
| Missing field | people tab: Emma, Tariq | "Experience missing" (the field is required, no value stored). |
| Not leveled | people without a record, e.g. everyone not listed above | Level "no information". |
