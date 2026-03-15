#!/usr/bin/env bash
# Kill running gmuxd instances and any processes holding dev ports.
# Run before starting dev to avoid EADDRINUSE / stale processes.
set -euo pipefail

ports=(8790 5173 8787)

# Kill gmuxd by name (catches instances not yet listening on a port)
if pids=$(pgrep -x gmuxd 2>/dev/null); then
  echo "killing gmuxd: $pids"
  kill $pids 2>/dev/null || true
fi

# Kill anything holding the dev ports
for port in "${ports[@]}"; do
  pid=$(lsof -ti "tcp:$port" 2>/dev/null || true)
  if [ -n "$pid" ]; then
    echo "port $port in use by pid $pid — killing"
    kill $pid 2>/dev/null || true
  fi
done

# Brief pause for ports to free up
sleep 0.3
