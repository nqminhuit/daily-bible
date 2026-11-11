'use strict';

// Source-derived specification cases, not an implemented national calendar.
// In particular, agreeing the target scope must not enable the rejected plugin.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { generate } = require('./export.cjs');
const dossier = JSON.parse(fs.readFileSync(
  path.join(__dirname, '../../data/vietnam-national-sources.json'), 'utf8'));

function assertDate(value) {
  assert.match(value, /^\d{4}-\d{2}-\d{2}$/);
  assert.equal(new Date(`${value}T00:00:00Z`).toISOString().slice(0, 10), value);
}

test('national source dossier retains evidence without becoming a calendar or Gospel approval', () => {
  assert.equal(dossier.schema_version, 1);
  assert.equal(dossier.purpose, 'national-calendar-source-dossier');
  assert.equal(dossier.status, 'partial-primary-rule-facts-plus-owner-approved-baseline-and-diocesan-evidence');
  assert.equal(dossier.gospel_entry_schema, undefined);
  assert.equal(dossier.days, undefined);
  const sources = new Map(dossier.sources.map(source => [source.id, source]));
  assert.equal(sources.size, dossier.sources.length);
  for (const source of sources.values()) {
    assert.equal(new URL(source.url).origin, 'https://hdgmvietnam.com');
    assert.ok(Number.isFinite(Date.parse(source.requested_at)));
    for (const field of ['html_sha256', 'text_sha256']) {
      assert.match(source[field], /^[a-f0-9]{64}$/);
    }
    for (const field of ['issued_on', 'page_published_on', 'page_updated_on']) {
      if (source[field]) {
        assertDate(source[field]);
        assert.ok(Date.parse(source[field]) <= Date.parse(source.requested_at));
      }
    }
  }
  for (const fact of [...dossier.mass_assignments, ...dossier.proclaimed_dates]) {
    assertDate(fact.date);
    assert.ok(sources.has(fact.source_id));
    assert.equal(fact.gospel_ref, undefined);
    assert.equal(fact.evidence_date, undefined); // Norms are not dated Gospel pages.
  }
  for (const id of dossier.season_and_hours.source_ids) assert.ok(sources.has(id));
  for (const fact of dossier.conditional_dates) {
    assert.ok(sources.has(fact.source_id));
    for (const date of fact.dates) assertDate(date);
  }
});

test('published national Mass cases are not satisfied by the Roman General profile', async () => {
  const expected = {
    '2022-01-02': 'epiphany_of_the_lord',
    '2024-02-10': 'tet_day_1_new_year',
    '2024-02-11': 'tet_day_2_ancestors',
    '2026-02-18': 'tet_day_2_ancestors',
    '2026-02-19': 'tet_day_3_labor',
    '2026-02-20': 'ash_wednesday',
    '2026-11-01': 'all_saints',
  };
  assert.deepEqual(Object.fromEntries(dossier.mass_assignments.map(fact =>
    [fact.date, fact.formulary])), expected);
  const roman = await generate('roman-general', [2022, 2024, 2026]);
  const days = new Map(roman.days.map(day => [day.date, day]));
  for (const fact of dossier.mass_assignments) {
    assert.equal(fact.scope, 'vietnam');
    assert.equal(fact.mass_option, 'day');
    if (fact.date === '2026-11-01') {
      assert.equal(days.get(fact.date).observance, fact.formulary);
    } else {
      assert.notEqual(days.get(fact.date).observance, fact.formulary,
        `${fact.date}: never substitute Roman General for the sourced national Mass`);
    }
  }
});

test('2026 Ash Mass transfer does not transfer Lent, the Hours or Easter', async () => {
  const facts = dossier.season_and_hours;
  assert.equal(facts.lent_starts_on, '2026-02-18');
  assert.equal(facts.ash_wednesday_hours_on, '2026-02-18');
  assert.equal(facts.ash_wednesday_mass_on, '2026-02-20');
  const roman = await generate('roman-general', [2026]);
  assert.equal(roman.days.find(day => day.date === facts.lent_starts_on).observance,
    'ash_wednesday');
  const easter = dossier.proclaimed_dates.find(fact => fact.observance === 'easter_sunday');
  assert.equal(easter.date, '2026-04-05');
  assert.equal(roman.days.find(day => day.date === easter.date).observance, 'easter_sunday');
});

test('Ascension on Sunday is an owner choice with diocesan evidence, never a national directive', () => {
  const fact = dossier.conditional_dates.find(day => day.observance === 'ascension_of_the_lord');
  assert.deepEqual(fact.dates, ['2026-05-14', '2026-05-17']);
  assert.match(fact.status, /owner-chose-sunday-2026-11-11-no-national-default-exists/);
  assert.equal(dossier.target_policy.ascension, 'sunday-owner-decision-2026-11-11');
  assert.equal(dossier.mass_assignments.some(day => fact.dates.includes(day.date)), false);
  assert.equal(dossier.proclaimed_dates.some(day => fact.dates.includes(day.date)), false);
});

test('diocesan sources stay separate from national directives and need two dioceses per fact', () => {
  const national = new Set(dossier.sources.map(source => source.id));
  const diocesan = new Map(dossier.diocesan_sources.map(source => [source.id, source]));
  for (const source of diocesan.values()) {
    assert.ok(!national.has(source.id));
    assert.equal(source.kind, 'official-diocesan-publication-not-a-national-directive');
    assert.equal(new URL(source.url).protocol, 'https:');
    assert.match(source.sha256, /^[a-f0-9]{64}$/);
  }
  for (const fact of dossier.diocesan_facts) {
    assertDate(fact.date);
    assert.equal(fact.gospel_ref, undefined);
    assert.ok(new Set(fact.source_ids.map(id => diocesan.get(id).diocese)).size >= 2);
  }
});

test('owner-approved target does not authorize Vietnam provider activation', async () => {
  assert.equal(dossier.target_policy.calendar_id, 'vietnam');
  assertDate(dossier.target_policy.scope_approved_on);
  assert.equal(dossier.target_policy.mass_option, 'day');
  assert.equal(dossier.target_policy.diocesan_overlay, null);
  assert.equal(dossier.target_policy.gospel_choice,
    'assigned-reference-then-first-listed-full-authorized-option');
  assert.equal(dossier.target_policy.activation,
    'blocked-pending-reviewed-rules-readings-and-coverage');
  await assert.rejects(generate('vietnam', [2026]), /disabled:.*provide reviewed national definitions/);
});
