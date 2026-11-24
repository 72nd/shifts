#!/bin/bash

# Load Testing Script for Shift Planning Application
# Usage: ./run_load_test.sh [mode] [host]
#
# Modes:
#   web     - Start with web UI (default)
#   quick   - Run a quick 2-minute headless test
#   stress  - Run a 10-minute stress test
#   report  - Run test and generate HTML report
#
# Examples:
#   ./run_load_test.sh
#   ./run_load_test.sh web http://localhost:8080
#   ./run_load_test.sh quick
#   ./run_load_test.sh stress https://test.example.com

set -e

# Default values
MODE="${1:-web}"
HOST="${2:-https://shifts.c3shoc.de}"
USERS=208
SPAWN_RATE=10

echo "═══════════════════════════════════════════════════════════"
echo "  Shift Planning Application - Load Testing"
echo "═══════════════════════════════════════════════════════════"
echo ""

# Check if locust is installed
if ! command -v uvx locust &> /dev/null; then
    echo "Error: Locust is not installed"
    echo ""
    echo "To install, choose one of the following methods:"
    echo ""
    echo "1. Using pip with requirements file (recommended):"
    echo "   pip install -r requirements-locust.txt"
    echo ""
    echo "2. Using pip directly:"
    echo "   pip install locust"
    echo ""
    echo "3. Using pip with user install (no sudo required):"
    echo "   pip install --user locust"
    echo ""
    echo "4. Using a virtual environment (isolated install):"
    echo "   python3 -m venv venv"
    echo "   source venv/bin/activate"
    echo "   pip install locust"
    echo ""
    exit 1
fi

# Check if locustfile exists
if [ ! -f "locustfile.py" ]; then
    echo "Error: locustfile.py not found"
    echo "Please run this script from the project root directory."
    exit 1
fi

echo "✓ Locust installed"
echo "✓ Test file found"
echo ""

# Display test configuration
echo "Test Configuration:"
echo "  Mode:        $MODE"
echo "  Host:        $HOST"
echo "  Users:       $USERS (8 admins, 100 participants, 150 viewers)"
echo "  Spawn Rate:  $SPAWN_RATE users/second"
echo ""

# Check if host is reachable
echo "Checking if host is reachable..."
if curl -s --head --request GET "$HOST" | grep "200\|301\|302\|404" > /dev/null; then
    echo "✓ Host is reachable"
else
    echo "⚠ Warning: Cannot reach host. Make sure the application is running."
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo ""
echo "═══════════════════════════════════════════════════════════"
echo ""

# Run based on mode
case "$MODE" in
    web)
        echo "Starting Locust with Web UI..."
        echo ""
        echo "Open your browser to: http://localhost:8089"
        echo ""
        echo "Recommended settings:"
        echo "  - Number of users: $USERS"
        echo "  - Spawn rate: $SPAWN_RATE"
        echo ""
        echo "Press Ctrl+C to stop"
        echo ""
        uvx locust -f locustfile.py --host="$HOST"
        ;;
    
    quick)
        echo "Running quick 2-minute test..."
        echo ""
        $LOCUST_CMD -f locustfile.py \
            --host="$HOST" \
            --users="$USERS" \
            --spawn-rate="$SPAWN_RATE" \
            --run-time=2m \
            --headless \
            --only-summary
        ;;
    
    stress)
        echo "Running 10-minute stress test..."
        echo ""
        $LOCUST_CMD -f locustfile.py \
            --host="$HOST" \
            --users="$USERS" \
            --spawn-rate="$SPAWN_RATE" \
            --run-time=10m \
            --headless
        ;;
    
    report)
        REPORT_DIR="load_test_reports"
        TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
        REPORT_FILE="$REPORT_DIR/report_$TIMESTAMP.html"
        
        mkdir -p "$REPORT_DIR"
        
        echo "Running test with HTML report generation..."
        echo ""
        echo "Report will be saved to: $REPORT_FILE"
        echo ""
        
        $LOCUST_CMD -f locustfile.py \
            --host="$HOST" \
            --users="$USERS" \
            --spawn-rate="$SPAWN_RATE" \
            --run-time=5m \
            --headless \
            --html="$REPORT_FILE"
        
        echo ""
        echo "═══════════════════════════════════════════════════════════"
        echo "✓ Test complete!"
        echo "  Report saved to: $REPORT_FILE"
        echo "═══════════════════════════════════════════════════════════"
        
        # Try to open the report
        if command -v open &> /dev/null; then
            open "$REPORT_FILE"
        elif command -v xdg-open &> /dev/null; then
            xdg-open "$REPORT_FILE"
        fi
        ;;
    
    *)
        echo "Error: Unknown mode '$MODE'"
        echo ""
        echo "Available modes:"
        echo "  web     - Start with web UI (default)"
        echo "  quick   - Run a quick 2-minute headless test"
        echo "  stress  - Run a 10-minute stress test"
        echo "  report  - Run test and generate HTML report"
        exit 1
        ;;
esac
