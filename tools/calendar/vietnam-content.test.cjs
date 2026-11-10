'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const review = require('../../data/vietnam-content-review.json');
const captures = require('../../data/vietnam-content-captures.json');
const rules = require('../../data/vietnam-pilot-2026.json');

test('content reviews retain actual captures and do not become automatic calendar approvals', () => {
  assert.equal(review.auto_approve, false);
  assert.equal(review.activation, 'temporary-evaluation-only-not-production-approved');
  assert.equal(review.reviews.length, 6);
  assert.equal(new Set(review.reviews.map(r => r.date)).size, 6);
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

test('unknown Tet default stays pending and endpoint/history differences stay explicit', () => {
  assert.deepEqual(review.pending.map(r => r.date), ['2026-02-19']);
  assert.equal(review.pending[0].status, 'default-option-order-unestablished');
  assert.deepEqual(review.pending[0].candidate_refs_in_page_order, ['Mt 25,14-30', 'Ga 5,16-20']);
  assert.ok(!review.reviews.some(r => r.date === review.pending[0].date));
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
