# Vietnam content/source continuation — 2026-11-10

## Result and limits

This continues the [initial pilot](vietnam-pilot-audit.md) on a consistent temporary
copy. **Six of the seven pending mappings were reviewed; eight dates now render.**
Mồng Ba Tết remains pending. The nine supported definitions and rules version
`vietnam-national-pilot-2026-0db4f1f3fdf6cd9a137a` are unchanged. No national
calendar was inferred from a diocesan book, and no new schema/provider work was
needed. A real source-format issue required one bounded parser fix.

| Stage | Renderable | Missing text | Unreviewed | Unsupported definitions |
| --- | ---: | ---: | ---: | ---: |
| Start | 2 | 0 | 7 | 356 |
| After six reference reviews | 7 | 1 | 1 | 356 |
| After reviewed five-row text repair | **8** | **0** | **1** | **356** |

These are **2026 pilot counts**, not full national coverage, archive recovery,
approved-edition certification or release assets. The runtime default remains
`vatican-vi`; both shipped databases and production are unchanged.

## Sources found, without promoting their scope

The machine-readable [review/source contract](vietnam-content-review.json) and
[capture fingerprints](vietnam-content-captures.json) retain actual URLs, request
times, source/reading dates, applicability, limitations and text reviews. They are
not manifests or automatic approvals. The earlier two reviews retain their
original evidence and timestamps.

### Official annual Ordo: diocesan, not national

