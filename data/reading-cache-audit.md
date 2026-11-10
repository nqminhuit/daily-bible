# Cache implementation audit — 2026-11-09/10

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

## First checkpoint: offline evaluation

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

## First checkpoint: validation

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

## Continuation: annual reuse, grouped queue, and provider rejection

A fresh consistent copy was evaluated with Gospel entry schema 2. Lectionary §69
establishes annual weekday **Gospel** reuse across I/II first-reading cycles;
Sunday/proper/memorial identities remain separate. The eight initial reusable
mappings were explicitly re-reviewed under the new rules version, retaining their
actual source evidence. Three additional Roman General Sunday-33 mappings
(A/B/C) were reviewed against the primary Lectionary tables. These are scoped
reference reviews, **not new Vietnamese-text comparisons or production approval**.

| Classification | After continuation |
| --- | ---: |
| `calendar_resolved` | 40 |
| `calendar_resolved_legacy_difference` | 4 |
| `source_checked` (date decisions) | 2 |
| `missing_text` (reviewed year-B Sunday 33) | 3 |
| `unreviewed_entry` (date rows) | 2,493 |
| `unsupported_calendar_date` | 14 |

This provides **46 available demonstration dates**, up from 34, without a Gospel
HTTP request for calendar import/reuse. Eleven reusable reviews support 47 date
selections, three of which cannot render because `Mc 13,24-32` has no local verse
rows; two independent date choices bring reference-supported selections to 49.
A normal missing-text run made **zero HTTP**, returned failure as expected, and
reported `unresolved=0 missing_text=1` rather than crawling a known reference again.

The read-only `entries` queue contains **1,052 distinct versioned entries**, of
which **1,041 require review**. Earlier unresolved counts were **date rows**, not
distinct entries. There are **27 entries with reference variants**. Legacy
candidate groups have 817 `structurally_available` and 48 `missing_text` statuses;
none was promoted by frequency or text coverage. Independent exact-date/source-only
choices are not grouped into reusable entries.

The Vietnam-provider evaluation uncovered an unsafe fixed date/rank: the pinned
plugin selects the Vietnamese Martyrs on November 13 and leaves November 24 as a
weekday. Automatic national export/import/reusable reviews and existing derived
caches with `romcal-` rules versions are now blocked. The rejected manifest was
not imported. Reviewed custom definitions need their own rules version; changing
the rejected version's name is not review. Independent accepted date choices are
not erased by this derived-provider restriction. See
[the research, primary references, and limitations](calendar-research.md).

For November 15, 2026, the temporary **Roman General** selection is `Mt 25,14-30`,
citing the real Lectionary PDF without a fabricated evidence date. The original
`Lc 9,23-26` is retained and can be a legitimate Vietnamese Martyrs alternative;
the intended local formulary/default remains undecided. The source-only policy
continues to report its one conflict. No evidence crosses namespaces silently.

Validation passed again:

- Uncached Go/race suites, builds, Staticcheck, and both repositories' workflow lint.
- Clean calendar install (zero vulnerabilities), five provider tests, and schema,
  version, annual reuse, rejected-provider and read-only queue regressions.
- Fifteen frontend tests, lint, reading checker and optimized production build.
- All **46 available dates plus four unavailable controls** across API, CLI,
  frontend lookup and production-mode pages: **200 consumer/date checks**.
  Missing-text, unreviewed and absent-definition controls remain unavailable;
  undated normative and older-year citations remain honest.
- Repeat export/import/audit is byte-identical and retains 11 entry reviews,
  two date reviews and 15 observations. Future Lateran warm reuse makes zero HTTP.
- Integrity/FK checks passed; all 2,191 original refs and the shipped database hash
  remain unchanged. Source-only still has 14 supported dates, one conflict and
  2,541 unverified dates over the extended range.

Three attempted dated Vatican research checks returned HTTP 403; none was
accepted and no access-control workaround was used. These were separate from the
zero-HTTP cache runs; the earlier 10-page word comparison was not repeated or
misrepresented as a fresh source check.

Continuation artifacts (temporary, not release assets):

```text
/tmp/daily-bible-calendar-continue.h2FdOQ/
/tmp/daily-bible-calendar-research-continue/
```

