'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const review = require('../../data/vietnam-content-review.json');
const captures = require('../../data/vietnam-content-captures.json');
const rules = require('../../data/vietnam-pilot-2026.json');
const baseline = require('../../data/vietnam-baseline-reviews.json');
const vatican = require('../../data/vietnam-vatican-captures.json');
const repair = require('../../data/vietnam-text-repair.json');

test('content reviews retain actual captures and do not become automatic calendar approvals', () => {
  assert.equal(review.auto_approve, false);
  assert.equal(review.activation, 'temporary-evaluation-only-not-production-approved');
  assert.equal(review.reviews.length, 7);
  assert.equal(new Set(review.reviews.map(r => r.date)).size, 7);
  assert.equal(review.days, undefined);
  assert.equal(review.gospel_entry_schema, undefined);
  for (const r of review.reviews) {
    assert.ok(rules.decisions.some(d => d.date === r.date && d.identity === r.mass));
    assert.ok(captures[r.source]);
    assert.ok(Date.parse(r.reviewed_at) >= Date.parse(captures[r.source].requested_at));
    assert.match(r.evidence_date, /^\d{4}-\d{2}-\d{2}$/);
    assert.ok(r.applicability && r.text_review);
  }
  for (const capture of Object.values(captures)) {
    assert.equal(new URL(capture.url).protocol, 'https:');
    assert.ok(Number.isFinite(Date.parse(capture.requested_at)));
    assert.match(capture.sha256, /^[a-f0-9]{64}$/);
    assert.match(capture.text_sha256 || capture.extracted_text_sha256, /^[a-f0-9]{64}$/);
  }
});

test('the diocesan Ordo and catalogue are not relabeled as a national baseline or approved edition', () => {
  const c = review.source_contract;
  assert.equal(c.annual_ordo.national_import_approved, false);
  assert.equal(c.annual_ordo.scope, 'diocese-of-vinh-long-not-the-national-calendar');
  assert.equal(c.annual_ordo.pdf_pages, 52);
  assert.match(c.lectionary_candidate.status, /approval-frontmatter-order-and-edition-relationship-unverified/);
  assert.match(c.status, /incomplete/);
  assert.ok(c.access_limits.some(s => s.includes('403')));
  assert.ok(c.access_limits.some(s => s.includes('success=false')));
});

test('the Tet default is an explicit first-listed review with its alternative and pending history retained', () => {
  assert.deepEqual(review.pending, []);
  const earlier = review.resolved_pending[0];
  assert.equal(earlier.status, 'default-option-order-unestablished');
  assert.deepEqual(earlier.candidate_refs_in_page_order, ['Mt 25,14-30', 'Ga 5,16-20']);
  const tet = review.reviews.find(r => r.date === earlier.date);
  assert.equal(tet.ref, earlier.candidate_refs_in_page_order[0]);
  assert.deepEqual(tet.alternative_refs_retained, ['Ga 5,16-20']);
  assert.match(tet.option_order_basis, /approved-lectionary-pages-not-inspected/);
  assert.equal(earlier.resolved_by_review_at, tet.reviewed_at);
  for (const id of tet.corroborating_sources) assert.ok(captures[id]);
  const corpus = review.reviews.find(r => r.date === '2026-06-07');
  assert.equal(corpus.ref, 'Ga 6,51-58');
  assert.equal(corpus.printed_ref, 'Ga 6,51-59');
  assert.equal(corpus.page_tag_ref, corpus.ref);
  assert.equal(corpus.source, 'vinh-long-pdf');
  assert.equal(corpus.source_printed_page, 24);
  assert.equal(corpus.corroborating_source, 'corpus-2026');
  assert.equal(corpus.evidence_date_kind, 'explicit-June-7-assignment-in-annual-Ordo-not-a-fresh-daily-page');
  assert.equal(review.superseded_review_retained.ref, corpus.ref);
  assert.equal(review.superseded_review_retained.source, 'corpus-2026');
  assert.ok(Date.parse(review.superseded_review_retained.reviewed_at) < Date.parse(corpus.reviewed_at));
  assert.deepEqual(review.earlier_review_times_retained.map(r => r.reviewed_at), ['2026-11-10T06:48:31Z', '2026-11-10T06:48:31Z']);
  for (const [date, evidence] of [['2026-11-22','2023-11-26'], ['2026-11-29','2023-12-03']]) {
    assert.equal(review.reviews.find(r => r.date === date).evidence_date, evidence);
  }
});

