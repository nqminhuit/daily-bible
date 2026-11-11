# Handoff — daily-bible reading/calendar repair

Updated **2026-11-11**, Asia/Ho_Chi_Minh. This is the current handoff, not a restoration of the obsolete review documents.

## 2026-11-11 update (wins over any older statement below)

- **Pilot is now 9 renderable / 0 unreviewed / 356 unsupported.** Mồng Ba Tết was explicitly reviewed as `Mt 25,14-30` at `2026-11-11T03:13:10Z`: first-listed on the dated HĐGMVN page, listed alone by Hà Nội, Vĩnh Long and a CGKPV-credited Hải Phòng reproduction. `Ga 5,16-20` stays a recorded alternative. The lectionary's own pages were still not inspected; use `-replace-entry` if the book disagrees.
- Definitions and rules version are unchanged. Current temporary DB: `build/vietnam-rules-2026-11-11/audit.db` (ignored; SHA-256 `7da1e3dedcfd6bc2dbd43d60d447dc72e11ba8ae42ad43a30faf6f93ea65c613`), ten review records, nine active mappings, 2,922 verse rows. Captures and request log sit beside it.
- **Two blockers are owner decisions, not missing documents.** Do not keep searching for a national default that does not exist:
  1. **November 15**: Hà Nội's official calendar keeps Sunday XXXIII (`Mt 25,14-30`) and says the Martyrs *may* be solemnly celebrated "tùy theo nhu cầu mục vụ (HĐGMVN, 4-1991)". HĐGMVN's site published the Martyrs Mass on Sunday 33 in 2018/2021/2025 and the Sunday Mass in 2023. The owner must choose the site default; the written policy ("no optional celebration") points to the Sunday.
  2. **Ascension**: Hà Nội keeps Thursday May 14 (obligatory); Vĩnh Long and Phan Thiết keep Sunday May 17; Bắc Ninh says either. The owner must choose Thursday or Sunday for a no-diocese site.
- **November 24** is printed as a solemnity by Hà Nội and Vĩnh Long (CD 878/90) with `Lc 9,23-26` first, `Ga 17,11b-19` second (also Hải Phòng, Huế). It stays unsupported because `vietnam-pilot.cjs` accepts only hdgmvietnam.com directive facts and the decree was not obtained. Accepting official diocesan calendars as evidence of cited national law is a third owner decision and would change the rules version (all nine entries then need explicit re-review).
- Epiphany January 4 (Sunday) agrees in Hà Nội and Vĩnh Long, with no primary national 2026 fact. Unsupported.
- **The backend branch is on origin**: `origin/fix/date-verified-mass-readings` equalled local `5483740` after a fresh fetch on November 11. The older "nothing pushed" statements below are wrong for the backend. The frontend worktree has no upstream. Nothing was deployed; this session pushed nothing.
- This session: 17 captured requests (all 200) plus four tool fetches; Đà Lạt returned 403 and was not bypassed. 19 calendar tests pass. Frontend repo docs still describe the Tết default as pending.
- Details: `data/vietnam-content-audit.md` (last section) and `rule_findings` in `data/vietnam-content-review.json`.

## Read this first

- The original code-review remediation is finished, pushed and previously passed
  CI. The current task is **liturgical correctness and safe reading reuse**.
- The owner already confirmed **Vietnam's national calendar, daytime Mass, no
  assumed diocesan/parish overlay** on November 10. **Do not ask the jurisdiction
  question again.** Follow an authoritative Gospel assignment; otherwise use the
  approved lectionary's first-listed full authorized option. This does not resolve
  an unknown formulary, select an optional celebration or choose the first web result.
- Cache architecture and consumer support are implemented. The bottleneck is now
  **authoritative national rules, approved lectionary/option evidence and coverage**,
  not more framework work.
- The latest temporary national pilot has **8 renderable dates, 1 unreviewed entry
  and 356 unsupported dates**. Only nine date-to-Mass definitions are supported.
  This is **not a complete national calendar, archive repair or release candidate**.
- **Nothing from the reading-fix branches has been pushed or deployed.** Both
  shipped databases are unchanged by the latest continuations. Production is
  unchanged. Target approval is not data, Gospel, snapshot or deployment approval.