They retain before/after/repeat reports and grouped queues, the rejected provider
example, normative/public Church sources, test logs, missing-text/warm-cache logs,
independent DELETE-mode runtime copies and the 46-date parity smoke results.

## November 10 follow-up: diagnostics and genuine upcoming gap

The frontend availability check previously reinterpreted selection failures with
a weaker second query path. It could mislabel a new same-source conflict as
missing verse rows, or an unsupported provider as a missing entry approval.
Rendering and operator diagnostics now share one resolver; the whole monitoring
window reads one consistent in-memory snapshot instead of reopening the file for
each date. The public unavailable-response shape is unchanged.

Diagnostics distinguish unsupported calendar rules/options, missing definitions
or current reviews, stale explicit date choices, same-source conflicts, invalid
canonical references and actual missing text. Gospel normalization/equivalence
also rejects impossible chapter/endpoints, matching Go's canonical checks; it
does not suggest crawling nonexistent verses.

Validation:

- **20 frontend tests**, lint, optimized production build and workflow lint passed.
  Disposable subprocess regressions confirm exactly one snapshot file read for
  the full three-day window, including missing-date diagnostics, and verify
  distinct source-policy versus calendar-definition cliff remedies. Passing
  output explicitly limits its claim to the checked window, not the archive.
- Latest API/CLI/frontend lookup/rendered-page parity: **46 available dates plus
  five unavailable controls = 204 consumer/date checks**, with honest citations.
- On November 10 the **unchanged default three-day window correctly fails** for
  November 12's unreviewed Josaphat selection. This is a genuine demonstration-
  coverage gap, not a regression or permission to approve a reference automatically.
- An explicitly bounded November 10–11 positive control passes; the production
  lookahead default was not reduced. An unsupported `vigil` control fails with
  its correct configuration diagnostic, not a missing-text/crawl recommendation.
- Both shipped snapshots remain unchanged. No new source crawl, mapping approval,
  calendar activation or publication was performed in this diagnostics follow-up.

The continuation directory now also retains `checker-results.json`, the expected
three-day failure, positive/unsupported-option control logs and updated smoke
results. The earlier successful November 9–11 check is a historical checkpoint,
not a claim that later rolling windows have complete coverage.

## November 10 follow-up: target agreed, national rules not yet implemented

The owner confirmed the Vietnam national/daytime target, with no assumed local
overlay and a consistent authorized Gospel default; see
[the policy](calendar-policy.md). This does not promote the Roman General
checkpoint into national coverage or activate the blocked plugin.

Six HĐGMVN/Ủy ban Phụng tự sources are recorded in the partial
[national source dossier](vietnam-national-sources.json), with actual URLs,
request times and captured-response/text fingerprints. They establish seven
specific Mass cases and the distinction between the 2026 Ash Mass transfer and
the unchanged start of Lent/Hours. The conditional Ascension proclamation leaves
territorial/default applicability unresolved. Five offline specification guards
join the five existing provider tests; they are not a working national adapter.
See [the research findings](calendar-research.md#confirmed-target-and-primary-national-guidance--2026-11-10).

Validation for this source/policy follow-up: clean pinned calendar install,
**10 offline calendar tests**, `go test -tags "fts5" -count=1 ./...`, **20 frontend
tests**, and both worktrees' whitespace checks passed. All six recorded HTML/text
fingerprint pairs match the captured responses.

No calendar definitions, Gospel reviews, observations, verse rows or shipped
snapshots were changed by this research. Both shipped database SHA-256 hashes
remain `e4cbcfe55f603c4e7cdb7c36bf8a3075469d72500fe490fab122c330b36ad1c7`.
The 46-date demonstration and all unresolved counts above remain **Roman General
evaluation results**, not Vietnam production coverage. Nothing was pushed or deployed.

## Remaining gates

Review and implement authoritative national rules, historical applicability,
lectionary editions and allowed/default formularies for the confirmed target;
review the remaining archive mappings/local exceptions and any needed text
repairs; generate a national-policy coverage/diff report; approve compatible
backend/frontend snapshots and rollback procedure. Do not replace unavailable
archive rows with unreviewed predictions or publish the opt-in demonstration as
a supported national calendar.
