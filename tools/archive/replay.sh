#!/bin/bash
# Offline replay of captured pages through the crawler worker and TSV converter. No network.
set -euo pipefail
R=$(cd "$(dirname "$0")/../.." && pwd); A=$R/build/vatican-archive; O=$A/replay/build
mkdir -p "$O"
cp "$R/tools/archive/archive_replay_test.go.txt" "$R/tools/crawler/archive_replay_temp_test.go"
trap 'rm -f "$R/tools/crawler/archive_replay_temp_test.go"' EXIT
cd "$R"
ARCHIVE_DIR=$A/pages ARCHIVE_OUT=$O go test -count=1 -run TestArchiveReplay ./tools/crawler 2>&1 | tail -3
go build -o "$A/replay/tsv" ./tools/tsv && (cd "$A/replay" && ./tsv)
wc -l "$O"/failed.txt "$O"/missing_verse_number.txt "$O"/gospels.tsv
