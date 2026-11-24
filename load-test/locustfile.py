"""
Locust load testing file for the Shift Planning Application

Expected load scenario:
- Up to 8 people acting as admins (creating plans, managing shifts, batch operations)
- Up to 100 people adding/removing themselves from shifts
- Additional 150 people viewing the frontend
- One batch operation on shifts

Default scenario (realtime plan, WP8): 100 SSE viewers (RealtimeViewer),
100 joiners/leavers (ParticipantUser), 8 admins (AdminUser) = 208 users.
Weights are configurable via environment variables (ratios, as Locust weights):
    LT_WEIGHT_REALTIME   (default 100)  RealtimeViewer, open SSE stream + reload
    LT_WEIGHT_PARTICIPANT(default 100)  ParticipantUser
    LT_WEIGHT_VIEWER     (default 0)    legacy polling ViewerUser (set e.g. 150)
Admins stay at a fixed count of 8. Run with --users 208 for the default mix.

How to read the results (Statistics tab / CSV):
- Join latency: row "/api/plans/[token]/shifts/[id]/participants [POST]",
  look at the 95%ile column.
- Reload latency: row "RELOAD types" (GET .../types after a change event,
  with If-None-Match and gzip; 200 and 304 both count as success).
- "EVENT->RELOAD done" is the delay from receiving a change event until the
  reload finished (includes the random 0-400 ms jitter, so expect ~200 ms + reload).
- "SSE connect" shows stream connects/reconnects; failures there mean dropped streams.
- Threshold: p95 of join and of RELOAD types should be <= 300 ms. Only if it is
  exceeded is the server-side response cache (plan section 4, WP8) needed.
Example:
    LT_WEIGHT_VIEWER=0 locust -f locustfile.py --host=http://localhost:8080 \
        --users 208 --spawn-rate 10 --headless -t 5m

Usage:
    locust -f locustfile.py --host=http://localhost:8080

Or with specific user distribution:
    locust -f locustfile.py --host=http://localhost:8080 \
        --users 258 --spawn-rate 10
        
For setup only (no load test):
    python locustfile.py --setup-only --host=http://localhost:8080
"""

import argparse
import json
import os
import random
import string
import sys
import time
import requests
from datetime import datetime, timedelta, timezone
import gevent
from locust import HttpUser, TaskSet, task, between, events
from locust.exception import StopUser


def random_string(length=8):
    """Generate a random string for names and comments."""
    return ''.join(random.choices(string.ascii_letters, k=length))


def generate_shift_data(type_id):
    """Generate random shift data."""
    # Use UTC time to ensure consistent timezone handling
    start_time = datetime.now(timezone.utc) + timedelta(days=random.randint(1, 30), hours=random.randint(6, 22))
    return {
        "type_id": type_id,
        "start_time": start_time.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "duration": random.choice([60, 120, 180, 240, 360, 480]),  # minutes
        "required": random.randint(1, 5),
        "comment_de": f"Kommentar {random_string(10)}",
        "comment_en": f"Comment {random_string(10)}"
    }


class SharedData:
    """Shared data across all users."""
    plan_token = None
    admin_password = None
    shift_types = []
    shifts = []
    participants = {}  # shift_id -> list of participant_ids
    setup_complete = False
    base_url = None


