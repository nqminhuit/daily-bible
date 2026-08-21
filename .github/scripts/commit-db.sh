#!/usr/bin/env bash
# Commits a refreshed database straight to the target branch and pushes it.
# No PR: a data refresh has nothing to review, and waiting on a human merge is
# what let gaps reach the site.
set -euo pipefail

FILE=${FILE:-resources/bible.db}
MESSAGE=${MESSAGE:?MESSAGE is required}
BRANCH=${BRANCH:-master}
ATTEMPTS=${ATTEMPTS:-3}

git config user.name 'github-actions[bot]'
git config user.email 'github-actions[bot]@users.noreply.github.com'

if git diff --quiet -- "$FILE"; then
  echo "$FILE is unchanged — nothing to commit."
  exit 0
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
cp "$FILE" "$tmp/keep"

# Nothing reviews these commits, so refuse to publish a database that lost
# readings — a partly-failed crawl or a truncated download would otherwise be
# pushed straight to production. Set ALLOW_SHRINK=1 for a deliberate removal.
if command -v sqlite3 > /dev/null && git cat-file -e "HEAD:$FILE" 2> /dev/null; then
  count_readings() {
    sqlite3 "$1" "SELECT COUNT(*) FROM daily_readings WHERE gospel_ref IS NOT NULL AND gospel_ref <> '';" 2> /dev/null || echo unknown
  }
  git show "HEAD:$FILE" > "$tmp/head.db"
  before=$(count_readings "$tmp/head.db")
  after=$(count_readings "$FILE")

  if [[ $before == unknown || $after == unknown ]]; then
    echo "Could not count readings (before=$before after=$after) — skipping the regression check."
  elif [[ $after -lt $before && ${ALLOW_SHRINK:-0} != 1 ]]; then
    echo "::error::Refusing to commit: readings dropped from $before to $after. Set ALLOW_SHRINK=1 if this is intended."
    exit 1
  else
    echo "Readings: $before -> $after"
  fi
fi

for attempt in $(seq 1 "$ATTEMPTS"); do
  git add -- "$FILE"
  git commit -qm "$MESSAGE"

  if git push --quiet origin "HEAD:$BRANCH"; then
    echo "Pushed $FILE to $BRANCH: $(git rev-parse --short HEAD)"
    exit 0
  fi

  echo "Push rejected (attempt $attempt/$ATTEMPTS) — $BRANCH moved. Replaying on top of it."
  # The file we just built is the authoritative one (freshly crawled or synced)
  # and a binary database has no meaningful merge, so on a race our copy wins:
  # rewind to the new tip and re-apply it.
  git fetch --quiet origin "$BRANCH"
  git reset --quiet --hard "origin/$BRANCH"
  cp "$tmp/keep" "$FILE"

  if git diff --quiet -- "$FILE"; then
    echo "$BRANCH already has this data — nothing to do."
    exit 0
  fi
done

echo "::error::Could not push $FILE to $BRANCH after $ATTEMPTS attempts."
exit 1
