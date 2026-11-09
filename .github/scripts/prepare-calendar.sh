#!/usr/bin/env bash
# Export definitions only. This neither crawls Gospels nor approves mappings.
set -euo pipefail
calendar=${READING_CALENDAR:-vatican-vi}
case "$calendar" in
  vatican-vi) exit 0 ;;
  roman-general|vietnam) ;;
  *) echo "Unsupported READING_CALENDAR: $calendar" >&2; exit 2 ;;
esac
IFS=',' read -r -a years <<< "${1:?comma-separated years required}"
output=${CALENDAR_MANIFEST:-build/calendar.json}
mkdir -p "$(dirname "$output")"
npm ci --prefix tools/calendar --ignore-scripts --no-audit --no-fund
node tools/calendar/export.cjs "$calendar" "${years[@]}" > "$output"