def run_setup(host):
    """Setup function that creates plan, types, and initial shifts.
    
    This can be called independently for testing setup logic without running the load test.
    
    Args:
        host: Base URL of the application (e.g., "http://localhost:8080")
        
    Returns:
        dict: Setup results with plan_token, admin_password, and counts of created items
    """
    print("=" * 80)
    print("SETUP PHASE - Creating test data...")
    print("=" * 80)
    
    # Clear any previous data from SharedData
    SharedData.plan_token = None
    SharedData.admin_password = None
    SharedData.shift_types = []
    SharedData.shifts = []
    SharedData.participants = {}
    SharedData.setup_complete = False
    SharedData.base_url = host
    
    # Create shift plan
    print("1. Creating shift plan...")
    # Use UTC time to ensure consistent timezone handling
    now = datetime.now(timezone.utc)
    start_date = now.replace(hour=0, minute=0, second=0, microsecond=0)
    end_date = (now + timedelta(days=60)).replace(hour=23, minute=59, second=59, microsecond=0)
    
    plan_data = {
        "title_en": "Load Test Shift Plan",
        "title_de": "Last Test Schichtplan",
        "description_de": "Test Schichtplan für Lasttests",
        "description_en": "Test shift plan for load testing",
        "start_date": start_date.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "end_date": end_date.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "is_multilingual": True,
        "collaborative_mode": True,
        "day_boundary_time": "00:00"
    }
    
    try:
        response = requests.post(f"{host}/api/plans", json=plan_data)
        if response.status_code != 200:
            print(f"   ✗ Failed to create plan:")
            print(f"      Status: {response.status_code}")
            print(f"      Response: {response.text}")
            print(f"      Sent data: {json.dumps(plan_data, indent=2)}")
            return
        data = response.json()
        SharedData.plan_token = data["token"]
        SharedData.admin_password = data["password"]
        print(f"   ✓ Plan created: {SharedData.plan_token}")
        print(f"   ✓ Admin password: {SharedData.admin_password}")
    except Exception as e:
        print(f"   ✗ Failed to create plan: {e}")
        if hasattr(e, 'response') and e.response:
            print(f"      Response: {e.response.text}")
        return
    
    # Create shift types
    print("2. Creating shift types...")
    shift_types_data = [
        {
            "title_en": "Morning Shift",
            "title_de": "Morgenschicht",
            "description_de": "Hilfe am Morgen",
            "description_en": "Help in the morning",
            "location_de": "Haupthalle",
            "location_en": "Main Hall"
        },
        {
            "title_en": "Afternoon Shift",
            "title_de": "Nachmittagsschicht",
            "description_de": "Hilfe am Nachmittag",
            "description_en": "Help in the afternoon",
            "location_de": "Cafeteria",
            "location_en": "Cafeteria"
        },
        {
            "title_en": "Night Shift",
            "title_de": "Nachtschicht",
            "description_de": "Hilfe in der Nacht",
            "description_en": "Help at night",
            "location_de": "Sicherheit",
            "location_en": "Security"
        },
        {
            "title_en": "Weekend Shift",
            "title_de": "Wochenendschicht",
            "description_de": "Hilfe am Wochenende",
            "description_en": "Help on weekends",
            "location_de": "Event-Bereich",
            "location_en": "Event Area"
        }
    ]
    
    for shift_type_data in shift_types_data:
        try:
            response = requests.post(
                f"{host}/api/plans/{SharedData.plan_token}/types",
                json=shift_type_data,
                headers={"X-Admin-Token": SharedData.admin_password}
            )
            if response.status_code == 200:
                SharedData.shift_types.append(response.json())
            else:
                print(f"   ✗ Failed to create shift type: {response.status_code}")
                print(f"      Response: {response.text}")
        except Exception as e:
            print(f"   ✗ Failed to create shift type: {e}")
    
    print(f"   ✓ Created {len(SharedData.shift_types)} shift types")
    
    # Create initial shifts
    print("3. Creating initial shifts...")
    for i in range(50):
        shift_type = random.choice(SharedData.shift_types)
        shift_data = generate_shift_data(shift_type["id"])
        
        try:
            response = requests.post(
                f"{host}/api/plans/{SharedData.plan_token}/shifts",
                json=shift_data,
                headers={"X-Admin-Token": SharedData.admin_password}
            )
            if response.status_code == 200:
                shift = response.json()
                SharedData.shifts.append(shift)
                SharedData.participants[shift["id"]] = []
            elif i < 3:  # Only print first 3 errors to avoid spam
                print(f"   ✗ Failed to create shift: {response.status_code}")
                print(f"      Response: {response.text}")
                print(f"      Data: {json.dumps(shift_data, indent=2)}")
        except Exception as e:
            if i < 3:
                print(f"   ✗ Failed to create shift: {e}")
    
    print(f"   ✓ Created {len(SharedData.shifts)} initial shifts")
    
    SharedData.setup_complete = True
    
    print("=" * 80)
    print("SETUP COMPLETE")
    print("=" * 80)
    print(f"Plan Token:      {SharedData.plan_token}")
    print(f"Admin Password:  {SharedData.admin_password}")
    print(f"Plan URL:        {host}/plans/{SharedData.plan_token}")
    print(f"Shift Types:     {len(SharedData.shift_types)}")
    print(f"Initial Shifts:  {len(SharedData.shifts)}")
    print("=" * 80)
    
    return {
        "plan_token": SharedData.plan_token,
        "admin_password": SharedData.admin_password,
        "plan_url": f"{host}/plans/{SharedData.plan_token}",
        "shift_types_count": len(SharedData.shift_types),
        "shifts_count": len(SharedData.shifts)
    }


