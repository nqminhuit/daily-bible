# Calendar research continuation — 2026-11-09

These findings support scoped implementation/review decisions, **not activation
of a production calendar**. The initial November 9 research preceded the owner's
November 10 confirmation of the Vietnam national/daytime target; see the
[continuation below](#confirmed-target-and-primary-national-guidance--2026-11-10).
Research used public sources; no shipped verse or reading snapshot was changed.

## Annual weekday Gospel reuse

The [General Introduction to the Lectionary, §69](https://www.liturgyoffice.org.uk/Resources/Rites/Lectionary.pdf)
explicitly provides a single annual cycle for weekday Gospels; the two-year I/II
cycle concerns the first reading. Consequently:

- A selected default weekday can share its **Gospel-only** entry across I/II and
  Sunday A/B/C cycles. This is not permission to reuse its complete Mass readings.
- Sunday entries still distinguish A/B/C. Proper observances, memorial choices,
  and Mass variants do not silently become temporal weekdays.
- Gospel entry schema **2** removes I/II from weekday keys. Exported manifests
  declare `gospel_entry_schema: 2`; imports reject missing/old/unknown schemas
  before changing calendar records. The adapter/rules hash also changes.
- Existing cached snapshots/review history are not rewritten in place. Regenerate
  definitions and explicitly re-review mappings under the new version on a copy;
  do not relabel an old review as a new approval.

The same document's Ordinary Time Sunday tables I/II/III give Sunday 33:

| Roman General temporal Sunday | Standard full Gospel |
| --- | --- |
| Year A | `Mt 25,14-30` |
| Year B | `Mc 13,24-32` |
| Year C | `Lc 21,5-19` |

These support three explicitly **Roman General** demonstration mappings. They
have a real, undated normative evidence URL, not a fabricated dated Vatican page.
They do not determine whether a local Mass instead celebrates the Vietnamese
Martyrs or which permitted Gospel is selected for that celebration.

## November 15: different formularies are a real possibility

The Vietnamese bishops' public pages list these Vietnamese Martyrs alternatives:
`Mt 10,17-22`, `Lc 9,23-26`, and `Ga 17,11b-19`:

- [“Bỏ mình và mất mình”](https://hdgmvietnam.com/chi-tiet/bo-minh-va-mat-minh-14-11-2021-chua-nhat-33-tn-nam-b-cac-thanh-tu-dao-viet-nam--44043), currently headed November 17, 2024, Sunday 33 B / Vietnamese Martyrs.
- [“Không thuộc về thế gian”](https://hdgmvietnam.com/chi-tiet/khong-thuoc-ve-the-gian-15-11-2020-chua-nhat-33-tn-cac-thanh-tu-dao-viet-nam--40916), currently headed November 24, 2025, Vietnamese Martyrs.

**URL slugs retain older dates while the page content has been updated.** They
must not be ingested as immutable exact-date evidence merely by parsing the slug.
These are public liturgical publications, not a complete historical calendar
policy or a decree establishing every parish's default option.

[Saigon's calendar lookup for November 15, 2026](https://tgpsaigon.net/Article/LiturgyCalendar?LiturgyDateTime=2026-11-15)
links a Sunday 33 A / solemn celebration of the Vietnamese Martyrs item with
`Mt 10,17-22`. This is meaningful archdiocesan publication evidence, but not by
itself a national transfer decree or approval of this site's intended calendar.

Thus the stored `Lc 9,23-26` is not intrinsically an invalid Martyrs Gospel, while
`Mt 25,14-30` is established for the Roman General temporal Sunday. Equality with
a dated Vatican reference cannot establish local celebration identity. The
**intended production selection remains unresolved**; the temporary Roman General
selection does not overwrite the legacy value or approve another namespace.

## Unsafe Vietnam provider is blocked

The pinned `@romcal/calendar.vietnam@3.0.0-dev.144` data contains an override of
`andrew_dung_lac_priest_and_companions_martyrs` to **November 13**, precedence
`PROPER_FEAST_8F`. Reproduction with the matching Romcal engine for 2026 selects:

| Date | Plugin output |
| --- | --- |
| November 13 | Vietnamese Martyrs, `FEAST` |
| November 15 | Ordinary Sunday 33 |
| November 24 | Ordinary weekday |

The fixed-date/rank discrepancy and external-solemnity policy are not settled by
this secondary beta provider. Leaving it enabled could let a valid weekday
mapping be reused on an incorrectly resolved local proper date—the original bug
in a new form. **Do not silently patch November 13/24 or invent a national rule.**

Safety measures:

- `tools/calendar/export.cjs vietnam ...` fails explicitly; no automatic fallback
  to Roman General. The pinned plugin remains only as reproducible regression
  evidence in the provider tests.
- The importer and reusable-entry reviewer reject `vietnam` rules versions with
  the automatic `romcal-` prefix. Go/frontend readers also refuse derived entries
  in existing caches under that unreviewed provider; audits report
  `unsupported_calendar_rules` instead of treating them as crawlable gaps.
- The `vietnam` namespace remains implemented for **reviewed custom national
  definitions with their own rules version**. A named diocesan/parish policy
  requires a distinct implemented namespace/adapter; it must not be silently
  relabeled national. Renaming the rejected export is not review. Approval requires
  authoritative dates/ranks, transfers and permitted/default options, followed by
  coverage and source audits.
- Independent, valid exact-date decisions remain separate evidence; blocking an
  unreliable derived calendar does not erase their history or automatically
  invalidate an explicitly reviewed date choice.

## Research and text limitations

The continuation's three attempted dated Vatican checks (2026-11-15,
2024-11-17, 2025-11-16) returned HTTP 403. No response was accepted as source
evidence, no alternate date was substituted, and no access-control workaround was
used. The earlier successful 10-page source audit remains a separate checkpoint,
not a claim of fresh verification for these three pages.

`Mc 13,24-32` has **no local verse rows** in the shipped snapshot. Its reviewed
reference can be reused but the corresponding three year-B dates in 2021–2027
remain unavailable with `missing_text`; another reference crawl cannot repair
this. Use the reviewed verse-crawler/TSV import pipeline on a copy when repairing
text. The A/C mappings pass structural coverage, not a new word-level comparison.

Research artifacts (temporary, not release assets):

```text
/tmp/daily-bible-calendar-research-continue/
/tmp/daily-bible-calendar-continue.h2FdOQ/
```

The first contains the downloaded normative PDF/text and public Church pages;
the second contains version-2 manifests, the rejected Vietnam-provider example,
read-only inventory/review queues, explicit demonstration reviews and validation.
See [the audit checkpoints](reading-cache-audit.md) and
[the operating contract](calendar-policy.md) for release gates.

## Confirmed target and primary national guidance — 2026-11-10

The owner agreed to **Vietnam's approved national calendar, daytime Mass, no
assumed diocesan/parish overlay**. Follow an authoritative Gospel assignment;
otherwise use the first-listed full authorized Gospel in the approved lectionary.
This is an editorial target/default decision, not ecclesiastical promulgation,
approval of the beta provider, or permission to publish a snapshot.

The [machine-readable source dossier](vietnam-national-sources.json) captures six
public HĐGMVN/Ủy ban Phụng tự sources, their actual URLs, document/page dates,
request timestamps and response/text SHA-256 fingerprints. Request times are not
review timestamps; page dates are not dates of Gospel evidence. Full responses
remain research artifacts rather than runtime data. Dynamic page bytes can change;
a fingerprint identifies the captured response, not an immutable publication.
This dossier deliberately has no executable calendar schema, complete date set,
Gospel approvals or production rules version.

### Tết and the 2026 Ash Mass are not ordinary calendar overrides

The [official 2026 Ash directive](https://hdgmvietnam.com/chi-tiet/uy-ban-phung-tu-thong-bao-ve-viec-cu-hanh-le-tro-nam-2026)
is signed July 14, 2025; its page includes an English translation added February
4, 2026. It cites Holy See indult **2407/98L, November 26, 1998**, and explicitly
assigns the following Masses **in Vietnam**:

| Civil date | Assigned Mass |
| --- | --- |
| 2026-02-18 | Mồng Hai Tết — ancestors and parents |
| 2026-02-19 | Mồng Ba Tết — sanctification of human labor |
| 2026-02-20 | Ash Wednesday Mass, ashes, fasting and abstinence |

This is a **Mass transfer**, not just a fasting dispensation. The Hours remain
Ash Wednesday on February 18. The committee's
[2026 proclamation](https://hdgmvietnam.com/chi-tiet/loi-thong-bao-long-trong-ve-le-phuc-sinh-va-nhung-ngay-le-chinh-trong-nam-phung-vu-2026-duoc-cong-bo-vao-ngay-le-chua-hien-linh)
explicitly says **Lent still begins February 18**, including omission of Alleluia.
Do not move the entire season, recalculate Easter, or map Mass and Hours through
one undifferentiated observance override. The national adapter must distinguish
these concepts; the present Roman General exporter is not that adapter.

The [2024 Tết directive](https://hdgmvietnam.com/chi-tiet/uy-ban-phung-tu-huong-dan-cu-hanh-phung-vu-tet-nguyen-dan-giap-thin-2024-54309)
assigns the New Year Mass throughout February 10 and the ancestors/parents Mass
throughout Sunday February 11, including evening instructions. Thus national Tết
Masses cannot simply be appended as optional saints while retaining every
underlying Sunday. Evening handling remains outside the implemented `day` option.

The [committee-endorsed 2021 explanation](https://hdgmvietnam.com/chi-tiet/hoi-dap-ve-thanh-le-trong-dip-tet-nguyen-dan-41422)
identifies distinct Tết formularies and the national supplication-day framework.
It also says a specific announcement accompanies an Ash transfer. It is an
explanation, not a newly issued decree; do not extrapolate the 2026 assignment
across all historical/future lunar-calendar collisions without reviewing the
applicable norms/notices. Its quoted excerpts are not a complete approved Gospel
option set or a basis for inserting partial passage text.

### Other confirmed dates and an unresolved transfer choice

- The [2026 All Saints directive](https://hdgmvietnam.com/chi-tiet/uy-ban-phung-tu-huong-dan-cu-hanh-phung-vu-le-cac-thanh-nam-2026),
  signed October 23, confirms All Saints on November 1 in place of Ordinary
  Sunday 31; October 31 evening also belongs to All Saints. No generic daytime
  reading for October 31 is changed by this research.
- The [Christmas 2021–2022 guidance](https://hdgmvietnam.com/chi-tiet/uy-ban-phung-tu-giai-dap-ve-phung-vu-mua-giang-sinh-nam-nay-2021-2022--44276)
  explicitly identifies Epiphany on Sunday January 2, 2022. This is an established
  dated example, not by itself a complete national transfer law for every year.
- The 2026 proclamation lists Easter April 5, Pentecost May 24, Corpus Christi
  June 7, Christ the King November 22 and Advent I November 29. Its Ascension
  text gives **both May 14 (Thursday) and May 17 (Sunday), conditionally**. It does
  not establish which territorial/default schedule this site should apply.
  Do not silently select Sunday, Thursday, or a diocesan practice as universally
  national. Corpus Christi's 2026 date likewise does not alone document every
  historical year's transfer rule.

These sources do **not** settle November 15's national external-solemnity policy,
its authorized Gospel/default, all national propers, or the approved lectionary
editions. The target scope is now confirmed; those source/rule reviews remain
substantive work, not another request to choose the target calendar.

`tools/calendar/vietnam-policy.test.cjs` provides **specification/guard tests**:
seven sourced Mass cases (six differ from the configured Roman General profile),
Mass versus season/Hours, conditional Ascension, evidence integrity, and continued
provider blocking despite target approval. These tests do not implement or
certify a national resolver. The full calendar test command now runs five
provider tests plus five dossier/specification guards, entirely offline.

### Subsequent implementation checkpoint

The [explicit 2026 pilot](vietnam-pilot-audit.md) now implements nine source-backed
definitions and marks 356 dates unsupported, rather than enabling a generic
Vietnam provider. Schema 3 separates assigned Mass from underlying observance
and seasons, with per-day directive evidence in migration 005. The earlier
schema-2 discussion and dossier-only tests above are historical checkpoints;
current imports require schema 3. No full Hours calendar is implemented.

Two additional HĐGMVN publications provide specific Gospel evidence:
[transferred Ash Mass](https://hdgmvietnam.com/chi-tiet/cha-hien-dien-noi-kin-an-14-2-2018-thu-tu-le-tro--31941)
and [All Saints](https://hdgmvietnam.com/chi-tiet/phuc-thay-ai-01-11-2020-chua-nhat-31-le-cac-thanh-nam-nu--40850).
Their current reading headings are February 20 and November 1, 2026, whereas
publication dates are February 19 and October 31. Old slugs are retained exactly;
no immutable dated URL is fabricated. The printed single Gospel references agree
with the two separately reviewed calendar decisions. The
[evidence inventory](vietnam-pilot-reading-evidence.json) records actual response
fingerprints and the different-wording/pericope-text limitations, not an automatic
approval or a complete approved-edition contract.

The full national calendar, remaining option sets and November 15 are still
unresolved. That two-reading checkpoint was not national archive coverage.

New temporary artifacts: `/tmp/daily-bible-vietnam-policy.pZMWNv/` (six recorded
sources, three supplementary publications, ten search responses, and the actual
19-request status/timestamp log). All requests succeeded; none approved a dated
Gospel or changed either shipped SQLite snapshot. Continue with reviewed national
rule/edition/applicability definitions before cache promotion or rollout.

### Content/source continuation — 2026-11-10

The [next source review](vietnam-content-audit.md) located an official 52-page
Vĩnh Long 2025–2026 Ordo and the publisher's 2017 three-volume lectionary catalogue.
Neither establishes the missing full national calendar/approved-edition contract.
The Ordo's CD 878/90 citation is a useful Martyrs decree lead, not the decree or
an established default Gospel. Its local anniversaries/choices were not imported.

Six additional pilot mappings were explicitly reviewed. Corpus Christi retains
its printed `Ga 6,51-59` discrepancy; the primary Ordo citation supports canonical
`Ga 6,51-58`. Christ-the-King and Advent-I evidence dates remain in 2023.
Mồng Ba Tết's two full published options remain pending until the authorized
default order is established. The successful dated Vatican capture for Advent I
was processed through the crawler/TSV pipeline after a bounded split-heading
parser fix; only five missing Mark rows were inserted on a temporary copy.

Current pilot result: **eight renderable, one unreviewed, 356 unsupported**.
The [review/source contract](vietnam-content-review.json) and
[capture fingerprints](vietnam-content-captures.json) preserve scope, source
limitations, dates and content fingerprints. Temporary evidence and audit copies:
`/tmp/daily-bible-vietnam-content.KJ5EK8/`. No shipped database, production policy
or deployment changed.

### Rule-evidence continuation — 2026-11-11

Mồng Ba Tết was reviewed as `Mt 25,14-30`, first-listed in every located listing; the pilot is now nine renderable, zero unreviewed, 356 unsupported. Hà Nội's official calendar cites HĐGMVN 4-1991 for an **optional** Martyrs celebration on Sunday XXXIII, and Ascension is confirmed to differ by diocese (Hà Nội Thursday; Vĩnh Long and Phan Thiết Sunday). Neither has a national default to discover: both need an owner editorial rule. See [the continuation](vietnam-content-audit.md#continuation--2026-11-11-tết-default-reviewed-rule-evidence-recorded).

### Owner-approved baseline — 2026-11-11

The owner approved the General Roman baseline, Sunday Ascension, Sunday XXXIII on November 15 and two-diocese evidence. The pilot now supports 360 of 365 definitions and resolves 233 dates on a temporary copy. See [the baseline continuation](vietnam-content-audit.md#baseline-continuation--2026-11-11-later-the-same-day).
