# Calendar research continuation — 2026-11-09

These findings support scoped implementation/review decisions, **not activation
of a production calendar**. The intended national/diocesan/parish policy and Mass
option defaults are still unconfirmed. Research used public sources; no shipped
verse or reading snapshot was changed.

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