- All latest code/data work is locally committed. No implementation/validation
  process is running. Both worktrees were clean before this handoff was written.

## Repositories, commits and production

| Item | Current location/state |
| --- | --- |
| Backend | `/opt/minh/test/daily-bible` |
| Backend origin | `git@github.com:nqminhuit/daily-bible.git` |
| Frontend | `/tmp/gospel-site-mass-fix.afwvYH` |
| Frontend origin | `git@github.com:nqminhuit/gospel-site.git` |
| Both branches | `fix/date-verified-mass-readings` |
| Backend latest code/data commit | `db1b3b2` — `data(readings): document Vietnam pilot content recovery` |
| Preceding parser fix | `2dd6fbe` — `fix(parser): read separately labeled Gospel references` |
| National pilot implementation | `8eee3fb` — `feat(calendar): add source-backed Vietnam 2026 pilot` |
| Target/source dossier checkpoint | `b205f3e` — `test(calendar): record Vietnam target and primary-source rule cases` |
| Frontend latest commit | `06efe91` — `docs(readings): record pilot recovery and source limits` |
| Frontend preceding implementation | `8576578` — `feat(readings): preserve national Mass evidence and pilot diagnostics` |
| Backend local `origin/master` tracking ref | `e13d663290bd074d5b9f2c10ea163a13baba93a1` |
| Frontend local `origin/master` tracking ref | `4c23dc24c0662545164516ae2b58cf0db59a5d17` |

A handoff-only commit may be newer than the backend code/data commit above.
Tracking refs were inspected locally, not freshly fetched. No push was performed.

Production: **https://loichua.prud.uk/**; date navigation is `/?date=YYYY-MM-DD`.
The separate frontend reads its **committed SQLite snapshot through sql.js**,
not the Go API or a request-time download. Its host's Go API/health URLs return
HTML 404s; do not mistake those for the Go server's endpoint behavior.

Verified frontend bundle:

- `build/gospel-site-calendar-cache.bundle`
- Bundle HEAD: `06efe91fbde7a39754a893c615b227dad33165b8`
- Required base: `4c23dc24c0662545164516ae2b58cf0db59a5d17`
- Created with `origin/master..HEAD`; `git bundle verify` passed.
- `build/` is ignored; the bundle is a handoff aid, not a release snapshot.

Both shipped database SHA-256 values are still:

```text
e4cbcfe55f603c4e7cdb7c36bf8a3075469d72500fe490fab122c330b36ad1c7
```

Paths: backend `resources/bible.db` and frontend `resources/bible.db`. These
already contain earlier **local interim corrections**, not the new national
cache/recovery work. Do not confuse unchanged local shipped files with the
unchanged, older deployed production data.

## Brief task history (do not redo completed remediation)

The initial November 9 investigation found the live site selecting `Lc 17,1-6`
instead of the Lateran dedication Gospel `Ga 2,13-22`. Root causes were unsafe
reuse through incomplete estimated calendar keys and a separate frontend snapshot.
An interim exact-date-only safeguard stopped false reuse, then the owner correctly
challenged mandatory per-date crawling. The reusable scoped cache replaced that
overly restrictive design; trustworthy calendar equivalence is the right fix.

The six earlier reviewed corrections remain in the local shipped baseline:

| Date | Previous reference | Reviewed reference |
| --- | --- | --- |
| 2024-04-08 | `Ga 3,1-8` | `Lc 1,26-38` |
| 2024-11-09 | `Lc 16,9-15` | `Ga 2,13-22` |
| 2025-11-09 | `Lc 20,27-38` | `Ga 2,13-22` |
| 2026-11-01 | `Mt 23,1-12` | `Mt 5,1-12a` |
| 2026-11-09 | `Lc 17,1-6` | `Ga 2,13-22` |
| 2026-11-11 | `Ga 2,13-22` | `Lc 17,11-19` |

