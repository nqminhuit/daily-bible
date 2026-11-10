# Plan: Correct Daily Gospels with Safe Cache Reuse

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
