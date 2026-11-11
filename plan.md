# Plan: Correct Daily Gospels with Safe Cache Reuse

## Current plan — 2026-11-11: one complete lectionary table for every future date

**This section is the active plan. Everything below it is history of how we got here.**

Decided by the owner on 2026-11-11: the site should work for all future dates without yearly review, and **Tết Masses are out of scope**. Started the same day; see Progress below.

### Why this works

The lectionary is a finite table of roughly 900 entries: Sundays on a three-year cycle, weekdays on a yearly cycle, saints' days on fixed dates. Which entry falls on which date is calculation, which the pinned calendar provider does through 2199. Fill the table once and no date needs individual review again. The 2026 pilot filled it one date at a time; this plan fills all of it.

Vietnam needs only four fixed rules, none of them yearly: Epiphany, Ascension and Corpus Christi on Sunday, and the Vietnamese Martyrs solemnity on November 24. Earlier owner decisions still apply: General Roman baseline, ordinary day over optional celebrations, first-listed full option, daytime Mass, no diocesan overlay.

### Steps

1. **Crawl the Vatican News archive once.** Every dated page from 2021-01-01 to 2026-12-31, about 2,200 pages at one request per second (roughly 40 minutes). Reuse the 97 pages already captured. Capture to disk with a request log and fingerprints; no bypassing of failures; record them and move on. The owner approved this crawl.
2. **Rebuild verse text from the crawl.** Replay every captured page offline through the fixed importer (suffixed parts, chapter changes, continuation paragraphs) and the TSV converter. Diff against the shipped store. This replaces pattern scanning: it finds every fused, mis-chaptered or truncated row, including the five known ones (`Ga 15,1-3`, `Mt 10,1`, `Mt 18,1`, `Mc 8,38`, `Mt 11,17`). Wording-only differences are listed, not applied silently.
3. **Make entry identities year-independent.** Today saints' days carry their year's cycles and weekday context, so nothing reuses. Target identities: weekday by season/week/day; Sunday plus A/B/C; fixed-date celebration by its name, with a cycle only where the Gospel really varies. This is a Gospel-entry schema change (schema 4): regenerate, never relabel old approvals.
4. **Fill the table from six years of evidence.** Map each crawled date to its entry with the provider, group the page Gospels by entry, and accept an entry only when every year that has it agrees. List disagreements for a hand check. Memorials need care: Vatican News sometimes follows an optional memorial; the owner's policy keeps the ordinary day.
5. **Check against independent evidence.** The 328 reviewed 2026 dates in `data/vietnam-baseline-reviews.json` (checked against the Hà Nội and Vĩnh Long calendars) are the test set. Any table entry that contradicts them is investigated, not overridden.
6. **Generate the calendar far ahead** (through 2050) with the Sunday transfers and the November 24 rule, lifting the 2026-only guard. Remove the Tết and Ash-transfer definitions and the now-unused formulary identities.
7. **Ship.** Build matching backend and frontend snapshots from the table, calendar and rebuilt verse store; diff against the current shipped database; keep rollback snapshots; switch `READING_CALENDAR` in both repositories; update the frontend docs. Push and deploy only on the owner's explicit go-ahead.

### Progress — 2026-11-11 (steps 1, 2 and 4 done on disk; nothing shipped)

