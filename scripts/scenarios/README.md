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
| `summer-festival` | Snapshot of a two-day volunteer festival: 6 shift types, 54 shifts, 138 sign-ups, one locked shift, a required number field "Experience". Built for trying the coordination: **Bar** has 4 shifts with 15 places and 38 sign-ups (3 people overlap with Food Service shifts, several have back-to-back bar shifts); **Cleanup** is mostly 3 to 5 of 5 filled. |

## Adding a scenario

Copy a file in `definitions/`. All days are offsets from the plan start (`0` = first day), `weekday` (0 = Monday) picks the default start. In the plan title and description `{{year}}`, `{{dates_de}}` and `{{dates_en}}` are replaced by the real dates. Person fields are referenced by their index in `fields`.

Not part of a scenario: shift type passwords, attendance and coordination settings of the plan (the API snapshot did not show them).
