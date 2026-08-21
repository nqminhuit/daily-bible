#!/usr/bin/env bash
# Probes today plus LOOKAHEAD_DAYS days ahead for a usable gospel reading.
#
# Exits 0 only if every date in the window resolves. Used by
# .github/workflows/heal-readings.yml both before and after crawling, so the
# same definition of "usable" decides whether to crawl and whether to alert.
#
# Also writes build/probe-years.txt — the comma-separated set of years the
# window touches — so the crawler covers next year too when the window spans
# New Year's Eve.
set -uo pipefail

CLI=${CLI:-./build/daily-bible}
LOOKAHEAD_DAYS=${LOOKAHEAD_DAYS:-2}
TZ_NAME=Asia/Ho_Chi_Minh

if ! [[ $LOOKAHEAD_DAYS =~ ^[0-9]+$ ]]; then
  echo "LOOKAHEAD_DAYS must be a non-negative integer, got '$LOOKAHEAD_DAYS'" >&2
  exit 2
fi

# Match the CLI's own notion of "today" (internal/constants.Timezone).
today=$(TZ=$TZ_NAME date +%F)

mkdir -p build
missing=0
years=""

for ((i = 0; i <= LOOKAHEAD_DAYS; i++)); do
  date=$(date -u -d "$today +$i day" +%F)
  year=${date%%-*}
  case ",$years," in
    *",$year,"*) ;;
    *) years="${years:+$years,}$year" ;;
  esac

  if out=$("$CLI" "$date" 2>&1); then
    ref=$(printf '%s' "$out" | jq -r '.ref')
    echo "  OK   $date  $ref"
  else
    # Drop the Go log timestamp prefix ("2026/08/21 14:42:38 ") for readability.
    echo "  MISS $date  $(printf '%s' "$out" | sed -E 's#^[0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} ##')"
    missing=$((missing + 1))
  fi
done

printf '%s' "$years" > build/probe-years.txt

if [[ $missing -gt 0 ]]; then
  echo "$missing of $((LOOKAHEAD_DAYS + 1)) day(s) have no usable reading (years: $years)"
  exit 1
fi

echo "All $((LOOKAHEAD_DAYS + 1)) day(s) have a reading."