- **Step 1 done.** The archive starts on 2021-03-11, not 2021-01-01, and the sitemap ends on 2026-12-15. All 2,106 listed pages were captured (97 reused); 2026-12-16..30 returned 404. Every page's title date equals its URL date. Pages: `build/vatican-archive/pages/` (ignored). Fingerprints: `data/vatican-archive-captures.json`.
- **Importer fixed again** for 2021 markup: the Gospel section is no longer taken from the last section on the page, bare `<span>14</span>` verse markers are read, and "Kết thúc Tin Mừng" headers are accepted. Replay: 2,079 of 2,106 pages parse; 27 fail (12 Passion narratives, 3 pages with no Gospel section, the rest have no printed reference or a header typo) and 20 have plain-text verse numbers and give no verse rows.
- **Step 2 done (comparison only).** The rebuild has 2,963 verse rows against 2,917 shipped, and no shipped row is absent from it. 7 shipped rows hold another verse's text or are cut short (`Ga 15,1-3`, `Mt 10,1`, `Mt 18,1`, `Mt 11,17`, `Lc 7,32`), 10 carry a fused part or a stray digit (including `Mc 8,38`), 46 rows are missing from the store, and 61 hold a wording printed by a minority of pages. Nothing was applied. Details: `data/verse-rebuild-diff.json`.
- **Step 4 done as a candidate table:** `data/lectionary-table.json`, 580 entries (527 with a Gospel, 53 memorials that read the weekday). 44 pages are excluded as evidence, each with a reason, in `data/lectionary-exceptions.json` (Tết and year-end Masses, the Martyrs external solemnity, optional memorials, typos).
- **Step 5 done.** Against the 328 reviewed 2026 dates: 322 agree, none uncovered, 6 differ. Three are the Tết days and one is Mid-Autumn (out of scope by decision); two are short forms where the table keeps the full form (Presentation `Lc 2,22-40`, Nativity of Mary `Mt 1,1-16.18-23`).
- **Coverage through 2050** (8,766 days): everything resolves except Palm Sunday, Holy Thursday, Good Friday and Holy Saturday every year, and 11 Sunday entries (44 dates, the first on 2028-06-25) that the archive never printed. They are listed in the table file under `uncovered_2027_2050`.
- Four weekday entries depend on the Sunday cycle (Lent V Monday, Easter IV Monday, Ordinary XVIII Monday and Tuesday); the last two apply only when Sunday XVIII is actually celebrated.
- Scripts: `tools/archive/` (`crawl.py`, `replay.sh`, `analyze.py`, `versediff.py`).

**Owner decisions, 2026-11-11 (the owner took the recommendations):**

1. Presentation and Nativity of Mary keep the full form (`Lc 2,22-40`, `Mt 1,1-16.18-23`).
2. Mary, Mother of the Church keeps the reviewed `Ga 19,25-34`.
3. The 27 single-page entries ship, marked single-source.
4. Holy Week stays blank in this release (Palm Sunday, Holy Thursday, Good Friday, Holy Saturday). Cross-chapter Passion references are follow-up work, wanted before Palm Sunday 2027-03-21.
5. The 11 uncovered Sundays stay uncovered for now; the first falls on 2028-06-25. Fill them later from another source.
6. `fix/date-verified-mass-readings` may be pushed; its base is `master`. Code tasks go on child branches with draft PRs into it. Deploying still needs a separate go-ahead.

### Progress — 2026-11-11, later (steps 3 and 6 done; release candidate built; not shipped)

- **Step 3 done.** Gospel entry schema is 4: each calendar day carries an explicit `gospel_entry`, checked against its observance and weekday. Tết and Ash-transfer formularies are removed. Merged into `fix/date-verified-mass-readings` from `feat/year-independent-gospel-entries` (no PR: `gh` is signed in to an account that cannot open PRs on this repository).
- **Step 6 done.** `node tools/calendar/vietnam.cjs 2021-2050` generates 10,957 days, rules version `vietnam-general-00ea383dd5315370a43e`. The version does not change when more years are generated or when an uncovered Sunday is filled.
- **Release candidate:** `build/release-2026-11-11/bible.db` (ignored; SHA-256 starts `7e1b3d6c`). Built from the shipped database: verse repair (46 rows inserted, 17 replaced), calendar 2021-2050 imported, 527 entries applied as explicit reviews by `build.py`. Legacy date/reference pairs are unchanged; two legacy estimate keys changed (2026-11-01, 2026-11-09).
- **Result over 2021-2050:** 10,789 dates render, 48 have no reviewed entry (the 11 uncovered Sundays), 120 are the four Holy Week days. CLI and frontend agree on reference, text, entry, version and evidence for all 10,957 dates.
- **Changed after the owner's decision 2:** Mary, Mother of the Church uses `Ga 19,25-27`. `Ga 19,25-34` has no text for verses 28-30 (they exist only in the Passion narrative), so the day rendered blank. Switch back when Passion text is supported.
- **Near-term gap: Sunday 2026-12-20** (Advent IV, Year B) has no entry. Its only archive page (2023-12-24) printed the Christmas night Mass. Add the entry when the dated Vatican News page for 2026-12-20 is published, then apply one review; the rules version stays the same.
- Frontend (`/tmp/gospel-site-mass-fix.afwvYH`, local commit only): definition proof is required for every Vietnam rules version; 24 tests and lint pass.

