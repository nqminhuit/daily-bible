'use strict';
const assert = require('node:assert/strict');
const test = require('node:test');
const { generatePilot } = require('./vietnam-pilot.cjs');
const rules = require('../../data/vietnam-pilot-2026.json');
const pilot = generatePilot([2026]);
const BASELINE_URL = 'https://github.com/romcal/romcal';

test('national pilot covers the year with sourced decisions over the owner-approved baseline', async () => {
  const m = await pilot;
  assert.equal(m.gospel_entry_schema, 3);
  assert.equal(m.calendar_id, 'vietnam');
  assert.match(m.rules_version, /^vietnam-national-pilot-2026-/);
  assert.equal(m.days.length, 365);
  assert.equal(rules.decisions.length, 13);
  assert.equal(m.days.filter(day => day.unsupported_reason).length, 3);
  assert.equal(m.days.filter(day => day.source_url === BASELINE_URL).length, 365 - 13 - 3);
  for (const day of m.days) {
    assert.equal(day.gospel_ref, undefined);
    if (day.unsupported_reason) continue;
    assert.match(day.review_note, /pilot/);
    const decided = rules.decisions.some(decision => decision.date === day.date);
    assert.equal(day.source_url !== BASELINE_URL, decided, day.date);
    if (!decided) assert.match(day.review_note, /baseline \(owner-approved 2026-11-11\).*not a national directive/);
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
  assert.equal(byDate.get('2026-06-07').observance, 'most_holy_body_and_blood_of_christ');
  assert.equal(byDate.get('2026-06-04').rank, 'WEEKDAY');
  assert.equal(byDate.get('2026-11-29').sunday_cycle, 'YEAR_B');
});

test('owner and diocesan decisions are explicit and keep their real sources', async () => {
  const days = new Map((await pilot).days.map(day => [day.date, day]));
  assert.equal(days.get('2026-01-04').observance, 'epiphany_of_the_lord');
  assert.equal(days.get('2026-01-06').rank, 'WEEKDAY');
  assert.equal(days.get('2026-05-17').observance, 'ascension_of_the_lord');
  assert.equal(days.get('2026-05-14').observance, 'matthias_apostle');
  assert.equal(days.get('2026-05-14').source_url, BASELINE_URL);
  assert.equal(days.get('2026-11-15').observance, 'ordinary_time_33_sunday');
  assert.match(days.get('2026-11-15').review_note, /Owner chose the Sunday Mass/);
  const martyrs = days.get('2026-11-24');
  assert.equal(martyrs.observance, 'andrew_dung_lac_priest_and_companions_martyrs');
  assert.equal(martyrs.rank, 'SOLEMNITY');
  assert.equal(martyrs.mass, undefined);
  for (const date of ['2026-01-04', '2026-05-17', '2026-11-15', '2026-11-24']) {
    assert.notEqual(days.get(date).source_url, BASELINE_URL);
    assert.equal(rules.decisions.find(decision => decision.date === date).evidence, 'diocesan');
  }
  assert.equal(days.get('2026-11-01').observance, 'all_saints');
});

test('known unresolved Vietnamese cases stay unsupported, not baseline', async () => {
  const days = new Map((await pilot).days.map(day => [day.date, day]));
  assert.match(days.get('2026-02-17').unsupported_reason, /Tết day 1/);
  assert.match(days.get('2026-10-04').unsupported_reason, /Rosary/);
  assert.match(days.get('2026-10-18').unsupported_reason, /Mission/);
});

test('national pilot is deterministic and rejects extrapolation to unreviewed years', async () => {
  assert.deepEqual(await pilot, await generatePilot([2026, 2026]));
  for (const years of [[], [2024], [2027], [2026, 2027], [NaN]]) {
    await assert.rejects(generatePilot(years), /only 2026/);
  }
});
