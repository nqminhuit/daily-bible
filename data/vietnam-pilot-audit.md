# Vietnam national pilot — 2026-11-10

## Scope and result

This is an **explicit partial 2026 pilot on temporary database copies**, not a
complete national calendar, approved lectionary edition, archive repair or release
snapshot. The owner's national/daytime/no-assumed-local-overlay target remains
confirmed. Runtime defaults, shipped databases and production are unchanged.

`tools/calendar/vietnam-pilot.cjs` implements the reviewed subset in
`data/vietnam-pilot-2026.json`, corroborated by the primary-source dossier. It uses
the pinned **General Roman** engine only as underlying calculation context; it
never uses the rejected Vietnam plugin or treats unreviewed base dates as
national decisions. Only 2026 is supported; other years fail explicitly.

The manifest includes all 365 dates, but only **nine source-backed definitions**:
Tết/Ash Mass assignments on February 18–20; Easter April 5; Pentecost May 24;
Corpus Christi June 7; All Saints November 1; Christ the King November 22; Advent I
November 29. The other **356 dates have no reusable entry**. Ascension May 14/17,
Epiphany and the Vietnamese Martyrs November 15/24 retain specific ambiguity
reasons. Even a plausible universal feast is not silently grandfathered.

| Classification | Definitions-only | After two explicit Gospel reviews |
| --- | ---: | ---: |
| `calendar_resolved` | 0 | 2 |
| `unreviewed_entry` | 9 | 7 |
| `unsupported_calendar_date` | 356 | 356 |

The two reviewed/renderable demonstration selections are:

| Date | Selected Mass | Gospel |
| --- | --- | --- |
| 2026-02-20 | Lễ Tro | `Mt 6,1-6.16-18` |
| 2026-11-01 | Các Thánh Nam Nữ | `Mt 5,1-12a` |

The [reference evidence inventory](vietnam-pilot-reading-evidence.json) retains
actual HĐGMVN publication URLs, captured response fingerprints, reading-heading
dates and distinct publication dates. Both URLs have old slugs; neither was
rewritten to manufacture a dated page. The inventory does not auto-approve anything.
The final copy received two deliberate CLI reviews, not imported approvals.

The printed references/named Masses and local pericope clause coverage were
reviewed. HĐGMVN's printed passages and the Vatican-derived local text have
different Vietnamese wording. This is **not an exact-word match or approved-edition
certification**. The existing Matthew 5:12 base row contains the selected liturgical
12a fragment, not a certified complete canonical verse. No text was overwritten.
A complete approved edition/option-set contract remains a release prerequisite.

## Model and safeguards

- **Schema 3** adds explicit Mass assignments, seasons, source URLs/notes and
  unsupported reasons. Schema-2 manifests are rejected before mutation so an old
  importer cannot ignore the Mass assignment. Existing supported caches/review
  histories are not relabeled. Changed exports require explicit re-review.
- Migration **005** stores each exact definition in `reading_calendar_day_details`,
  atomically with the compact selected-entry record. Definition/evidence changes
  under the same version are rejected. Rollback/cancellation conventions remain.
- Assigned Masses are not fabricated liturgical ranks. The February 18 underlying
  observance remains Ash Wednesday and the season remains Lent, while the selected
  Mass is Mồng Hai Tết. February 20 retains Friday-after-Ash underlying context but
  selects the Ash Mass. No Hours resolver or season transfer is inferred.
- Assigned-formulary identities conservatively retain A/B/C and I/II context until
  broader lectionary applicability is established. Ordinary weekday Gospel annual
  reuse remains unchanged. The named Ash Mass rejects the underlying Friday's
  `Mt 9,14-15` as an incorrect review.
- National supported definitions require their own HTTPS evidence and scope note.
  Pilot warm lookups/reviews/crawl candidates also require matching definition
  evidence; renaming a version or retaining only a warm Gospel row is insufficient.
- Corrupt auxiliary definitions cannot invalidate an independent, valid exact-date
  choice. Derived entries still fail closed; audits retain a definition-error
  diagnostic rather than confusing it with missing verse text.