**Left for step 7 (needs the owner's go-ahead):** copy the release candidate over both `resources/bible.db` files with rollback copies, switch `READING_CALENDAR` to `vietnam` in both repositories and their workflows, update the frontend docs, push the frontend branch, deploy, then check the live site. After activation the heal job may fetch dated Vatican pages as candidates for unreviewed entries; candidates are not approvals.

### Step 7 — 2026-11-11 (owner said go)

- Backend: `resources/bible.db` replaced by the release candidate (SHA-256 `f3c3bd62…`, journal mode DELETE); the previous file is in git history at `c8a74f3` and in `build/rollback-2026-11-11/`. The default calendar is `vietnam` in `internal/reading`, `prepare-calendar.sh` and the three workflows. The probe passes for today plus two days; over 60 days only 2026-12-20 misses. The yearly and heal workflows were simulated offline on a copy with zero HTTP requests.
- Frontend: same database (identical SHA-256), default `vietnam` in `lib/reading.js`, the checker and its workflow; README updated. 24 tests, lint, production build and a seven-page check pass.
- **Shipped 2026-11-11:** both `master` branches were fast-forwarded from `fix/date-verified-mass-readings` (backend first, because the frontend's sync job copies the backend's database). Backend CI passes on `master`. The live site shows `Lc 17,11-19` today from the Vietnam calendar, `Lc 9,23-26` on 2026-11-24, `Lc 2,16-21` on 2027-01-01, a 2035 date, and the no-reading message on 2026-12-20 and Good Friday 2027.
- **Still to do:** the 2026-12-20 entry (before that Sunday); Passion readings before 2027-03-21; the other 10 uncovered Sundays (first 2028-06-25); celebration labels; the database grew from 1.4 MB to 12.7 MB, mostly per-day definition text, and could be slimmed.
- To roll back: restore `resources/bible.db` from `c8a74f3` and set `READING_CALENDAR=vatican-vi` (repository variable or environment).

### Known limits to state plainly at release

- Tết days show the ordinary day, or Ash Wednesday when they clash.
- Where the Church allows a choice the site shows one fixed option.
- A newly added saint's day needs a one-line table update.
- The calendar provider is a beta. It matched the diocesan calendars on the reviewed 2026 dates; keep a yearly spot-check against a published calendar.
- Vatican News is the single text source. Entries that occur only once in 2021-2026, or never (rare week/weekday combinations), will be listed as uncovered rather than guessed.

### Urgent regardless of this plan

The shipped date table ends on **2026-12-31** (2,191 rows, 2021-2026). The frontend monitor warns 45 days before the end, around 2026-11-16, and the site has no reading for 2027-01-01 until either this plan ships or the old table is extended as a stopgap.

## Goal

Match the Gospel to the intended Mass calendar without crawling every date or
rediscovering the same passage every year.

**The problem is reuse through an incomplete calendar key, not reuse itself.**
Keep three responsibilities separate:

1. **Gospel text:** reuse the existing `verses` table by book/chapter/verse/suffix.
2. **Lectionary entry → Gospel reference:** reuse a trustworthy cached mapping
   when the celebration, reading cycle, and Mass option really are equivalent.
3. **Date → lectionary entry:** resolve precedence, transfers, and local rules
   before selecting the cached reference.

A dated source page is useful evidence and an audit tool. It must not be a
mandatory network dependency for every date.

## Starting State (before implementation)

- The original crawler copied references between dates sharing an estimated
  `lectionary_key`. Proper celebrations could therefore receive weekday readings.
- Local, unpublished changes now require exact-date provenance and stop that
  propagation. They also suppress estimated celebration labels and require
  explicit review of reference disagreements.
- Those safeguards are useful, but the per-date-only lookup contract is too
  restrictive: it cannot yet serve a date resolved through a trustworthy reusable
  lectionary entry.
- The prepared snapshots contain 14 usable source-verified dates and six reviewed
  corrections. Most historical rows remain unverified, not necessarily wrong.
- The November 15, 2026 discrepancy remains unresolved: stored `Lc 9,23-26`
  versus Vatican News `Mt 25,14-30`. Do not assume either is the correct selection
  for the intended local calendar until that calendar is established.
- The separate `gospel-site` repository reads its own SQLite snapshot directly;
  backend changes alone do not change its lookup behavior or labels.

**Deployment gate:** refine the current strict patch and review coverage before
publishing. Do not make nearly the entire archive unavailable merely because it
lacks individual dated-page records if trustworthy calendar-based reuse can
establish those readings.

## Execution Status — 2026-11-09/10

The cache implementation and Roman General temporary-copy validation are
complete. The owner **confirmed the Vietnam national/daytime target on
2026-11-10**, with documented authorized defaults and no assumed local overlay.
**Authoritative national definitions, remaining archive reviews, and rollout
are not complete.**

- Added additive reviewed-entry/date-decision/observation schemas and scoped,
  versioned lookups. Existing `verses`/legacy refs and review history are retained.
- Added a pinned, maintenance-only Roman General evaluation adapter with
  precedence/transfer fixtures. Continued research exposed a Vietnam-plugin
  date/rank discrepancy: automatic national export/import, reusable reviews and
  derived lookups under `romcal-` versions are now blocked. The `vietnam` namespace
  requires reviewed custom definitions; no guessed national overlay replaces it.
- API, CLI and companion frontend now support both evidence methods and genuine
  source dates; cache-first maintenance, offline annual extension, and separate
  source-audit automation are implemented.
- Audited 2,556 dates (2021–2027) on consistent copies, preserving all 2,191
  baseline dates and Gospel references. The first checkpoint supported 34 dates;
  the continuation supports **46** from 11 reviewed reusable mappings and two
  date-only choices, using zero Gospel HTTP for import/reuse. **2,493 date rows
  remain unreviewed (1,041 distinct entries); three approved selections lack text,
  and 14 generic daytime dates are unsupported. This is not release-ready coverage.**
- Corrected Gospel-only weekday identities to annual reuse across I/II first-reading
  cycles, supported by Lectionary §69; Sunday A/B/C and proper identities remain
  separate. Schema-2 manifests/version invalidation require explicit re-review.
- Added deterministic read-only entry review queues, with scoped/versioned grouping,
  normalized legacy candidates, structural text statuses and genuine observations.
  There are 27 grouped entries with reference variants, not automatic corrections.
- A separate 10-page source sample confirmed all references and Gospel words;
  one cached passage differs only by its terminal closing quotation mark.
- Uncached/race Go suites, builds, Staticcheck, workflow lint, calendar/frontend
  tests and production build passed. That full validation had five provider tests,
  20 frontend tests, and parity across all **46** available dates plus five
  unavailable controls in API, CLI, frontend lookup and production-mode pages.
- Frontend diagnostics now use the rendering resolver and one snapshot per
  monitoring window, distinguishing evidence conflicts/unsupported rules from
  missing text. Cliff warnings now request supported definitions rather than
  legacy estimates; a pass certifies its window, not full archive coverage.
  The November 10 default three-day check correctly fails for unreviewed November
  12; it was not made green by changing defaults or approvals.
- Warm future-year reuse makes zero HTTP requests. Missing text is reported and
  remains an operator repair through the existing verse pipeline; no automatic
  missing-text ingestion or blanket legacy approval was introduced.
- Runtime default remains source-only `vatican-vi`; target approval does not
  activate a profile. November 15's national celebration/default formulary remains
  unresolved. Its Roman General temporal reference is independently established
  without replacing the legacy Martyrs candidate or approving the national
  formulary. Neither shipped database was modified, and nothing was pushed/deployed.
- Added a six-source national dossier and offline specification tests. Official
  2026 guidance assigns February 18/19 to Tết Masses and February 20 to the Ash
  Mass, while Lent and Ash Wednesday Hours remain February 18. The national
  proclamation mentions both Ascension schedules without settling their default
  applicability. These facts are not a complete calendar or Gospel approvals.
  The offline calendar test command now includes five provider tests and five
  source-dossier/specification guards.

Details: [`data/calendar-policy.md`](data/calendar-policy.md) and
[`data/reading-cache-audit.md`](data/reading-cache-audit.md), and
[`data/calendar-research.md`](data/calendar-research.md). Unchecked items below
are substantive approval/archive-release work, not permission to bypass evidence.

## Vietnam pilot checkpoint — 2026-11-10

Implemented an explicit source-backed national subset, not a full calendar:

- A 365-row 2026 manifest supports **nine reviewed date-to-Mass definitions** and
  marks **356 dates unsupported**, with no Roman General/legacy fallback.
- Schema **3** and additive migration **005** preserve the underlying observance
  and seasons separately from an explicitly assigned Mass, plus precise per-day
  directive evidence and unsupported reasons. Old manifests must be regenerated;
  old approvals are not relabeled. The Hours are not implemented or shifted.
- Two explicit temporary-copy Gospel reviews (Ash on February 20 and All Saints
  November 1) yield **two renderable dates, seven unreviewed entries, 356 unsupported
  dates**. These are national pilot counts, not the earlier Roman General queue.
- Warm reuse and assigned/unresolved gap controls make zero HTTP. Wrong-scope
  Vatican crawling is suppressed for unreviewed assigned national formularies.
- Full Go/race/build/static checks, 14 calendar tests, 24 frontend tests/lint/build,
  and **1,110 consumer/date checks** passed. Genuine upcoming monitoring gaps still
  fail. All original refs/verse rows and both shipped snapshots are unchanged.
- November 15, full national rules/lectionary editions, historical applicability,
  national archive recovery and coordinated release remain incomplete. Nothing
  was pushed or deployed.

Details and reproducible commands: [`data/vietnam-pilot-audit.md`](data/vietnam-pilot-audit.md).

## Vietnam content/source continuation — 2026-11-10

- Reviewed six of the seven pending pilot mappings on a consistent temporary
  copy. Eight dates now render; Mồng Ba Tết remains unreviewed because its two
  published full options do not establish the approved lectionary's default order.
- Located a downloadable official Vĩnh Long 2025–2026 Ordo and a publisher's 2017
  three-volume lectionary catalogue. Kept their actual scope/status: the former
  is diocesan, not an approved national baseline; the latter is not approval
  frontmatter or an option-order contract. Full national rules remain incomplete.
- Preserved Corpus Christi's printed `Ga 6,51-59` discrepancy while reviewing
  canonical `Ga 6,51-58` from the corroborating Ordo, page tag and actual body.
  Historical Christ-the-King/Advent-I evidence dates remain in 2023.
- A genuine missing-text repair exposed a split-reference parser gap. Added a
  bounded sibling-block fallback and 11 regression cases; replayed the successful
  captured page through the existing crawler/TSV pipeline and inserted only five
  missing Mark rows on the temporary copy. No existing text/reference, prior
  review or observation changed. `Mc 13,24-32` remains missing.
- Repeated audits and review application are idempotent. Full Go/race/build/static
  checks, 18 calendar tests, 24 frontend tests/lint/build and **1,115 consumer/date
  checks** passed. Warm reuse and genuine pending/unsupported gaps made zero HTTP;
  default monitoring still fails honestly. Both shipped databases are unchanged.
- Current national counts: **8 calendar-resolved, 1 unreviewed, 356 unsupported**.
  No new national definitions, rules version, schema or production activation.
  Nothing pushed or deployed. To finish the source contract, obtain the applicable
  approved national calendar and lectionary approval/edition/options material;
  do not substitute another local calendar or infer the first web option.

Details: [`data/vietnam-content-audit.md`](data/vietnam-content-audit.md) and the
machine-readable non-executable review ledger/capture fingerprints alongside it.

## Design Rules

- Never use the existing estimated seasonal key as sufficient proof of equivalence.
- Never promote an observation from one date into a universal reading rule solely
  because several generated keys or observed references match.
- Reuse requires an established lectionary rule/mapping and a supported calendar
  resolution, with evidence and scope recorded for both.
- A celebration name alone is not a complete cache key: cycles, vigil/day Masses,
  long/short forms, and other permitted options can change the Gospel selection.
- Vietnamese-language Vatican News is a source, not proof of a Vietnamese national
  or diocesan calendar. Record source scope separately from language.
- Do not silently substitute ordinary readings when a proper celebration or local
  choice is unresolved. Conversely, a source from another calendar must not
  automatically invalidate an already established local reading.
- Preserve existing verse text, legacy references, and source observations for
  audit. Do not label unsupported legacy data as verified just to retain coverage.
- Cached evidence does not require daily refreshing. Recheck deliberately or when
  relevant rules, source mappings, or content change.

## Phase 1 — Establish the Calendar Contract

- [x] Confirm the site's default: Vietnam's approved national calendar, daytime
  Mass, no assumed diocesan/parish overlay (owner agreement, 2026-11-10).
- [ ] Identify authoritative calendar and lectionary references for that scope;
  document Vatican News's role and limitations.
- [x] Define the default Mass/reading option: follow an authoritative assignment;
  otherwise the first-listed full authorized Gospel in the approved lectionary.
  Applying this still requires the selected formulary and allowed set to be
  established. Distinguish legitimate alternatives from incorrect selections.
- [x] Resolve the November 15 discrepancy against the chosen calendar, or retain
  an explicit unresolved status. Do not hard-code a presumed local explanation.
- [x] Record calendar identity, rules version, and supported exceptions. Unsupported
  cases must be visible rather than falling back to an ordinary weekday.

**Deliverable:** a documented calendar/source policy. Architecture and tests can
proceed before this decision, but production-wide reference repair cannot.

## Phase 2 — Define Reusable Entries and Evidence

Design an additive schema migration, building on `003_reading_sources.sql` rather
than discarding the existing evidence. Final table names can follow implementation
needs, but the model must distinguish:

| Concept | Required information |
| --- | --- |
| Calendar policy | Identity, applicable local rules, version, default options |
| Lectionary entry | Stable identity, celebration/formulary, applicable cycles and option, canonical Gospel reference |
| Entry evidence | Source, actual source date where relevant, scope, review status, applicable variants |
| Date resolution | Date + calendar + Mass option, selected entry, resolution method, rules/mapping versions |
| Source observation | Requested date, actual page date, observed reference, source scope, retrieval time, conflict/review history |

- [x] Keep `verses` as the shared passage store; do not duplicate its text per date
  or celebration.
- [x] Define reference normalization using the existing parser, preserving suffixes,
  discontiguous segments, and genuinely different long/short options.
- [x] Define statuses such as `source_checked`, `calendar_resolved`,
  `legacy_unverified`, `unresolved`, and `conflict`.
- [x] Retain multiple observations/review decisions where needed instead of
  overwriting the only record of a disagreement.
- [x] Separate direct date evidence from entry-level evidence. A source from an
  earlier year may support an entry, but must never be presented as a page fetched
  for the newly resolved date.
- [x] Define dependency-based invalidation: changed rules or mappings affect their
  dependent resolutions, not every cached reading.

**Deliverable:** schema and lookup contract with explicit evidence semantics.

## Phase 3 — Correct Date-to-Entry Resolution

Primary files: `internal/lectionary/types.go`, `calendar.go`, and `generators.go`.

- [x] Separate the seasonal estimate from the actual selected observance/entry.
- [x] Implement precedence for the candidate calendars, including proper feasts on
  Sundays where allowed, protected Sundays/seasons, and transferred solemnities.
- [x] Fix the omissions exposed by November 9 and the blanket Sunday exclusion in
  `hasDateSpecificReadings()`. Do not simply allow every fixed celebration to
  replace every Sunday.
- [ ] Add approved national/local rules only within their declared calendar scope.
  The explicit 2026 pilot now implements nine sourced definitions; this item
  remains incomplete for the full national calendar. Source facts are recorded
  in `data/vietnam-national-sources.json`, decisions in `data/vietnam-pilot-2026.json`.
  Assigned Mass and underlying season are separate; Hours are not implemented.
  A transferred Ash Mass does not shift the start of Lent.
- [x] Resolve cycles and Mass options before constructing a reusable entry identity.
- [x] Produce a structured explanation: selected observance, displaced seasonal
  entry where relevant, rules version, and any unsupported/ambiguous choice.
- [x] Derive celebration labels from the resolved observance, not the old key.

**Deliverable:** deterministic, offline calendar resolution with reviewed fixtures.

## Phase 4 — Make Crawling Cache-First

Primary files: `tools/lectionarycrawler/main.go` and `verification.go`.

- [x] Use this sequence:
  1. Reuse a valid accepted date resolution for the requested calendar/option.
  2. Otherwise resolve the date and reuse a trustworthy cached entry/reference.
  3. Check structural verse availability; report missing text for a necessary,
     reviewed repair through the existing verse pipeline (not a reference recrawl).
  4. Crawl a dated page only for a genuine gap, supported ambiguity investigation,
     requested refresh, or selected audit.
  5. Retain unresolved cases when available sources cannot establish the target
     calendar's selection.
- [x] Do not treat an unavailable future page as a missing reading when calendar
  resolution and a trustworthy cached entry already establish it.
- [x] Keep exact page-date validation for every dated crawl. Wrong-date pages must
  never become evidence for the requested date.
- [x] Keep reviewed-correction safeguards, but scope conflicts to calendar and Mass
  option; a different permitted option is not automatically a data error.
- [x] Preserve source observations independently of acceptance. Accepting a
  date-specific correction must not automatically change a reusable entry or all
  dates that depend on it.
- [x] Keep operations idempotent and transactional; never update references merely
  by matching the old `lectionary_key`.
- [x] Preserve cancellation, request timeouts, bounded retries, rate limiting, and
  SQLite's single-connection behavior. Do not hold transactions during HTTP calls.
- [x] Report cache hits, resolved dates, HTTP requests, missing entries/text,
  disagreements, and accepted corrections separately.

**Deliverable:** a populated, supported calendar window resolves with zero HTTP
requests unless an audit or refresh is explicitly requested.

## Phase 5 — Audit and Reuse the Existing Database

- [x] Work on a consistent temporary copy of `resources/bible.db`; retain a baseline
  hash/backup and produce a reviewable difference report.
- [x] Inventory existing refs, source records, recurring entries, and collisions.
- [x] Resolve all existing dates offline against the candidate calendar first.
  Classify rows as supported matches, potential reusable candidates, disagreements,
  unsupported calendar cases, or missing text/reference data.
- [x] Seed reusable mappings from reviewed lectionary evidence and applicable
  existing observations. Do not automatically trust each old key's first reference
  or its majority reference.
- [x] Demonstrate restored date coverage through supported entry reuse without
  fetching a dated page for each historical row (46 available candidate dates).
- [ ] Review remaining archive mappings and restore coverage under the confirmed
  national target after its authoritative rules/source contract are reviewed.
- [ ] Crawl only remaining gaps and a risk-based audit sample: proper celebrations,
  transferred dates, Sundays, local exceptions, and dates affected by prior key
  collisions. Audit ordinary dates as controls too.
- [ ] Review disagreements before applying bounded corrections. Preserve legacy
  values and explanations; do not perform an unaudited archive-wide overwrite.
- [x] Report trusted coverage, unresolved coverage, reference/label changes, cache
  reuse, and actual crawl count. Repeat the run to confirm idempotence.

**Deliverable:** a reviewed repair snapshot and evidence report, not a blanket
recrawl or an automatic promotion of legacy predictions.

## Phase 6 — Update Consumers and Scheduled Jobs

- [x] Update `internal/reading/reading.go`, API handlers, and CLI to accept either
  matching direct-date evidence or a supported resolution backed by a trustworthy
  cached entry. Simply removing the provenance guard is not sufficient.
- [x] Return the resolution method and genuine evidence. Define compatible
  semantics for `verified`, `source_url`, and `verified_at`; do not fabricate an
  exact-date source URL or a fresh crawl timestamp for a reused entry.
- [x] Update `gospel-site/lib/reading.js`, labels, source display, and tests to use
  the same contract. Distinguish direct observation from calendar-based reuse.
- [x] Keep unsupported/conflicting selections unavailable with useful diagnostics;
  absence of a dated page alone must no longer cause a 404/unavailable result.
- [x] Update `.github/scripts/probe-readings.sh` and the frontend checker to test
  supported resolution plus usable text, not require a dated crawl for every day.
- [x] Make healing fill the smallest actual gaps. Yearly calendar generation should
  reuse existing entries rather than crawl every generated date.
- [x] Keep scheduled source audits separate from availability probes and compare
  actual dated references, not merely HTTP success.
- [ ] Add authoritative celebration/allowed-alternative comparison for the chosen
  local calendar once its source contract is approved; do not infer a celebration
  solely because a Gospel reference matches.
- [x] Update `README.md` and `AGENTS.md` to explain reuse, evidence, calendar scope,
  correction review, and operator commands.

## Phase 7 — Tests and Controlled Rollout

### Regression and reuse tests

- [x] November 9 on a weekday (2026) and Ordinary Sunday (2025): resolve the Lateran
  dedication and `Ga 2,13-22`, not the estimated weekday/Sunday entry.
- [x] November 1, 2026: reviewed All Saints selection `Mt 5,1-12a`.
- [x] April 8, 2024: transferred Annunciation selection `Lc 1,26-38`.
- [x] November 11, 2026: no leakage of the Lateran Gospel; reviewed `Lc 17,11-19`.
- [x] Same ordinary entry across eligible dates/years reuses one reference; a
  superseding celebration does not share that entry.
- [x] Weekday Gospel-only entries reuse across I/II first-reading cycles; Sunday
  A/B/C, proper choices, Mass options, and calendar scopes do not accidentally
  share incompatible selections. Equivalent Gospel mappings may share text/reference
  storage without conflating their date-resolution rules.
- [x] November 15 remains explicitly unresolved until the calendar decision is
  supported; a wrong-scope observation cannot silently replace a local selection.
- [x] Future dates and historical dates resolve offline when dependencies are
  supported and cached. Assert zero HTTP requests on warm-cache runs.
- [x] Missing rows/suffixes/empty text, wrong-date pages, stale rules/mappings,
  duplicate imports, transaction failures, and cancellation behave safely.
- [x] Valid cached readings remain usable through a network outage; explicit
  invalidation is distinguished from an unsuccessful optional refresh.
- [x] API, CLI, frontend, and health probes agree on reference, label, resolution
  method, evidence, and unavailable/conflict behavior.

### Validation

Use temporary databases and preserve existing fixture/isolation conventions.
Always use the FTS5 tag for Go builds, tests, and runs:

```sh
go test -tags "fts5" -count=1 ./...
go test -tags "fts5" -race -count=1 ./...
go build -tags "fts5" ./...
staticcheck -tags "fts5" ./...
actionlint
```

In the companion frontend, run its unit tests, lint, reading checker, production
build, and production-mode page checks against the reviewed snapshot.

### Release gates

- [x] Target calendar scope and editorial default options are approved.
- [ ] Authoritative national rules, historical applicability, lectionary editions
  and evidence contract are reviewed; a complete supported national adapter is
  available (the nine-definition pilot does not satisfy this gate).
- [x] Blanket key propagation is removed, and direct-date proof is no longer the
  sole availability contract: explicit calendar profiles support reviewed reuse
  without a legacy fallback. The deliberately source-only default stays date-bound
  until the confirmed national target has reviewed definitions, mappings and coverage.
- [ ] Coverage and data differences are reviewed; any remaining unavailable dates
  are explicit, not a surprise caused by migration.
- [ ] Representative source comparisons validate liturgical correctness, not merely
  API/CLI parity or successful rendering.
- [ ] Backend and frontend migrations, snapshot format, and deployment order are
  coordinated, including compatibility with their automatic database-sync jobs.
- [ ] Prepare rollback snapshots and commits for both repositories. Checkpoint WAL
  data before packaging snapshots.
- [ ] Publish only after approval; verify the live Gospel, celebration label,
  evidence display, near-term dates, and representative archived dates afterward.

## Definition of Done

The supported calendar selects the correct lectionary entry before reference
reuse. Gospel text and trustworthy entry mappings are reused across applicable
dates, with no per-date crawl requirement. Genuine gaps and audits drive network
requests. Local-calendar choices and reading alternatives are explicit, conflicts
are reviewable, historical coverage is recovered where evidence supports it, and
backend/frontend output identifies how each reading was established.