Original remediation also fixed configured readiness paths, empty-reading 404s,
crawler persistence-before-processed semantics, strict/bounded reference parsing,
cancellation/rollback/output failures, parser/CORS tests and crawler configuration
separation. Original CI passed:
`https://github.com/nqminhuit/daily-bible/actions/runs/42575985528`.
That is historical CI for the original remediation, **not CI/deployment of the
unpublished national-calendar continuation**.

## Essential project rules

Read `AGENTS.md`, then `data/calendar-policy.md`.

- **Always use `-tags "fts5"` for Go build/test/run** involving this project.
- Preserve SQLite's single connection, context cancellation, bounded retries and
  timeouts, transactions and rollback. Never hold a transaction during HTTP.
- Use consistent temporary SQLite backups; audit before considering promotion.
- Preserve legacy references, shared text, observations and append-only review
  history. No majority/first-reference or successful-crawl auto-approval.
- Missing text requires reviewed repair through the existing verse crawler/TSV
  pipeline; another reference crawl cannot repair it. No automatic text ingestion.
- No fabricated source URL, evidence date or fresh crawl/review timestamp on reuse.
  Request fingerprints/times are research capture facts, not review times.
- Do not bypass source access controls. Keep genuinely unresolved cases visible.
- Only `READING_MASS_OPTION=day` is implemented. No guessed vigil/night/dawn option.
- Calendar import, Gospel review and production activation are separate operations.
- Do not shorten monitoring defaults or approve data just to make checks pass.
- `PLAN.md` and `code-review.md` remain deleted. Active `plan.md` is lowercase.
  `HANDOFF.md` was newly created here at the owner's explicit request.

## Implemented architecture and safeguards

1. Shared verse text: `verses(book, chapter, verse, verse_suffix, text)`.
2. Reviewed lectionary-entry → Gospel mapping, scoped by calendar/option/version.
3. Calendar-aware date → entry resolution with independent exact-date choices.

Important files/packages:

- `internal/reading/`: imports, reviews, lookup, observations, grouped audits and
  date decisions; `internal/scripture/`: strict reference parsing/query/completeness.
- Migrations `003_reading_sources.sql`, `004_reading_cache.sql`,
  `005_calendar_day_details.sql`; the maintenance tool applies these idempotently.
- `tools/lectionarycrawler/`: cache-first maintenance, explicit reviews and offline
  audits. Warm supported selections make zero Gospel HTTP requests.
- `tools/calendar/export.cjs`: pinned maintenance-only General Roman evaluation.
  Dependencies `romcal` and both calendar packages are `3.0.0-dev.144`.
- `tools/calendar/vietnam-pilot.cjs` + `data/vietnam-pilot-2026.json`: explicit,
  limited 2026 source-backed national subset; **other years are rejected**.
- Frontend `lib/reading.js`, `lib/refParser.js`, `scripts/check-readings.mjs` and
  `app/page.js` match Go selection/text semantics and display actual evidence.
  Rendering and diagnostics share the resolver; monitoring opens one consistent
  in-memory snapshot for its entire window, not a fresh DB per date.

Runtime default remains **`READING_CALENDAR=vatican-vi`** (source-only policy),
not national activation. `roman-general` remains evaluation-only.

**Automatic Vietnam Romcal exports/imports/reusable approvals and derived warm
lookups are blocked.** The pinned plugin puts Vietnamese Martyrs on November 13
with a wrong date/rank profile and leaves November 24 a weekday. Do not enable it,
rename its export, substitute Roman General as national rules, or patch two dates
and call the resulting profile reviewed. National rules need an explicit reviewed
version; a diocesan/parish policy needs a distinct implemented adapter/namespace.

Current **Gospel entry schema is 3**. Old/missing schemas are rejected before
mutation; regenerate and explicitly re-review changed dependencies. Never relabel
old approvals. Weekday Gospels repeat annually across I/II first-reading cycles;
this does not authorize sharing whole Mass readings. Sundays retain A/B/C and
proper/memorial/formulary identities remain conservative.

Schema 3/005 distinguish **assigned Mass, underlying observance and seasons**,
with per-day evidence and unsupported reasons. February 18, 2026 remains Lent's
start/Ash Wednesday Hours while its Mass is Mồng Hai Tết; the Ash Mass is February
20. **Do not move the entire season, Easter or the Hours.** Hours are not implemented.

