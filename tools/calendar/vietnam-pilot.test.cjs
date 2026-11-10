'use strict';
const assert = require('node:assert/strict');
const test = require('node:test');
const { generatePilot } = require('./vietnam-pilot.cjs');
const pilot = generatePilot([2026]);

test('national pilot covers the whole year but supports only nine explicitly sourced definitions', async () => {
  const m = await pilot;
  assert.equal(m.gospel_entry_schema, 3);
  assert.equal(m.calendar_id, 'vietnam');
  assert.match(m.rules_version, /^vietnam-national-pilot-2026-/);
  assert.equal(m.days.length, 365);
  assert.equal(m.days.filter(day => !day.unsupported_reason).length, 9);
  for (const day of m.days) {
    assert.equal(day.gospel_ref, undefined);
    if (!day.unsupported_reason) {
      assert.equal(new URL(day.source_url).origin, 'https://hdgmvietnam.com');
      assert.match(day.review_note, /pilot/);
    }
  }
});

test('national pilot assigns Ash and Tet Masses without shifting the underlying season', async () => {
  const byDate = new Map((await pilot).days.map(day => [day.date, day]));
  const first = byDate.get('2026-02-18');
  assert.equal(first.observance, 'ash_wednesday');
  assert.equal(first.mass.formulary, 'tet_day_2_ancestors');
  assert.deepEqual(first.seasons, ['lent']);
  assert.equal(byDate.get('2026-02-19').mass.formulary, 'tet_day_3_labor');
  const transferred = byDate.get('2026-02-20');
  assert.equal(transferred.observance, 'friday_after_ash_wednesday');
  assert.equal(transferred.mass.formulary, 'ash_wednesday');
  assert.deepEqual(transferred.seasons, ['lent']);
  assert.equal(byDate.get('2026-04-05').observance, 'easter_sunday');
  assert.equal(byDate.get('2026-06-07').mass.formulary, 'most_holy_body_and_blood_of_christ');
  assert.equal(byDate.get('2026-11-29').sunday_cycle, 'YEAR_B');
});

test('national pilot retains specific ambiguities and never substitutes the Roman General result', async () => {
  const days = new Map((await pilot).days.map(day => [day.date, day]));
  for (const date of ['2026-05-14', '2026-05-17']) assert.match(days.get(date).unsupported_reason, /Ascension/);
  assert.match(days.get('2026-11-15').unsupported_reason, /Martyrs external solemnity/);
  assert.ok(days.get('2026-11-09').unsupported_reason); // A plausible universal proper is not national review.
  assert.ok(days.get('2026-11-12').unsupported_reason);
  assert.equal(days.get('2026-11-01').observance, 'all_saints');
});

test('national pilot is deterministic and rejects extrapolation to unreviewed years', async () => {
  assert.deepEqual(await pilot, await generatePilot([2026, 2026]));
  for (const years of [[], [2024], [2027], [2026, 2027], [NaN]]) {
    await assert.rejects(generatePilot(years), /only 2026/);
  }
});