# Initialize shared data on test start
@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    """Setup: Create plan, types, and initial shifts before any users spawn."""
    result = run_setup(environment.host)
    
    print("=" * 80)
    print("Starting load test...")
    print("Expected load:")
    print("  - 8 admins (managing shifts, batch operations)")
    print("  - 100 participants (joining/leaving shifts)")
    print("  - 100 realtime viewers (SSE + reload), 0 polling viewers by default")
    print("=" * 80)


class AdminBehavior(TaskSet):
    """Behavior for admin users managing the shift plan."""

    def on_start(self):
        """Wait for setup to complete."""
        max_wait = 60
        waited = 0
        while not SharedData.setup_complete and waited < max_wait:
            time.sleep(1)
            waited += 1
        
        if not SharedData.setup_complete:
            print("Admin: Setup not complete after 60s, stopping user")
            raise StopUser()

    @task(10)
    def view_plan_stats(self):
        """View plan statistics."""
        if not SharedData.plan_token:
            return

        self.client.get(
            f"/api/plans/{SharedData.plan_token}/stats",
            name="/api/plans/[token]/stats"
        )

    @task(5)
    def create_shift(self):
        """Create a new shift."""
        if not SharedData.plan_token or not SharedData.shift_types:
            return

        shift_type = random.choice(SharedData.shift_types)
        shift_data = generate_shift_data(shift_type["id"])

        with self.client.post(
            f"/api/plans/{SharedData.plan_token}/shifts",
            json=shift_data,
            headers={"X-Admin-Token": SharedData.admin_password},
            catch_response=True,
            name="/api/plans/[token]/shifts [POST]"
        ) as response:
            if response.status_code == 200:
                shift = response.json()
                SharedData.shifts.append(shift)
                SharedData.participants[shift["id"]] = []
                response.success()
            else:
                response.failure(f"Failed to create shift: {response.status_code} - {response.text}")

    @task(3)
    def update_shift(self):
        """Update an existing shift."""
        if not SharedData.plan_token or not SharedData.shifts or not SharedData.shift_types:
            return

        shift = random.choice(SharedData.shifts)
        
        # Ensure shift ID and type_id are valid
        if "id" not in shift or shift["id"] is None or "type_id" not in shift:
            return
        shift_id = int(shift["id"])
        
        # Use the shift's current type_id (database doesn't support changing type)
        update_data = {
            "type_id": shift["type_id"],
            "start_time": shift["start_time"],
            "duration": random.choice([60, 120, 180, 240]),
            "required": random.randint(1, 5),
            "comment_de": f"Aktualisiert {random_string(8)}",
            "comment_en": f"Updated {random_string(8)}"
        }

        with self.client.put(
            f"/api/plans/{SharedData.plan_token}/shifts/{shift_id}",
            json=update_data,
            headers={"X-Admin-Token": SharedData.admin_password},
            catch_response=True,
            name="/api/plans/[token]/shifts/[id] [PUT]"
        ) as response:
            if response.status_code == 204:
                response.success()
            else:
                response.failure(f"Failed to update shift: {response.status_code} - {response.text}")

    @task(2)
    def batch_create_shifts(self):
        """Batch create multiple shifts."""
        if not SharedData.plan_token or not SharedData.shift_types:
            return

        # Create 10-30 shifts in a batch
        num_shifts = random.randint(10, 30)
        shifts_data = []
        
        for _ in range(num_shifts):
            shift_type = random.choice(SharedData.shift_types)
            shifts_data.append(generate_shift_data(shift_type["id"]))

        batch_data = {"shifts": shifts_data}

        with self.client.post(
            f"/api/plans/{SharedData.plan_token}/shifts/batch",
            json=batch_data,
            headers={"X-Admin-Token": SharedData.admin_password},
            catch_response=True,
            name="/api/plans/[token]/shifts/batch [POST]"
        ) as response:
            if response.status_code == 200:
                result = response.json()
                for shift in result.get("created", []):
                    SharedData.shifts.append(shift)
                    SharedData.participants[shift["id"]] = []
                response.success()

    @task(2)
    def lock_shift(self):
        """Lock a shift."""
        if not SharedData.plan_token or not SharedData.shifts:
            return

        shift = random.choice(SharedData.shifts)
        if "id" not in shift or shift["id"] is None:
            return
        shift_id = int(shift["id"])
        
        with self.client.post(
            f"/api/plans/{SharedData.plan_token}/shifts/{shift_id}/lock",
            headers={"X-Admin-Token": SharedData.admin_password},
            catch_response=True,
            name="/api/plans/[token]/shifts/[id]/lock"
        ) as response:
            if response.status_code == 204:
                response.success()
            else:
                response.failure(f"Failed to lock shift: {response.status_code} - {response.text}")

    @task(2)
    def unlock_shift(self):
        """Unlock a shift."""
        if not SharedData.plan_token or not SharedData.shifts:
            return

        shift = random.choice(SharedData.shifts)
        if "id" not in shift or shift["id"] is None:
            return
        shift_id = int(shift["id"])
        
        with self.client.post(
            f"/api/plans/{SharedData.plan_token}/shifts/{shift_id}/unlock",
            headers={"X-Admin-Token": SharedData.admin_password},
            catch_response=True,
            name="/api/plans/[token]/shifts/[id]/unlock"
        ) as response:
            if response.status_code == 204:
                response.success()
            else:
                response.failure(f"Failed to unlock shift: {response.status_code} - {response.text}")

    @task(1)
    def delete_shift(self):
        """Delete a shift (rarely)."""
        if not SharedData.plan_token or not SharedData.shifts or len(SharedData.shifts) < 20:
            return

        shift = random.choice(SharedData.shifts)
        if "id" not in shift or shift["id"] is None:
            return
        shift_id = int(shift["id"])
        
        with self.client.delete(
            f"/api/plans/{SharedData.plan_token}/shifts/{shift_id}",
            headers={"X-Admin-Token": SharedData.admin_password},
            catch_response=True,
            name="/api/plans/[token]/shifts/[id] [DELETE]"
        ) as response:
            if response.status_code == 204:
                SharedData.shifts.remove(shift)
                if shift["id"] in SharedData.participants:
                    del SharedData.participants[shift["id"]]
                response.success()
            else:
                response.failure(f"Failed to delete shift: {response.status_code} - {response.text}")


