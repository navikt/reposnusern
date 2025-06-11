#!/bin/bash

set -e

CONTAINER_NAME="reposnusern_benchmark"
IMAGE_NAME="reposnusnern"
CPU_LIMIT="$1"
TIMESTAMP=$(date -Iseconds | tr ':' '-')
CSV_FILE="benchmarks.csv"
RAW_STATS_FILE="raw_stats_$TIMESTAMP.json"

if [ -z "$CPU_LIMIT" ]; then
  echo "Bruk: $0 <cpu_limit>"
  exit 1
fi

echo "🚀 Kjører benchmark med --cpus=$CPU_LIMIT"

# CSV-header
if [ ! -f "$CSV_FILE" ]; then
  echo "timestamp,cpu_limit,peak_rss_mib,varighet_sec,num_gc" > "$CSV_FILE"
fi

# Start container i bakgrunnen
podman run -d \
  --cpus="$CPU_LIMIT" \
  --name "$CONTAINER_NAME" \
  -e ORG=org \
  -e GITHUB_TOKEN="" \
  -e POSTGRES_DSN="" \
  -e REPOSNUSERDEBUG=false \
  "$IMAGE_NAME"

# Vent til container er i "running" state
for i in {1..30}; do
  STATE=$(podman inspect -f '{{.State.Status}}' "$CONTAINER_NAME" 2>/dev/null || true)
  if [ "$STATE" == "running" ]; then
    break
  fi
  sleep 1
done

echo "📈 Starter JSON-måling til $RAW_STATS_FILE ..."

# Samle raw stats som JSON hvert sekund
while [[ "$(podman inspect -f '{{.State.Running}}' "$CONTAINER_NAME" 2>/dev/null)" == "true" ]]; do
  podman stats --no-stream --format json "$CONTAINER_NAME" >> "$RAW_STATS_FILE"
  sleep 1
done

# Hent logg før sletting
LOGFILE=$(mktemp)
podman logs "$CONTAINER_NAME" > "$LOGFILE" 2>/dev/null || true
podman rm "$CONTAINER_NAME" > /dev/null || true

# Parse logg
VARIGHET=$(grep '"varighet":' "$LOGFILE" | sed -E 's/.*"varighet":"([^"]+)".*/\1/')
NUM_GC=$(grep '"numGC":' "$LOGFILE" | tail -n1 | sed -E 's/.*"numGC":([0-9]+).*/\1/')
VARIGHET_SEC=0
if [ -n "$VARIGHET" ]; then
  VARIGHET_SEC=$(echo "$VARIGHET" | awk -F: '{ print ($1 * 3600) + ($2 * 60) + $3 }')
fi

# Bruk python for å hente peak mem fra JSON senere, men vi placeholder nå
echo "$(date -Iseconds),$CPU_LIMIT,0,$VARIGHET_SEC,$NUM_GC" >> "$CSV_FILE"
echo "✅ Logget (uten peak ennå): varighet=$VARIGHET_SEC sek, GC=$NUM_GC"
echo "📊 JSON-logg for parsing: $RAW_STATS_FILE"