The [Vĩnh Long publication](https://giaophanvinhlong.net/lich-cong-giao-giao-phan-vinh-long-2025-2026/)
links a [52-page PDF](https://giaophanvinhlong.net/wp-content/uploads/2025/12/38e8267e-b839-4990-9ec9-005af8333e3f.pdf):
*Tòa Giám Mục Vĩnh Long, Lịch Công Giáo Năm Phụng Vụ 2025–2026, Năm A*,
Nhà xuất bản Đồng Nai, 2025. The landing page is dated December 2, 2025; the
captured PDF metadata reports modification on January 3, 2026. Request hashes
identify the captured bytes, not an immutable December publication version.

It supplies useful named-Mass reference evidence, but **was not imported as
Vietnam's calendar**. It includes local anniversaries and adoration schedules.
Its page 10 prints the Ash transfer year as 2025, and page 36 prints
`Lc 9,11-28` for November 18. The first disagrees with the explicit national
2026 directive; the latter is a suspect weekday reference. Neither was silently
repaired or promoted into a rule.

Page 36 lists both Sunday 33 and Martyrs external solemnity on November 15, with
Martyrs readings chosen from the Commons; November 24 cites **CD 878/90** and
names the national patrons. This is a useful decree lead, **not the underlying
decree, its territorial/default applicability or an authorized national default
Gospel**. November 15/24 remain unsupported. The tail identifies the new year B
on November 29 but does not provide its complete Gospel references.

### Lectionary edition: a candidate, not an established contract

The [publisher catalogue](https://ktcgkpv.org/books?page=2) lists
*Sách Bài Đọc Trong Thánh Lễ (3 quyển)*, publication year **2017**, out of stock.
It does not provide the needed approval/frontmatter, full option ordering or
proof that the site's readings are this exact edition. The 2023 HĐGMVN interview
announcement concerns the **Missal**, not a promulgated lectionary option set.
Neither establishes the release-required edition contract.

A Hanoi public search request returned **403** and was not bypassed. The
ktcgkpv.org public date form returned HTTP 200 but explicitly denied access to
the requested historical day; no alternate endpoint was used to obtain it.
Dalat timed out; irrelevant Bing results and a Google interstitial were not
accepted as evidence. There were **51 research HTTP requests**, not a full-year
daily-reference crawl; HTTP success alone was not approval.

## Additional reviewed mappings

| Pilot date | Selected Mass | Gospel | Actual evidence date |
| --- | --- | --- | --- |
| 2026-02-18 | Ancestors/parents, Mồng Hai Tết | `Mt 15,1-6` | 2026-02-18 |
| 2026-04-05 | Easter, daytime/morning | `Ga 20,1-9` | 2026-04-05 |
| 2026-05-24 | Pentecost, year A, daytime | `Ga 20,19-23` | 2026-05-24 |
| 2026-06-07 | Corpus Christi, year A | `Ga 6,51-58` | Explicit June 7 assignment in annual Ordo |
| 2026-11-22 | Christ the King, year A | `Mt 25,31-46` | **2023-11-26** |
| 2026-11-29 | Advent I, year B | `Mc 13,33-37` | **2023-12-03** |

Date-to-Mass decisions still come from the earlier national directives, not the
Gospel publication or diocesan Ordo. These reviews are scoped to the supported
pilot entries, not other years/calendars, the Easter/Pentecost vigils, Easter's
evening option or complete Mass readings. In particular, the Tết publication's
Alleluia is not ingested and does not override Lent's actual start.

### Corpus Christi endpoint evidence stays explicit

The HĐGMVN publication prints `Ga 6,51-59`, while its tag and the Ordo's page 24
give `Ga 6,51-58`. The printed Gospel body ends with the living-forever sentence.
The local canonical verse 59 is a distinct synagogue-location sentence, absent
from that body. The approved mapping therefore uses **the Ordo as its primary
Gospel citation**, with actual body/tag corroboration. Its evidence date denotes
the explicit June 7 table assignment, not a fresh dated page.

The HĐGMVN endpoint remains in the ledger; it is not globally aliased to 58 or
silently called an error. An explicit citation refinement retained the earlier
review in history: **eight active mappings, nine review records**. This does not
adopt Vĩnh Long's calendar or certify the still-unestablished edition relationship.

### Why Mồng Ba Tết is still unreviewed

The dated publication contains full `Mt 25,14-30` and `Ga 5,16-20` options. The
Ordo lists Matthew alone. Those facts do **not** establish the approved national
lectionary's ordering or prescribe a national default. Neither the first web
option, a single diocesan listing nor an available legacy reference was promoted.
No date acceptance was used to bypass the uncertainty.

## Missing-text repair and the parser fix

`Mc 13,33-37` initially had **no local rows**. Its successfully captured
[Vatican page](https://www.vaticannews.va/vi/loi-chua-hang-ngay/2023/12/03.html)
places `Tin Mừng Mc 13,33-37` before one summary paragraph and an unnumbered
liturgical header. The parser previously required the reference in that header.

The fix looks back at **at most two sibling paragraphs**, requires an explicit
Gospel label containing only the reference, and checks its book against the
liturgical header. It never borrows an acclamation/first-reading reference,
crosses a wrapper or scans distant content. Eleven regression cases cover the
accepted shape and rejection boundaries; existing parser/crawler tests pass.

The captured successful response was processed **offline**, not claimed as a new
HTTP crawl, through the existing crawler worker/parser/cleanup and TSV converter.
After reviewing all five clauses/verse rows, a single transaction inserted only
the five missing keys. No transaction spanned a network request, and no existing
text was updated/deleted. FTS triggers indexed all five rows. No new date
observation or acceptance was created, and **`Mc 13,24-32` remains absent**.
The source URL/date and TSV hash are recorded in the ledger. No automatic text
ingestion, general repair command or new runtime maintenance dependency was added.

Selected existing/new passage sections and endings were examined, not merely
counted. Vietnamese wording/variant differences remain disclosed; this is **not
an exact-word match, a complete canonical chapter or approved-edition certification**.

## Validation and preservation

- Uncached and race Go suites, all builds, Staticcheck, workflow lint and shell
  syntax checks passed. **18 calendar tests** and **24 frontend tests**, lint and
  optimized production build passed.
- API, CLI and frontend lookup agreed for all **365 dates plus two outside-year
  controls**; **14 production-mode pages** covered all eight renderable dates and
  six unavailable controls: **1,115 consumer/date checks**. Bodies, labels, scope,
  versions, Mass/season metadata and actual evidence URLs/dates agreed, including
  older-year evidence and the annual-Ordo citation.
- Export/import/audit repetition was byte-identical. Reapplying identical reviews
  adds no history or fresh review time. Warm Advent reuse made **zero HTTP**;
  pending Tết and unsupported November 15 also failed with **zero HTTP/candidates**.
- The unchanged November 10–12 monitoring window still fails for genuinely
  unsupported national definitions. A preliminary invocation lacked the required
  Node server condition; that invocation error was retained separately and the
  correct command was rerun to verify the actual three-date failure.
- All **2,191 original date/reference pairs**, **2,917 original verse rows**, the
  two original pilot reviews and **15 observations** remain unchanged. The
  temporary copy adds only five verse rows, for 2,922 total. Integrity, foreign
  keys and new-row FTS checks passed.
- Both shipped SHA-256 values remain
  `e4cbcfe55f603c4e7cdb7c36bf8a3075469d72500fe490fab122c330b36ad1c7`.

Artifacts: `/tmp/daily-bible-vietnam-content.KJ5EK8/` contains captures/request
logs, the PDF/text, `initial-pilot.db`, `audit.db`, manifest, before/reviewed/after/
repeat reports, review history logs, missing-only TSV and replay harness, integrity
checks, validation logs, independent runtime copies and parity results. Temporary
artifacts may disappear; the committed ledger/fingerprints retain review facts,
not the full source texts or an automatically applied database patch.

## Remaining gate: specific authoritative material, not more framework work

To expand beyond the subset, obtain the applicable **national calendar/transfer
rules** and **approved Vietnamese lectionary title/approval and option pages**,
especially Mồng Ba Tết and the Martyrs Commons/default. The found diocesan Ordo
and catalogue do not fill this gap. Review historical applicability and relevant
universal-calendar updates before expanding the archive; do not extrapolate this
2026 pilot to another year. Then audit coverage and prepare coordinated release
snapshots/rollback assets. Eight dates are not adequate production/archive
coverage. Nothing from this continuation was pushed or deployed.

## Continuation — 2026-11-11: Tết default reviewed, rule evidence recorded

Result on a new temporary copy: **nine renderable, zero unreviewed, 356 unsupported**. Definitions and rules version `vietnam-national-pilot-2026-0db4f1f3fdf6cd9a137a` are unchanged. Still not national coverage, an edition certification or a release asset; both shipped databases are unchanged.

### Mồng Ba Tết: `Mt 25,14-30` as the first-listed option

Two further independent listings were captured. The [Hà Nội archdiocesan February 2026 calendar](https://www.tonggiaophanhanoi.org/thang-hai-2026/) and a [Hải Phòng reproduction](https://gphaiphong.org/chi-tiet/bai-doc-cac-thanh-le-dip-tet-nguyen-dan-at-ty) credited to Nhóm Phiên Dịch CGKPV each give `Mt 25,14-30` alone, as does Vĩnh Long; the dated HĐGMVN page prints it first with `Ga 5,16-20` under "Hoặc đọc". No located listing puts John first. The entry was explicitly reviewed at `2026-11-11T03:13:10Z` with the HĐGMVN page as evidence (evidence date 2026-02-19). `Ga 5,16-20` stays recorded as a legitimate alternative, and the earlier pending decision is kept in `resolved_pending`.

Limit: the approved lectionary's own title/approval and option pages were **still not inspected**. This applies the owner's first-listed policy to consistent published listings. If the book shows a different order, replace the mapping with `-replace-entry`.

The 17 existing verse rows were read end to end, not counted. Reapplying the review added no history or timestamp, and the repeat audit was byte-identical. Ten review records, nine active mappings, zero date reviews, 15 observations, 2,191 date rows and 2,922 verse rows; integrity and foreign-key checks pass. CLI and frontend resolver both return `Mt 25,14-30` for February 19 and nothing for November 15 or May 14.

### Rule evidence that did not become definitions

The pilot exporter accepts only HĐGMVN directive facts, and none of the following is one. Details and fingerprints are in `rule_findings` and the captures file.

| Question | What the captured sources say | Status |
| --- | --- | --- |
| November 15 | Hà Nội keeps Sunday XXXIII (`Mt 25,14-30`) and says the Martyrs **may** be solemnly celebrated "tùy theo nhu cầu mục vụ", citing **HĐGMVN 4-1991**. Vĩnh Long lists both. HĐGMVN's site published the Martyrs Mass for Sunday 33 in 2018, 2021 and 2025, the Sunday Mass in 2023. | Permitted, not prescribed. The site's default is an owner decision; the 1991 act itself was not obtained. |
| November 24 | Hà Nội prints rank "Trọng"; Vĩnh Long "Lễ trọng (CD 878/90)". | Consistent, but the decree was not obtained. Unsupported. |
| Martyrs Gospel | Hà Nội (twice), Hải Phòng and Huế list `Lc 9,23-26` first, then `Ga 17,11b-19`. | Order consistent; nothing approved because no definition is supported. |
| Ascension | Hà Nội: Thursday May 14, obligatory. Vĩnh Long and Phan Thiết: Sunday May 17. Bắc Ninh: either is allowed. | Real diocesan variation. No national default exists to find; owner decision. |
| Epiphany | Hà Nội and Vĩnh Long: Sunday January 4. | No primary national 2026 fact captured. Unsupported. |

Seventeen captured requests all returned 200. A Đà Lạt page on the canonization legal texts returned **403** and was not bypassed; a Qui Nhơn Ordo PDF URL now serves an unrelated page. Artifacts: `build/vietnam-rules-2026-11-11/` (ignored): captures, request log, `audit.db`, before/after/repeat reports and review logs.

## Baseline continuation — 2026-11-11 (later the same day)

The owner then decided four things: **Sunday XXXIII on November 15**, **Ascension on Sunday**, **agreeing official diocesan calendars count as evidence**, and **the General Roman calendar is the baseline for dates with no known Vietnamese rule**. These are site editorial choices, not ecclesiastical acts. The section above describes the state before them.

Result on a new temporary copy, rules version `vietnam-national-pilot-2026-65af9d5b44eebbca1149`:

| Status | Dates |
| --- | ---: |
| Renderable | **233** |
| Reviewed, verse text missing | 13 |
| Supported definition, Gospel unreviewed | 114 |
| Unsupported definition | 5 |

Still 2026 only, still a temporary copy. Both shipped databases are unchanged and nothing is activated.

### Definitions

`vietnam-pilot.cjs` now builds on the pinned provider with Epiphany, Ascension and Corpus Christi on Sunday. Thirteen dates carry explicit decisions: the nine earlier national ones plus Epiphany January 4, Ascension May 17, Sunday XXXIII November 15 and the Martyrs solemnity November 24. The four new ones cite two official calendars of different dioceses; the exporter rejects a diocesan decision with fewer. Every other supported date is labelled as owner-approved baseline with the provider as its source, never as a national directive.

Unsupported: Holy Thursday and Holy Saturday (no generic daytime Mass), February 17 (Hà Nội assigns the Tết Mass, Vĩnh Long only permits it), October 4 (Rosary external solemnity in Hà Nội) and October 18 (Mission Sunday).

### Gospel reviews

All earlier approvals were invalidated by the new version, as designed. 246 explicit entry reviews were made, each with its own evidence URL, date and note; the full list with actual review times is in [`vietnam-baseline-reviews.json`](vietnam-baseline-reviews.json).

- **14 individual**: the nine earlier entries re-reviewed unchanged, plus `Mt 2,1-12` (Epiphany), `Ga 15,9-17` (Saint Matthias, May 14), `Mt 28,16-20` (Ascension), `Mt 25,14-30` (November 15) and `Lc 9,23-26` (November 24).
- **232 by a two-calendar rule**, applied by script: the Hà Nội calendar's Gospel for the date must also appear inside the Vĩnh Long Ordo's entry for that same date. A typo or a different celebration in either calendar fails the match and leaves the date unreviewed. This is agreement between two independent official calendars, not a majority vote or a crawl result, but it is machine matching: the approved lectionary was not inspected and no wording was certified.

One agreeing pair was withheld: both calendars print `Mt 10,6-13` for Saint Barnabas (June 11) where the lectionary table gives `Mt 10,7-13`.

The reviews also expose the legacy store: on 34 reviewed dates the old stored reference differs, almost always a saint's Gospel filed on the wrong day (for example Saint Lawrence's `Ga 12,24-26` on August 8 and 12, not August 10).

### What is left

- **114 unreviewed**: all of July (the Hà Nội page returned HTTP 520 three times) and December (not yet published for the new year) account for 62. The other 52 are dates where the two calendars differ, one has a typo, or the parser found no single reading line.
- **13 missing text**: `Mt 1,16.18-21.24a`, `Ga 3,7b-15`, `Ga 14,27-31a`, `Ga 15,26-16,4a`, `Ga 16,20-23a`, `Ga 17,1-11a`, `Ga 17,11b-19`, `Mc 12,28b-34`, `Mt 18,21-19,1`, `Mt 20,1-16a`, `Lc 6,20-26`, `Lc 7,31-35`, `Lc 9,43b-45`. They need the verse pipeline, not more reviews.

### Validation

Go tests, race suite and Staticcheck pass; 22 calendar tests and 24 frontend tests pass. CLI and frontend resolver agree with the ledger on all 367 checked dates (the year plus one day either side). Reapplying all 246 reviews added no history and the repeat audit is byte-identical. Integrity and foreign-key checks pass; 2,191 legacy date rows, 2,922 verse rows and 15 observations are unchanged. Lint, the production build and page-level smoke tests were not rerun; no frontend or Go runtime code changed.

Artifacts: `build/vietnam-baseline-2026-11-11/` (ignored): `audit.db` (SHA-256 `7834c4d5b0886f64a94fc8e3336f90b8ae439a01dd7ac5d3d9ae51f82bb92175`), manifest, reports, review plan and log, captures and the matching scripts. 27 captured requests this day, no daily crawl.

## Ordinary-day decision — 2026-11-11 (final state of the day)

The owner then chose the **ordinary day** for February 17, October 4 and October 18; the Tết, Rosary and Mission Masses stay recorded alternatives. Those three dates moved from unsupported to baseline, which changed the rules version to `vietnam-national-pilot-2026-803f03d3d1f36e12fe27` and required every entry to be reviewed again. The section above is the state before this.

| Status | Dates |
| --- | ---: |
| Renderable | **237** |
| Reviewed, verse text missing | 13 |
| Supported definition, Gospel unreviewed | 113 |
| Unsupported (Holy Thursday, Holy Saturday) | 2 |

250 explicit reviews: 232 by the two-calendar rule and 18 individual. The four new individual ones are `Mc 8,14-21` (February 17, Vĩnh Long), `Mt 21,33-43` (October 4) and `Mt 22,15-21` (October 18), both listed by Hà Nội and Vĩnh Long for the Sunday, and `Mt 10,7-13` for Saint Barnabas on June 11 from the dated Vatican News page; the two calendars' `Mt 10,6-13` stays recorded as a discrepancy.

The same validation was repeated and passes: Go tests, race suite, Staticcheck, 22 calendar tests, CLI/frontend parity on 367 dates, idempotent re-review, integrity checks, unchanged shipped hashes. The missing-text list and the July/December gap are unchanged. Artifacts: `build/vietnam-baseline-2026-11-11b/` (ignored), `audit.db` SHA-256 `0a1cfd281009004aada178582230998971845bb099134d5c61c624d22e76a204`.

## Text repair and gap review — 2026-11-11 (current state)

Same rules version (`…803f03d3d1f36e12fe27`); no definition changed, so earlier reviews stand. The sections above are earlier states of the same day.

| Status | Dates |
| --- | ---: |
| Renderable | **327** |
| Reviewed, verse text missing | 1 |
| Supported definition, Gospel unreviewed | 35 |
| Unsupported (Holy Thursday, Holy Saturday) | 2 |

### Importer bugs found and fixed

The 13 missing-text passages were not missing downloads. The verse importer mishandled three page shapes, and the same bugs had already damaged stored rows:

1. **Part markers** such as `<sup>24a</sup>` were not verse markers, so `31a` was fused into verse 30 and no `31a` row existed.
2. **Chapter changes** inside a passage (`<b>19</b> <sup>1</sup>`) left the chapter number glued to the previous verse and filed the next verses under the old chapter.
3. **Unnumbered paragraphs** (quoted poetry, continued sentences) ended extraction, cutting `Lc 7,31-35` after verse 32.

Fixes: `tools/crawler` accepts suffixed markers and emits a `[[chapter:N]]` line; `tools/tsv` follows it only for the directly following chapter; `internal/parser` keeps one unnumbered paragraph when a numbered one follows it. A trailing unnumbered paragraph is still not taken. Regression tests cover all three and their rejection cases.

### Verse repair on the temporary copy

Sixteen captured dated Vatican News pages were replayed offline through the fixed crawler and TSV converter. [`vietnam-text-repair.json`](vietnam-text-repair.json) lists the result: **21 rows inserted** and **9 existing rows replaced**. Verse rows go from 2,922 to 2,943; FTS matches.

The nine replacements **depart from the earlier missing-only rule and need owner confirmation before any promotion**. Each replaced row carried importer damage, not a wording choice: `Ga 14,30`, `Ga 16,22`, `Ga 17,10`, `Mt 20,15` and `Ga 5,2` contained the following part's text; `Ga 15,27` and `Mt 10,42` ended in a stray chapter number; `Lc 6,25` and `Lc 7,32` were cut short. Inserting the missing part without correcting them would print sentences twice. Five rows that differ only in wording or spacing were left alone. Old and new hashes are in the ledger.

Still missing: `Ga 19,25-34` (May 25). The Vatican page prints `Ga 19,25-27`, so no reviewed capture covers verses 28-34.

### Gap review

Hà Nội's July page returned HTTP 520 a fifth time. For the 82 non-December dates still unreviewed, the dated Vatican News page was captured (one request each, 97 Vatican captures in total, fingerprints in [`vietnam-vatican-captures.json`](vietnam-vatican-captures.json)).

- **71 by a fallback rule**: the Vatican page's Gospel must equal the reference Vĩnh Long or Hà Nội lists for that date. All of July is here (Vatican plus Vĩnh Long).
- **7 by hand**, each with its discrepancy written into the note: `Ga 5,1-3a.5-16` (March 17), `Ga 14,1-6` (May 1, single-calendar evidence), `Ga 19,25-34` (May 25, the two calendars against Vatican), `Mt 10,26-33` (June 21), `Mc 6,17-29` (August 29), `Ga 19,25-27` (September 15), `Lc 19,11-28` (November 18). On May 1 and November 18 Vatican News follows an optional memorial, which the owner's policy does not select.

Three memorials (August 22, October 7, November 21) took the proper Gospel that Vatican and Vĩnh Long share; Hà Nội lists the weekday Gospel there. That is a real choice between permitted readings, recorded in the note.

Totals: 328 explicit reviews (232 two-calendar, 71 Vatican fallback, 25 individual). On 57 reviewed dates the legacy stored reference differs.

Unreviewed: December except Advent I's Sunday (31 dates; neither calendar has published the new year's readings), Palm Sunday and Good Friday (no usable single Gospel line), All Souls (several permitted sets) and Saint Andrew on November 30.

### Validation

Go tests, race suite, `go vet` and Staticcheck pass; 23 calendar tests and 24 frontend tests pass. CLI and frontend agree on **reference and passage text** for all 367 checked dates. Reapplying all 328 reviews added no history; the repeat audit is byte-identical. Integrity and foreign-key checks pass; 2,191 legacy date rows and 15 observations unchanged; shipped hashes unchanged. Frontend lint, production build and page smoke tests were not rerun.

Artifacts: `build/vietnam-baseline-2026-11-11c/` (ignored), `audit.db` SHA-256 `6e484f46f9c41902c663e285bf3e99e2d192395d06091ac3f650b034c39e6465`, with both replay pipelines and all captures.