class ParticipantBehavior(TaskSet):
    """Behavior for users who join/leave shifts."""

    def on_start(self):
        """Wait for setup to complete."""
        max_wait = 60
        waited = 0
        while not SharedData.setup_complete and waited < max_wait:
            time.sleep(1)
            waited += 1
        
        if not SharedData.setup_complete:
            print("Participant: Setup not complete after 60s, stopping user")
            raise StopUser()

        # Store user-specific data
        self.my_name = random_string(10)
        self.my_participations = []  # List of (shift_id, participant_id)

    @task(20)
    def add_to_shift(self):
        """Join a shift."""
        if not SharedData.plan_token or not SharedData.shifts:
            return

        # Don't join too many shifts
        if len(self.my_participations) >= 5:
            return

        shift = random.choice(SharedData.shifts)
        
        participant_data = {
            "name": self.my_name
        }

        with self.client.post(
            f"/api/plans/{SharedData.plan_token}/shifts/{shift['id']}/participants",
            json=participant_data,
            catch_response=True,
            name="/api/plans/[token]/shifts/[id]/participants [POST]"
        ) as response:
            if response.status_code == 200:
                participant = response.json()
                self.my_participations.append((shift["id"], participant["id"]))
                
                # Update shared data
                if shift["id"] in SharedData.participants:
                    SharedData.participants[shift["id"]].append(participant["id"])
                
                response.success()
            elif response.status_code == 400:
                # Invalid input, that's okay
                response.success()
            elif response.status_code == 403:
                # Shift is locked - participant cannot join without admin token
                # This is expected behavior, not an error
                response.success()
            elif response.status_code == 409:
                # Shift is full or duplicate name - expected during load testing
                response.success()
            else:
                response.failure(f"Failed to join shift: {response.status_code} - {response.text}")

    @task(10)
    def remove_from_shift(self):
        """Leave a shift."""
        if not SharedData.plan_token or not self.my_participations:
            return

        shift_id, participant_id = random.choice(self.my_participations)

        with self.client.delete(
            f"/api/plans/{SharedData.plan_token}/shifts/{shift_id}/participants/{participant_id}",
            catch_response=True,
            name="/api/plans/[token]/shifts/[id]/participants/[pid] [DELETE]"
        ) as response:
            if response.status_code == 204:
                self.my_participations.remove((shift_id, participant_id))
                
                # Update shared data
                if shift_id in SharedData.participants and participant_id in SharedData.participants[shift_id]:
                    SharedData.participants[shift_id].remove(participant_id)
                
                response.success()
            elif response.status_code == 403:
                # Shift is locked - participant cannot remove themselves
                # This is expected behavior, not an error
                response.success()
            else:
                response.failure(f"Failed to remove participant: {response.status_code} - {response.text}")

    @task(30)
    def view_shift_plan(self):
        """View the shift plan page."""
        if not SharedData.plan_token:
            return

        self.client.get(
            f"/plans/{SharedData.plan_token}",
            name="/plans/[token]"
        )


