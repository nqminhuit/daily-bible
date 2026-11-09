# Cache implementation audit — 2026-11-09

## Scope and publication status

This is an **evaluation on temporary database copies**, not approval of the site's
production calendar or an archive-wide repair. No shipped database was changed;
no changes were pushed/deployed. The production calendar and November 15, 2026
local-calendar choice remain open. See [the policy contract](calendar-policy.md).

Backend baseline `resources/bible.db`:

```text
e4cbcfe55f603c4e7cdb7c36bf8a3075469d72500fe490fab122c330b36ad1c7
```

The audit retained all **2,191 original dates and 1,586 non-null references**.
Every original `gospel_ref` was compared with the audit copy: **zero changes**.
Six legacy seasonal keys have multiple reference values; this is a review queue,
not permission to choose their first/majority value.

## Offline evaluation

Imported the candidate Roman General definitions for 2021–2027: **2,556 dates**.
Importing them alone approved no readings. Then explicitly reviewed these eight
reusable mappings, each with scope/cycle/option notes and real source evidence:

| Evidence date | Reviewed entry | Gospel |
| --- | --- | --- |
| 2026-11-09 | Lateran dedication, invariant daytime proper | `Ga 2,13-22` |
| 2026-11-01 | All Saints, invariant daytime proper | `Mt 5,1-12a` |
| 2024-04-08 | Annunciation, including lawful transfer | `Lc 1,26-38` |
| 2026-11-03 | Ordinary 31 Tuesday, cycle II, default weekday | `Lc 14,15-24` |
| 2026-11-05 | Ordinary 31 Thursday, cycle II | `Lc 15,1-10` |
| 2026-11-06 | Ordinary 31 Friday, cycle II | `Lc 16,1-8` |
| 2026-11-07 | Ordinary 31 Saturday, cycle II, not optional Marian Mass | `Lc 16,9-15` |
| 2026-11-08 | Ordinary 32 Sunday, year A | `Mt 25,1-13` |

Two additional **date-only** reviews selected the standard weekday Gospel on the
November 10 and 11, 2026 memorials (`Lc 17,7-10`, `Lc 17,11-19`). These do not
approve universal memorial entries or other permitted formularies.

| Classification | Before reviews | After reviews |
| --- | ---: | ---: |
| `calendar_resolved` | 0 | 32 |
| `source_checked` (scoped date decisions) | 0 | 2 |
| `unreviewed_entry` | 2,542 | 2,508 |
| `unsupported_calendar_date` | 14 | 14 |

This demonstrates **34 supported selections** from eight reusable mappings plus
two date decisions: 31 historical dates and three future dates in 2027. The
unsupported days are the generic daytime Holy Thursday/Holy Saturday cases.
**It is not sufficient archive coverage for deployment.** Remaining mappings and
local rules require review after calendar scope is confirmed.

The separate unchanged source-only policy still has **14 accepted dates**, one
conflict, and 2,541 unverified dates in this extended copy. These are different
policies; evidence was not silently transferred between them.

- Calendar import, inventory, reviewed-cache reuse: **zero Gospel HTTP requests**.
- A normal, non-offline 2027-11-09 run: one cache hit, **zero HTTP**, no unresolved
  selection/text; citation remains the actual 2026-11-09 evidence URL.
- Repeated import/audit produced an identical report, no duplicated reviews, and
  unchanged counts: eight entry reviews, two date reviews, 15 retained observations.
- SQLite integrity and foreign-key checks passed. Consistent DELETE-mode copies
  were prepared for independent backend/frontend validation, not promotion.

## Independent source sample

A separate, explicit audit fetched **10 dated Vatican pages**, one for each
review above. It validated the actual title date and reference, then compared the
source Gospel paragraphs with the shared cached text.

- Reference agreement: **10/10**.
- Normalized text equality: **9/10**.
- All Saints: identical Gospel words/span, with only a missing terminal closing
  curly quotation mark in the cache. This punctuation difference was inspected;
  no global verse-text overwrite was made.

This is a bounded source audit, not a crawl of 2,556 dates. Structural verse
coverage alone cannot detect a preexisting fragment inside a legacy base-verse
row; passage comparisons remain necessary when certifying new mappings.

## Validation

Passed:

- Uncached Go tests and race suite, all Go builds (`-tags "fts5"`).
- Staticcheck and both repositories' actionlint checks.
- Four pinned calendar-provider tests, including precedence/transfers and versions.
- Thirteen frontend tests; lint, reading checker, production build.
- API, CLI, frontend lookup and production-mode pages for all **34 supported
  dates**, agreeing on reference, body, label, method, scope and actual evidence.
- The same four consumers reject three controls: November 2/15, 2026 (unreviewed
  candidate selections) and November 9, 2028 (no imported calendar definitions).
- Unit regressions for future reuse, wrong scopes/options, stale date exceptions,
  complete range/suffix coverage, retained history, rollback and cancellation.
- Explicit refresh disagreement/network-outage tests retain usable reviewed cache
  entries; request counts include bounded retry attempts.

Detailed temporary artifacts are under:

```text
/tmp/daily-bible-cache-execution.1OOFNU/
```

They include the baseline hash, `audit.db`, both calendar manifests,
`before.json`/`after.json`/`repeat.json`, inventory, source comparison HTML/JSON,
source-only report, Go test/race logs, warm-cache log, independent runtime copies,
and `smoke.py`/`smoke-results.json`. Temporary artifacts are not release assets.

## Remaining gates

Confirm the intended calendar and Mass options; review the remaining archive
mappings/local exceptions and any needed text repairs; generate a coverage/diff
report for that policy; approve compatible backend/frontend snapshots and rollback
procedure. Do not replace unavailable archive rows with unreviewed predictions or
publish the opt-in demonstration as a production calendar decision.