- Unreviewed assigned national Masses do **not** trigger a dated Vatican reference
  crawl. Their proper evidence needs review. Explicit source audits of supported
  mappings remain separate and cannot replace national selections automatically.
- Go API/CLI and frontend expose the selected Mass, underlying context and precise
  calendar directive separately from Gospel evidence. The page links both sources
  and visibly calls this a limited pilot, not “Vietnam (Romcal)”.

## Reproduce the definitions-only audit safely

```sh
mkdir -p build
sqlite3 -readonly resources/bible.db ".backup 'build/vietnam-pilot.db'"
node tools/calendar/vietnam-pilot.cjs 2026 > build/vietnam-pilot.json
DB_PATH=build/vietnam-pilot.db READING_CALENDAR=vietnam \
  go run -tags "fts5" ./tools/lectionarycrawler \
  -years=2026 -calendar-file=build/vietnam-pilot.json \
  -offline -audit-report=build/vietnam-pilot-audit.json
```

This approves **zero** Gospels. Review references/applicability separately using
`-review-entry`; do not treat the evidence inventory, first/majority legacy ref or
a successful observation as permission to approve it. Automatic
`export.cjs vietnam` / `prepare-calendar.sh` paths remain blocked. Do not set
production/workflow `READING_CALENDAR=vietnam` to this incomplete pilot.

## Validation and preservation

- Uncached Go suite, race suite, all builds, Staticcheck, workflow lint and shell
  syntax checks passed.
- **14 offline calendar tests**, including four actual pilot tests, passed.
- **24 frontend tests**, lint and optimized production build passed.
- All **365 pilot dates plus two outside-year controls** agreed across API, CLI
  and frontend lookup. Nine production-mode pages covered both available dates,
  unresolved dates and an outside-year control: **1,110 consumer/date checks**.
  Reference, text, labels, source links, methods, versions, Mass and season metadata
  agreed. Limited-pilot attribution and genuine historical URLs remained visible.
- Repeated export/import/audit was byte-identical. Warm Ash lookup: one cache hit,
  **zero HTTP**. An assigned Tết reference gap and unresolved November 15 each
  failed honestly with **zero HTTP/candidates**; no wrong-scope crawl was attempted.
- Default November 10–12 national-pilot monitoring correctly fails for unsupported
  definitions. Its default window was not shortened or made green by approvals.
- Every one of the **2,191 original date/reference pairs** and all **2,917 shared
  verse rows** were unchanged. Fifteen compatibility observations were retained;
  no fresh Vatican observation was added. Integrity and FK checks passed.
- Both shipped SHA-256 hashes remain
  `e4cbcfe55f603c4e7cdb7c36bf8a3075469d72500fe490fab122c330b36ad1c7`.

Final evaluated rules version:
`vietnam-national-pilot-2026-0db4f1f3fdf6cd9a137a`.

Temporary artifacts: `/tmp/daily-bible-vietnam-pilot.FEpcYL/` contains the final
manifest, consistent `final-audit.db`, before/after/repeat reports, two explicit
reviews, captured publications/status log, validation logs, independent runtime
copies, full-year parity script/results and expected monitoring failure. The
initial evaluation remains separate; after the adapter hash changed, both
mappings were explicitly re-reviewed on a fresh final copy, not relabeled.
These temporary copies are **not release assets**.

## Remaining work

Obtain/review the full approved national calendar and lectionary editions,
historical applicability, transfer/default choices and other national propers.
Resolve November 15's national celebration/formulary rather than choosing either
legacy or Vatican reference by default. Extend the national definitions and
review queue, approve reusable mappings/legitimate exceptions, repair necessary
text through the existing verse pipeline, then audit national archive coverage
and prepare coordinated backend/frontend snapshots and rollback assets.

Two renderable dates and nine supported definitions prove the pilot mechanics;
they do not satisfy production/archive release gates. Nothing was pushed or deployed.
