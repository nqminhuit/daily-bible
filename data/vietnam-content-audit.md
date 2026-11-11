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
