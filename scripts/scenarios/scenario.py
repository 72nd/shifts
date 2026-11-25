#!/usr/bin/env python3
"""Create a test scenario (a complete shift plan) on a running instance via the API.

Usage:
    ./scenario.py                              list the available scenarios
    ./scenario.py summer-festival              create it, start on the next fitting weekday
    ./scenario.py summer-festival --start 2026-07-18 --host http://127.0.0.1:8080
    ./scenario.py summer-festival --dry-run    print what would be sent, change nothing

A scenario is a JSON file in definitions/. All days are offsets from the plan start
(0 = first day), so one file works for any start date. Needs only Python 3, no packages.
"""

import argparse
import datetime
import json
import sys
import urllib.error
import urllib.request
from pathlib import Path

DATA_DIR = Path(__file__).parent / "definitions"

WEEKDAYS_DE = ["Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag", "Sonntag"]
WEEKDAYS_EN = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"]
MONTHS_DE = ["Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August",
             "September", "Oktober", "November", "Dezember"]
MONTHS_EN = ["January", "February", "March", "April", "May", "June", "July", "August",
             "September", "October", "November", "December"]


class Api:
    """Minimal JSON client. In dry-run mode nothing is sent and ids are made up."""

    def __init__(self, host, dry_run):
        self.host = host.rstrip("/")
        self.dry_run = dry_run
        self.admin = ""
        self.requests = 0
        self._fake_id = 0

    def call(self, method, path, body=None, admin=False):
        self.requests += 1
        if self.dry_run:
            self._fake_id += 1
            return {"id": self._fake_id, "token": "DRYRUN", "password": "dry-run", "created": [
                {"id": self._fake_id + i} for i in range(len((body or {}).get("shifts", [])))]}
        headers = {"Content-Type": "application/json"}
        if admin:
            headers["X-Admin-Token"] = self.admin
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(self.host + path, data=data, headers=headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                raw = resp.read()
        except urllib.error.HTTPError as err:
            sys.exit(f"{method} {path} failed: HTTP {err.code} {err.read().decode(errors='replace').strip()}")
        except urllib.error.URLError as err:
            sys.exit(f"{method} {path} failed: {err.reason} (is the server running at {self.host}?)")
        return json.loads(raw) if raw.strip() else None


def load(name):
    path = DATA_DIR / f"{name}.json"
    if not path.is_file():
        sys.exit(f"Unknown scenario '{name}'. Available: {', '.join(p.stem for p in sorted(DATA_DIR.glob('*.json')))}")
    return json.loads(path.read_text(encoding="utf-8"))


def list_scenarios():
    for path in sorted(DATA_DIR.glob("*.json")):
        sc = json.loads(path.read_text(encoding="utf-8"))
        print(f"{path.stem:20} {sc.get('summary', '')}")


def default_start(weekday):
    """The next given weekday (0 = Monday) after today."""
    today = datetime.date.today()
    return today + datetime.timedelta(days=(weekday - today.weekday() - 1) % 7 + 1)


def date_range(start, end):
    """German and English date range for the {{dates_de}} / {{dates_en}} placeholders."""
    wd_de = WEEKDAYS_DE[start.weekday()] + ("" if start == end else "-" + WEEKDAYS_DE[end.weekday()])
    wd_en = WEEKDAYS_EN[start.weekday()] + ("" if start == end else "-" + WEEKDAYS_EN[end.weekday()])
    ys = "" if start.year == end.year else f" {start.year}"  # a differing start year is written out
    if start == end:
        de = f"{start.day}. {MONTHS_DE[start.month - 1]} {start.year}"
        en = f"{MONTHS_EN[start.month - 1]} {start.day}, {start.year}"
    elif start.month == end.month and start.year == end.year:
        de = f"{start.day}.-{end.day}. {MONTHS_DE[start.month - 1]} {end.year}"
        en = f"{MONTHS_EN[start.month - 1]} {start.day}-{end.day}, {end.year}"
    else:
        de = f"{start.day}. {MONTHS_DE[start.month - 1]}{ys} - {end.day}. {MONTHS_DE[end.month - 1]} {end.year}"
        en = f"{MONTHS_EN[start.month - 1]} {start.day}{',' + ys if ys else ''} - {MONTHS_EN[end.month - 1]} {end.day}, {end.year}"
    return f"{de} ({wd_de})", f"{en} ({wd_en})"


def build(api, sc, start):
    plan = sc["plan"]
    end = start + datetime.timedelta(days=plan["days"] - 1)
    dates_de, dates_en = date_range(start, end)

    def fill(text):
        return (text.replace("{{year}}", str(start.year))
                    .replace("{{dates_de}}", dates_de).replace("{{dates_en}}", dates_en))

    def stamp(day, time):
        return f"{start + datetime.timedelta(days=day)}T{time}:00"

    created = api.call("POST", "/api/plans", {
        "title_de": fill(plan["title_de"]), "title_en": fill(plan["title_en"]),
        "description_de": fill(plan["description_de"]), "description_en": fill(plan["description_en"]),
        "start_date": str(start), "end_date": str(end),
        "is_multilingual": plan["is_multilingual"], "collaborative_mode": plan["collaborative_mode"],
        "day_boundary_time": plan["day_boundary_time"], "time_zone": plan["time_zone"],
    })
    token, api.admin = created["token"], created["password"]
    base = f"/api/plans/{token}"

    # Required person fields would block the sign-ups below, so they are created
    # optional and switched to required at the end (like in a plan that grew over time).
    # A choice value is the index of its option in the scenario, the API wants the option id.
    field_ids, field_options, experience_id, levels = [], [], None, []
    for f in sc["fields"]:
        body = {k: f[k] for k in ("label_de", "label_en", "kind", "options", "editable_by")}
        created_field = api.call("POST", f"{base}/fields", {**body, "required": False}, admin=True)
        field_ids.append(created_field["id"])
        # (a dry run answers without options, so make some up)
        field_options.append(created_field.get("options") or [{**o, "id": f"opt{i}"} for i, o in enumerate(f["options"])])
        if f.get("experience"):  # its options (lowest level first) are the levels of the exp_min_level settings
            experience_id, levels = created_field["id"], field_options[-1]

    # Coordination settings of the plan: switch on, experience field, attendance and limits
    coord = sc["coordination"]
    api.call("PUT", base, {
        "title_de": fill(plan["title_de"]), "title_en": fill(plan["title_en"]),
        "description_de": fill(plan["description_de"]), "description_en": fill(plan["description_en"]),
        "start_date": str(start), "end_date": str(end),
        "is_multilingual": plan["is_multilingual"], "collaborative_mode": plan["collaborative_mode"],
        "day_boundary_time": plan["day_boundary_time"],
        "coordination_enabled": True, "experience_field_id": experience_id,
        **{k: coord[k] for k in ("attendance_mode", "default_target_minutes", "min_rest_minutes",
                                 "max_day_minutes", "show_declined")},
    }, admin=True)

    def level_id(index):
        return None if index is None else levels[index]["id"]

    shifts = []  # (shift id, scenario shift)
    for t in sc["types"]:
        type_body = {k: t[k] for k in ("title_de", "title_en", "description_de", "description_en",
                                       "location_de", "location_en", "load_factor", "popularity", "exp_min_count")}
        type_body["exp_min_level"] = level_id(t.get("exp_min_level")) or ""
        type_id = api.call("POST", f"{base}/types", type_body, admin=True)["id"]
        for i in range(0, len(t["shifts"]), 100):  # the batch endpoint takes at most 100
            chunk = t["shifts"][i:i + 100]
            result = api.call("POST", f"{base}/shifts/batch", {"shifts": [{
                "type_id": type_id, "start_time": stamp(s["day"], s["time"]), "duration": s["duration"],
                "required": s["required"], "comment_de": s["comment_de"], "comment_en": s["comment_en"],
            } for s in chunk]}, admin=True)
            if result.get("failed") or len(result["created"]) != len(chunk):
                sys.exit(f"Batch create of '{t['title_en']}' failed: {result.get('failed')}")
            shifts += [(c["id"], s) for c, s in zip(result["created"], chunk)]
            # The batch endpoint ignores per-shift overrides of the type's coordination settings,
            # so shifts with overrides are updated one by one.
            for c, s in zip(result["created"], chunk):
                overrides = {k: s[k] for k in ("load_factor", "popularity", "exp_min_count") if k in s}
                if "exp_min_level" in s:
                    overrides["exp_min_level"] = level_id(s["exp_min_level"])
                if overrides:
                    api.call("PUT", f"{base}/shifts/{c['id']}", {
                        "type_id": type_id, "start_time": stamp(s["day"], s["time"]), "duration": s["duration"],
                        "required": s["required"], "comment_de": s["comment_de"], "comment_en": s["comment_en"],
                        **overrides}, admin=True)

    for shift_id, s in shifts:
        for name in s["participants"]:
            api.call("POST", f"{base}/shifts/{shift_id}/participants", {"name": name})

    # People: attendance, target and field values (some people have no record, so their fields are missing)
    for p in sc["people"]:
        values = {}
        for k, v in p["values"].items():
            index = int(k)
            values[str(field_ids[index])] = field_options[index][v]["id"] if sc["fields"][index]["kind"] == "choice" else v
        api.call("PUT", f"{base}/people", {
            "name": p["name"],
            "available_from": stamp(**p["available_from"]) if p.get("available_from") else None,
            "available_until": stamp(**p["available_until"]) if p.get("available_until") else None,
            "target_minutes": p["target_minutes"], "prefers_long_days": p["prefers_long_days"], "values": values,
        }, admin=True)

    # Assignments made by the organisers (status "assigned"), e.g. to meet a shift's experience need
    ops = [{"shift_id": shift_id, "name": name, "status": "assigned"}
           for shift_id, s in shifts for name in s.get("assigned", [])]
    if ops:
        api.call("POST", f"{base}/coordination/apply", {"ops": ops, "lock": []}, admin=True)

    for field_id, options, f in zip(field_ids, field_options, sc["fields"]):
        if f["required"]:
            body = {k: f[k] for k in ("label_de", "label_en", "kind", "editable_by", "required")}
            api.call("PUT", f"{base}/fields/{field_id}", {**body, "options": options}, admin=True)  # options keep their ids

    for shift_id, s in shifts:
        if s["locked"]:
            api.call("POST", f"{base}/shifts/{shift_id}/lock", admin=True)

    return token, api.admin, len(shifts), sum(len(s["participants"]) for _, s in shifts)


def main():
    ap = argparse.ArgumentParser(description="Create a test scenario via the API.")
    ap.add_argument("scenario", nargs="?", help="scenario name (file in definitions/); omit to list them")
    ap.add_argument("--start", help="first day of the plan, YYYY-MM-DD (default: next fitting weekday)")
    ap.add_argument("--host", default="http://127.0.0.1:8080", help="base URL of the instance")
    ap.add_argument("--dry-run", action="store_true", help="send nothing, only count the requests")
    args = ap.parse_args()

    if not args.scenario:
        list_scenarios()
        return
    sc = load(args.scenario)
    try:
        start = datetime.date.fromisoformat(args.start) if args.start else default_start(sc["weekday"])
    except ValueError:
        sys.exit("--start must look like 2026-07-18")

    api = Api(args.host, args.dry_run)
    token, password, n_shifts, n_signups = build(api, sc, start)
    verb = "Would create" if args.dry_run else "Created"
    print(f"{verb} '{args.scenario}' from {start} ({api.requests} requests, {n_shifts} shifts, {n_signups} sign-ups)")
    if not args.dry_run:
        print(f"Plan:     {api.host}/plans/{token}")
        print(f"Password: {password}")


if __name__ == "__main__":
    main()