test('text repair covers only five explicitly reviewed rows, not a chapter or another missing passage', () => {
  const r = review.text_repair;
  assert.equal(r.scope, 'temporary-copy-only');
  assert.deepEqual(r.inserted_verse_numbers, [33,34,35,36,37]);
  assert.equal(r.updated_or_deleted_existing_rows, 0);
  assert.equal(r.evidence_date, '2023-12-03');
  assert.equal(captures[r.source].url, 'https://www.vaticannews.va/vi/loi-chua-hang-ngay/2023/12/03.html');
  assert.match(r.tsv_sha256, /^[a-f0-9]{64}$/);
  assert.match(r.remaining_missing_range, /24-32 remains absent/);
});

test('rule findings are captured evidence and the later owner decisions are explicit', () => {
  const f = review.rule_findings;
  assert.match(f.status, /acted-on-by-owner-decisions/);
  assert.match(review.rules_version_note, /superseded.*re-reviewed/);
  assert.notEqual(baseline.rules_version, review.rules_version);
  for (const date of ['2026-01-04', '2026-05-17', '2026-11-15', '2026-11-24']) {
    assert.equal(rules.decisions.find(d => d.date === date).evidence, 'diocesan');
  }
  assert.deepEqual(rules.unsupported, {});
  assert.ok(rules.baseline_notes['2026-02-17']);
  assert.deepEqual(f.martyrs_proper_gospels.refs_in_listed_order, ['Lc 9,23-26', 'Ga 17,11b-19']);
  for (const finding of Object.values(f).filter(v => v.sources)) {
    for (const id of finding.sources) assert.ok(captures[id], id);
  }
});

test('baseline review ledger is a record of explicit temporary-copy reviews, not an importable approval', () => {
  assert.equal(baseline.auto_approve, false);
  assert.match(baseline.purpose, /not-an-importable-approval/);
  assert.equal(baseline.days, undefined);
  const dates = new Set();
  const earlier = JSON.stringify(require('../../data/vietnam-pilot-reading-evidence.json'));
  const captured = new Set(Object.values(captures).map(c => c.url));
  const vaticanDates = new Set(vatican.captures.map(c => c[0]));
  const vaticanDate = url => (url.match(/^https:\/\/www\.vaticannews\.va\/vi\/loi-chua-hang-ngay\/(\d{4})\/(\d{2})\/(\d{2})\.html$/) || []).slice(1).join('-');
  for (const url of baseline.evidence_urls) assert.ok(captured.has(url) || earlier.includes(JSON.stringify(url)) || vaticanDates.has(vaticanDate(url)), url);
  for (const [date, ref, kind, url, evidenceDate, reviewedAt, status] of baseline.reviews) {
    assert.match(date, /^2026-\d{2}-\d{2}$/);
    assert.ok(!dates.has(date));
    dates.add(date);
    assert.match(ref, /^(Mt|Mc|Lc|Ga) \d/);
    assert.ok(['individual', 'agreement', 'vatican-agreement'].includes(kind));
    if (kind === 'vatican-agreement') assert.equal(vaticanDate(baseline.evidence_urls[url]), date);
    assert.ok(baseline.evidence_urls[url]);
    if (kind !== 'individual') assert.equal(evidenceDate, date);
    assert.ok(Date.parse(reviewedAt) >= Date.parse('2026-11-11T00:00:00Z'));
    assert.ok(['calendar_resolved', 'calendar_resolved_legacy_difference', 'missing_text'].includes(status));
    assert.ok(!rules.unsupported[date]);
  }
  assert.equal(baseline.reviews.filter(r => r[2] === 'individual').length, 25);
  for (const date of baseline.not_reviewed.unreviewed_dates) assert.ok(!dates.has(date));
  assert.equal(dates.size + baseline.not_reviewed.unreviewed_dates.length + baseline.not_reviewed.unsupported_dates.length, 365);
});

test('verse repair ledger lists bounded inserts and explicit corrections of importer damage', () => {
  assert.equal(repair.scope, 'temporary-copy-only');
  assert.match(repair.purpose, /not-an-automatic-ingestion/);
  assert.equal(repair.inserted_missing_rows.length, 21);
  assert.equal(repair.replaced_corrupted_rows.length, 9);
  assert.match(repair.replacement_reason, /needs owner confirmation/);
  const vaticanDates = new Set(vatican.captures.map(c => c[0]));
  for (const date of repair.source_dates) assert.ok(vaticanDates.has(date), date);
  for (const row of repair.replaced_corrupted_rows) {
    assert.match(row.old_sha256, /^[a-f0-9]{64}$/);
    assert.match(row.new_sha256, /^[a-f0-9]{64}$/);
    assert.notEqual(row.old_sha256, row.new_sha256);
    assert.ok(!repair.inserted_missing_rows.includes(row.verse));
  }
  for (const capture of vatican.captures) {
    assert.match(capture[2], /^[a-f0-9]{64}$/);
    assert.match(capture[3], /^[a-f0-9]{64}$/);
  }
});