class ViewerBehavior(TaskSet):
    """Behavior for users who only view the frontend."""

    def on_start(self):
        """Wait for setup to complete."""
        max_wait = 60
        waited = 0
        while not SharedData.setup_complete and waited < max_wait:
            time.sleep(1)
            waited += 1
        
        if not SharedData.setup_complete:
            print("Viewer: Setup not complete after 60s, stopping user")
            raise StopUser()

    @task(50)
    def view_shift_plan(self):
        """View the shift plan page."""
        if not SharedData.plan_token:
            return

        self.client.get(
            f"/plans/{SharedData.plan_token}",
            name="/plans/[token]"
        )

    @task(10)
    def view_plan_api(self):
        """View plan details via API."""
        if not SharedData.plan_token:
            return

        self.client.get(
            f"/api/plans/{SharedData.plan_token}",
            name="/api/plans/[token]"
        )

    @task(5)
    def view_shift_types(self):
        """View shift types."""
        if not SharedData.plan_token:
            return

        self.client.get(
            f"/api/plans/{SharedData.plan_token}/types",
            name="/api/plans/[token]/types"
        )


class RealtimeViewerBehavior(TaskSet):
    """Holds an SSE stream open and reloads /types on every change event."""

    def on_start(self):
        waited = 0
        while not SharedData.setup_complete and waited < 60:
            time.sleep(1)
            waited += 1
        if not SharedData.setup_complete:
            raise StopUser()
        self.etag = None

    def reload_types(self, event_ts):
        """GET types like the real client: If-None-Match + gzip."""
        headers = {"Accept-Encoding": "gzip"}
        if self.etag:
            headers["If-None-Match"] = self.etag
        with self.client.get(
            f"/api/plans/{SharedData.plan_token}/types",
            headers=headers,
            catch_response=True,
            name="RELOAD types",
        ) as resp:
            if resp.status_code in (200, 304):
                if resp.status_code == 200 and resp.headers.get("ETag"):
                    self.etag = resp.headers["ETag"]
                resp.success()
            else:
                resp.failure(f"reload failed: {resp.status_code}")
        events.request.fire(
            request_type="EVENT",
            name="EVENT->RELOAD done",
            response_time=(time.time() - event_ts) * 1000,
            response_length=0,
            exception=None,
            context={},
        )

    def fire_connect(self, start, exc=None):
        events.request.fire(
            request_type="SSE",
            name="SSE connect",
            response_time=(time.time() - start) * 1000,
            response_length=0,
            exception=exc,
            context={},
        )

    @task
    def stream(self):
        """One connection lifetime; Locust calls this again to reconnect."""
        url = f"{SharedData.base_url}/api/plans/{SharedData.plan_token}/events"
        start = time.time()
        try:
            # Plain requests session: the Locust client would wait for the full body.
            # gevent monkey-patching (done by Locust) makes this cooperative.
            resp = requests.get(
                url,
                stream=True,
                headers={"Accept": "text/event-stream"},
                timeout=(5, 90),  # server pings every 25 s
            )
            if resp.status_code != 200:
                raise RuntimeError(f"status {resp.status_code}")
        except Exception as e:
            self.fire_connect(start, e)
            gevent.sleep(random.uniform(1, 3))  # reconnect backoff
            return
        self.fire_connect(start)

        event = None
        try:
            for raw in resp.iter_lines(decode_unicode=True):
                if raw is None:
                    continue
                line = raw.strip()
                if line.startswith(":"):  # ping comment
                    continue
                if line.startswith("event:"):
                    event = line[6:].strip()
                elif line.startswith("data:"):
                    if event == "change":
                        event_ts = time.time()
                        gevent.sleep(random.uniform(0, 0.4))  # jitter
                        self.reload_types(event_ts)
                    elif event == "deleted":
                        break
                elif line == "":
                    event = None
        except Exception as e:
            self.fire_connect(time.time(), e)  # stream dropped
        finally:
            resp.close()
        gevent.sleep(random.uniform(1, 3))  # reconnect backoff


