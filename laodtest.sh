#!/usr/bin/env bash
./build.sh
set -euo pipefail

USERS=${1:-100}          # number of clients
DURATION=${2:-60}        # seconds each client runs
INTERVAL_MS=${3:-200}    # send interval in ms
RAMP_MS=${4:-20}         # ramp-up delay between clients
LOG_DIR=${5:-logs}

mkdir -p "$LOG_DIR"

echo "Starting load test:"
echo "  users      = $USERS"
echo "  duration   = ${DURATION}s"
echo "  interval   = ${INTERVAL_MS}ms"
echo "  ramp-up    = ${RAMP_MS}ms"
echo

for i in $(seq 1 "$USERS"); do
    from="u$i"
    to="u$(( (i % USERS) + 1 ))"

    timeout -k 5s "$((DURATION + 5))s" \
        ./build/testclient \
        -user "$from" \
        -to "$to" \
        -duration "${DURATION}s" \
        -interval "${INTERVAL_MS}ms" \
        > "$LOG_DIR/$from.log" 2>&1 &

    sleep "$(awk "BEGIN {print $RAMP_MS/1000}")"
done

wait
echo "Load test complete."