Lookup prefers a valid current-version exact-date decision, then its reviewed
entry. No cross-calendar/option/version/legacy fallback. Stale explicit decisions
block fallback. Same-canonical-page observations can quarantine source-only
selection, not independently reviewed calendar choices. Corrupt auxiliary
calendar metadata likewise cannot invalidate independent valid exact-date evidence.

Outputs distinguish `source_checked` and `calendar_resolved`. `verified: true`
means supported selection evidence, **not a fresh crawl**. Inspect `calendar_id`,
`mass_option`, `rules_version`, `entry_key`, `source_url`, `evidence_date`,
`verified_at`, `calendar_source_url`, `calendar_review_note`, `mass_formulary`,
`calendar_observance`, `temporal`, `seasons`, `celebration` and the estimate marker.
Structural verse coverage is not word-level/approved-edition certification; some
legacy base verses contain liturgical fragments (e.g. Matthew 5:12's 12a portion).

## Latest national pilot result

Rules version: **`vietnam-national-pilot-2026-0db4f1f3fdf6cd9a137a`**.
Definitions/rules version were **not changed** in the latest content pass.

| Date | National pilot Mass | Gospel | Actual evidence date |
| --- | --- | --- | --- |
| 2026-02-18 | Mồng Hai Tết, ancestors/parents | `Mt 15,1-6` | 2026-02-18 |
| 2026-02-19 | Mồng Ba Tết, work | **unreviewed default** | Published alternatives retained |
| 2026-02-20 | Transferred Ash Mass | `Mt 6,1-6.16-18` | 2026-02-20 |
| 2026-04-05 | Easter daytime/morning | `Ga 20,1-9` | 2026-04-05 |
| 2026-05-24 | Pentecost A daytime | `Ga 20,19-23` | 2026-05-24 |
| 2026-06-07 | Corpus Christi A | `Ga 6,51-58` | Explicit June 7 assignment in annual Ordo |
| 2026-11-01 | All Saints | `Mt 5,1-12a` | 2026-11-01 |
| 2026-11-22 | Christ the King A | `Mt 25,31-46` | **2023-11-26** |
| 2026-11-29 | Advent I B | `Mc 13,33-37` | **2023-12-03** |

The initial checkpoint had two renderable dates; six of the other seven mappings
were explicitly reviewed. Before text repair: seven renderable, one missing text,
one unreviewed, 356 unsupported. After repair: **8 / 0 / 1 / 356**.

Current temporary DB: one calendar, eight active entry mappings, **nine review
records** (one explicit Corpus citation refinement), zero date reviews,
15 unchanged observations, 2,191 original date rows and **2,922 verse rows**.
All 2,191 original date/reference pairs and **all 2,917 original verse rows** are
unchanged. Exactly five missing verse rows were added. Prior Ash/All Saints
review timestamps remain `2026-11-10T06:48:31Z`; actual subsequent review times
and superseded Corpus evidence are retained in the ledger and DB.

### Important unresolved/evidence details

- **Mồng Ba Tết**: the dated HĐGMVN page publishes full `Mt 25,14-30` and
  `Ga 5,16-20`; the diocesan Ordo lists Matthew alone. This does **not** establish
  the approved national lectionary's default ordering. No entry approval or
  date-acceptance bypass was made. Do not pick the first web/legacy candidate.
- **Corpus Christi**: HĐGMVN's body header prints `Ga 6,51-59`, its tag and
  the Ordo page 24 give `Ga 6,51-58`. The actual Gospel body ends at canonical 58;
  local 59 is the distinct synagogue-location sentence. The active primary Gospel
  citation is the **Ordo PDF**, not the ambiguous header. Evidence date June 7
  denotes its explicit table assignment, not a fresh daily page. Original HĐGMVN
  evidence remains in review history; no global 58/59 alias or text overwrite.
- Christ-the-King and Advent-I reviews deliberately keep actual **2023** evidence.
  Never synthesize 2026 dated pages or overwrite their source dates on warm reuse.
- Tết source chants/Alleluia are not ingested. Gospel reuse does not approve all
  Mass texts or override Lent's actual start.

## Sources found and actual blockers

**A complete approved national baseline and lectionary contract were not
established.** Target choice is settled; do not frame that as the remaining blocker.

Latest durable source/review files:

- `data/vietnam-content-audit.md` — latest full report, limitations and validation.
- `data/vietnam-content-review.json` — six scoped reviews, actual review times,
  pending Tết options, partial source contract and five-row repair facts;
  **non-executable / `auto_approve: false`**.
- `data/vietnam-content-captures.json` — 12 actual response/text fingerprint pairs.
- `data/vietnam-pilot-audit.md` — historical initial two-reading checkpoint.
- `data/vietnam-pilot-reading-evidence.json` — original two Gospel sources.
- `data/vietnam-national-sources.json` — earlier six-source partial national dossier,
  not an executable calendar or Gospel approval.
- `data/calendar-research.md`, `data/calendar-policy.md`, `plan.md` — policy/history.

The official **Vĩnh Long** 52-page *Lịch Công Giáo Năm Phụng Vụ 2025–2026, Năm A*
was found, but is **diocesan, not a national calendar**:

- Landing: `https://giaophanvinhlong.net/lich-cong-giao-giao-phan-vinh-long-2025-2026/`
- PDF: `https://giaophanvinhlong.net/wp-content/uploads/2025/12/38e8267e-b839-4990-9ec9-005af8333e3f.pdf`
- Publisher: Tòa Giám Mục Vĩnh Long / NXB Đồng Nai, 2025. Landing date December 2,
  2025; captured PDF metadata modified January 3, 2026. Hashes identify the
  captured bytes, not a frozen original publication version.
- It contains local observances/schedules and apparent typos. Page 10 prints the
  Ash transfer year as 2025; page 36 prints `Lc 9,11-28` for November 18. No
  wholesale import or silent corrections were made.
- Page 36 lists both Sunday 33 and Martyrs external solemnity, readings from the
  Commons, and cites **CD 878/90** for November 24/national patrons. This is a
  useful decree lead, **not the actual decree, established national default or
  authorized default Gospel**. November 15/24 remain unsupported.

`https://ktcgkpv.org/books?page=2` lists *Sách Bài Đọc Trong Thánh Lễ (3 quyển)*,
**2017**, out of stock. Catalogue presence is **not approval/frontmatter,
authorized option order or proof that website readings use that edition**.
The 2023 HĐGMVN Missal interview announcement is not lectionary promulgation.

Other unresolved source requirements:

1. Applicable approved national annual calendar/recurring rules and historical
   applicability, including relevant universal-calendar updates to the beta base.
2. **Ascension** territorial/default applicability: national proclamation gives
   May 14 Thursday and May 17 Sunday **conditionally**, not one national default.
3. **Vietnamese Martyrs** national proper/external-solemnity law and default
   formulary, particularly **November 15, 2026**; authoritative decree/Commons.
4. Approved Vietnamese lectionary editions, authorization, full alternatives and
   ordering. Specifically request relevant title/approval and **Mồng Ba Tết option
   pages** if the owner has the book/PDF. Do not silently adopt the 2017 candidate.
5. Full national archive coverage, passage-content/celebration audits, missing
   text repair where justified, coordinated snapshots, rollback and release approval.

The latest pass made **51 research HTTP requests**, not 365 daily crawls. Hanoi's
public search returned 403; Dalat timed out; KTCG's historical date form returned
200 with `success=false`/access denial. No bypass/alternate endpoint was used.
Irrelevant Bing output and a Google interstitial were not accepted as evidence.
Three earlier Vatican November requests also returned 403; no verification was
claimed. Do not work around these failures with renamed URLs/proxies.

## Real parser fix and reviewed text repair

`2dd6fbe` fixes `internal/parser/parser.go`: the successful captured Vatican
2023-12-03 page put `Tin Mừng Mc 13,33-37` before a summary paragraph and the
liturgical header. The old parser required the reference inside that header.

New fallback checks **at most two preceding sibling paragraphs**, requires an
explicit Gospel label containing only a reference, matches its book to the
header, and stops at wrappers. No global first-reading/acclamation/distant-ref
search. `internal/parser/split_reference_test.go` has **11 cases**.

The successful captured response was replayed **offline through the existing
crawler worker/parser/cleanup and TSV converter**, not claimed as a new HTTP
request. A missing-only transaction inserted **Mc 13:33–37**, no existing row
updates/deletes, no network inside transaction. FTS indexed all five. No new
reading observation/date acceptance was inserted. No auto-ingestion feature.

- Text source: `https://www.vaticannews.va/vi/loi-chua-hang-ngay/2023/12/03.html`
- TSV SHA-256: `33b6a38ee433a13dff692fcd4cae27ce5a24154b0357b2197094ee35d8763a16`
- **Mc 13,24-32 is still missing.** Do not call this a full chapter repair.
- Temporary replay Go test was removed from `tools/crawler/`; its source is kept
  in the artifact directory. Only the actual bounded parser fix/tests were committed.

## Validation already completed

Latest code/data validation passed:

- `go test -tags "fts5" -count=1 ./...`
- `go test -tags "fts5" -race -count=1 ./...`
- All Go builds; Staticcheck with FTS5 tag.
- **18 calendar tests** (`npm test --prefix tools/calendar`).
- **24 frontend tests**, ESLint and optimized Next production build.
- Both repositories' actionlint, shell syntax, whitespace checks.
- **1,115 consumer/date checks**: API/CLI/frontend lookup for all 365 pilot dates
  plus two outside-year controls; 14 production-mode pages (all eight available
  dates plus six unavailable controls). Bodies, metadata, actual evidence and
  versions agree. This tests parity, not complete liturgical/edition certification.
- Repeat export/import/audit byte-identical; reapplying reviews creates no new
  history or fresh review timestamp. Actual ledger timestamps checked against DB.
- Warm Advent lookup: one hit, **zero HTTP**. Pending Tết and unsupported November
  15 fail honestly with **zero HTTP/candidates**. Fingerprints/TSV checked.
- Integrity/FKs, original reference/text preservation, unchanged observations and
  both shipped hashes verified.
- Default November 10–12 national-pilot monitor still correctly fails for all
  three unsupported definitions. Do not make it green by changing its window.
  An initial invocation missed Node's required server condition; that error log
  is separate. The correct invocation was rerun and its three-date failure checked:

```sh
cd /tmp/gospel-site-mass-fix.afwvYH
DB_PATH=/tmp/daily-bible-vietnam-content.KJ5EK8/frontend.db \
  READING_CALENDAR=vietnam \
  node --conditions=react-server scripts/check-readings.mjs
# Expected nonzero; default today + two-day window stays unchanged.
```

Staticcheck/actionlint binaries: `/tmp/daily-bible-review-tools.G4kXrj/`.
Local Go was 1.26.1; CI uses Go 1.25. No downgrade of GitHub action versions:
checkout v6, setup-go v6, upload-artifact v7 exist; original version concern was false.

## Current temporary artifacts and safe resumption

Latest directory: **`/tmp/daily-bible-vietnam-content.KJ5EK8/`**.
Pointer: `build/vietnam-content-workdir.txt`. Contains:

- `requests.json`, captures and text/link extracts, `vinh-long-pdf.bin` / `.txt`.
- `initial-pilot.db` (prior two approvals), **`audit.db`** (current eight),
  `manifest.json`, `before.json`, `references-reviewed.json`, `after.json`,
  `repeat.json`, `final.json`, individual review/citation logs.
- `pipeline/build/gospels.txt`, `.tsv`, replay harness/logs,
  `text-repair.json`, `source-integrity.json`, `local-passages.json`.
- Binaries `server`, `daily-bible`, `lectionarycrawler`, `tsv`.
- Independent `server.db`, `frontend.db`, `smoke.py`, `smoke-results.json`.
- Go/race/calendar/frontend validation logs, warm/gap/monitor logs, shipped hashes.

Temporary directories may disappear. Durable facts/URLs/times/hashes are in the
committed source ledger and reports, not the full source bytes or a DB patch.
No current temporary DB should be promoted as a release snapshot.

To inspect the current result **without network or new approvals**, first back
up the artifact DB to a new path and use that copy:

```sh
cd /opt/minh/test/daily-bible
mkdir -p build
sqlite3 -readonly /tmp/daily-bible-vietnam-content.KJ5EK8/audit.db \
  ".backup 'build/resume-national-audit.db'"
DB_PATH=build/resume-national-audit.db READING_CALENDAR=vietnam \
  go run -tags "fts5" ./tools/lectionarycrawler \
  -years=2026 -offline -audit-report=build/resume-national-audit.json
```

Fresh definitions-only reproduction (approves **zero** Gospels):

```sh
sqlite3 -readonly resources/bible.db ".backup 'build/new-national-pilot.db'"
node tools/calendar/vietnam-pilot.cjs 2026 > build/new-national-pilot.json
DB_PATH=build/new-national-pilot.db READING_CALENDAR=vietnam \
  go run -tags "fts5" ./tools/lectionarycrawler \
  -years=2026 -calendar-file=build/new-national-pilot.json \
  -offline -audit-report=build/new-national-pilot-audit.json
```

Do not overwrite existing audit copies inadvertently. Explicit review flags are
`-review-entry`, `-ref`, `-evidence`, `-evidence-date`, `-review-note`, with
`-replace-entry` for deliberate replacements; exact-date choices use `-accept-date`
and `-replace-date`. These are separate approvals, not report-import operations.
Never bulk-approve this inventory or use a date choice to bypass an unknown default.

Older checkpoints (historical, not current national coverage):

- `/tmp/daily-bible-vietnam-pilot.FEpcYL/`: original nine-definition/two-reading
  pilot, schema-3 validation, `final-audit.db`, 1,110-check parity.
- `/tmp/daily-bible-vietnam-policy.pZMWNv/`: six-source national dossier research.
- `/tmp/daily-bible-calendar-continue.h2FdOQ/`: **Roman General/schema-2 evaluation**,
  46 renderable dates, 11 reusable reviews + two date decisions, 204 parity checks.
  It had 2,493 unreviewed date rows / 1,041 unreviewed identities, three missing-text
  selections, 14 unsupported days. These are **not Vietnam coverage**; old manifests
  need regeneration/re-review, not relabeling. Its Mc 13,24-32 gap remains real.
- `/tmp/daily-bible-cache-execution.1OOFNU/`: earlier 34-date architecture checkpoint.
- `/tmp/daily-bible-calendar-research-continue/`: normative Lectionary PDF/text,
  HĐGMVN Martyrs material and Saigon calendar samples.

November 15 baseline conflict remains: legacy `Lc 9,23-26`, dated Vatican source
`Mt 25,14-30`. Source-only selection is quarantined. The Roman General year-A
Sunday-33 mapping supports Matthew, but does **not** invalidate a legitimate
Martyrs alternative or establish the national external-solemnity/default policy.

## Recommended next work / stopping boundary

1. Obtain/review the **missing authoritative national calendar and lectionary
   approval/edition/option documents**. Use the Ordo's CD 878/90 citation as a
   source lead, not a decree substitute. If not publicly accessible, request the
   specific approved material from the owner, not another jurisdiction choice.
2. Resolve the pending Tết default only with appropriate authority; preserve all
   legitimate alternatives and superseded decisions. Resolve Martyrs and Ascension
   scope/default applicability without guessing a diocese's practice.
3. Extend reviewed national definitions, generate a fresh Vietnam-scoped grouped
   queue, and recover reusable mappings/true exceptions. No blanket date crawling.
4. Repair justified missing text through the existing pipeline with passage-content
   review; keep missing status when source/edition is not established.
5. Expand historical/future coverage under applicable rules, audit celebration and
   allowed alternatives as well as refs/text, then prepare matching backend/frontend
   snapshots, diffs, rollback and coordinated policy/workflow activation.
6. Push/deploy only after substantive coverage/reviews and authorization. **Do not
   activate the sparse pilot or treat green tests/parity as liturgical approval.**

The last user request was only to update this handoff because context was nearly
full. It did not authorize deployment, change the agreed target/default or resolve
any remaining source/edition question.