class RealtimeViewer(HttpUser):
    """Viewer with an open SSE stream (no wait_time: the stream blocks)."""
    tasks = [RealtimeViewerBehavior]
    wait_time = between(0, 0)
    weight = int(os.environ.get("LT_WEIGHT_REALTIME", 100))


class AdminUser(HttpUser):
    """Admin user with management capabilities."""
    tasks = [AdminBehavior]
    wait_time = between(2, 8)
    fixed_count = 8  # Cap at exactly 8 admins


class ParticipantUser(HttpUser):
    """Regular user who joins/leaves shifts."""
    tasks = [ParticipantBehavior]
    wait_time = between(1, 5)
    weight = int(os.environ.get("LT_WEIGHT_PARTICIPANT", 100))


class ViewerUser(HttpUser):
    """User who only views the frontend."""
    tasks = [ViewerBehavior]
    wait_time = between(2, 5)
    weight = int(os.environ.get("LT_WEIGHT_VIEWER", 0))  # legacy polling viewers, off by default


# =============================================================================
# Command-line interface for setup-only mode
# =============================================================================

def main():
    """Main function for running setup without load test."""
    parser = argparse.ArgumentParser(
        description="Shift Planning Load Test Setup",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Run only the setup to create test data:
  python locustfile.py --setup-only --host=http://localhost:8080
  
  # Run the full load test (use locust command):
  locust -f locustfile.py --host=http://localhost:8080
        """
    )
    parser.add_argument(
        "--setup-only",
        action="store_true",
        help="Run only the setup phase (create plan, types, shifts) without starting the load test"
    )
    parser.add_argument(
        "--host",
        type=str,
        default="http://localhost:8080",
        help="Base URL of the application (default: http://localhost:8080)"
    )
    
    args = parser.parse_args()
    
    if args.setup_only:
        print("Running setup only (no load test will be started)")
        print(f"Target host: {args.host}")
        print()
        
        try:
            result = run_setup(args.host)
            print()
            print("✓ Setup completed successfully!")
            print()
            print("You can now:")
            print(f"  - View the plan at: {result['plan_url']}")
            print(f"  - Use admin password: {result['admin_password']}")
            print("  - Run load tests against this plan")
            sys.exit(0)
        except Exception as e:
            print()
            print(f"✗ Setup failed: {e}")
            import traceback
            traceback.print_exc()
            sys.exit(1)
    else:
        print("Use 'locust -f locustfile.py --host=http://localhost:8080' to run the load test")
        print("Or use --setup-only to run just the setup phase")
        sys.exit(0)


if __name__ == "__main__":
    main()
