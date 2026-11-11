'use strict';
const assert = require('node:assert/strict');
const test = require('node:test');
const { generateVietnam } = require('./vietnam.cjs');
const { entries } = require('../../data/lectionary-table.json');
const baseline = require('../../data/vietnam-baseline-reviews.json');

const MARTYRS = 'andrew_dung_lac_priest_and_companions_martyrs';
const all = generateVietnam(Array.from({ length: 30 }, (_, i) => 2021 + i));
const byDate = async () => new Map((await all).days.map(day => [day.date, day]));

test('Vietnamese Martyrs fall on November 24, or transfer from a Sunday to November 25', async () => {
  const days = await byDate();
  assert.equal(days.get('2026-11-24').observance, MARTYRS);
  assert.equal(days.get('2026-11-24').rank, 'SOLEMNITY');
  assert.equal(days.get('2026-11-24').gospel_entry, 'observance:' + MARTYRS);
  assert.notEqual(days.get('2024-11-24').observance, MARTYRS);
  assert.equal(days.get('2024-11-25').observance, MARTYRS);
  assert.equal(days.get('2026-11-15').gospel_entry, 'temporal:ordinary_time_33_sunday:YEAR_A');
});

test('Epiphany, Ascension and Corpus Christi are on Sunday', async () => {
  const days = await byDate();
  assert.equal(days.get('2026-01-04').observance, 'epiphany_of_the_lord');
  assert.equal(days.get('2026-05-17').observance, 'ascension_of_the_lord');
  assert.equal(days.get('2026-06-07').observance, 'most_holy_body_and_blood_of_christ');
});

test('weekday memorials read the weekday entry and proper ones their own', async () => {
  const days = await byDate();
  assert.equal(entries['observance:agnes_of_rome_virgin'].weekday, true);
  assert.equal(days.get('2026-01-21').gospel_entry, 'temporal:ordinary_time_2_wednesday');
  assert.equal(days.get('2026-10-07').gospel_entry, 'observance:our_lady_of_the_rosary');
});

test('conditional weekday entries depend on the cycle and the preceding Sunday', async () => {
  const days = await byDate();
  assert.equal(days.get('2026-03-23').gospel_entry, 'temporal:lent_5_monday');
  assert.equal(days.get('2025-04-07').gospel_entry, 'temporal:lent_5_monday:YEAR_C');
  assert.equal(days.get('2026-08-03').gospel_entry, 'temporal:ordinary_time_18_monday:YEAR_A');
  assert.equal(days.get('2023-08-07').gospel_entry, 'temporal:ordinary_time_18_monday');
});

test('Holy Week days without a daytime Gospel entry are unsupported', async () => {
  const days = await byDate();
  for (const date of ['2027-03-21', '2027-03-25', '2027-03-26', '2027-03-27']) {
    assert.ok(days.get(date).unsupported_reason, date);
    assert.equal(days.get(date).gospel_entry, undefined);
    assert.equal(days.get(date).source_url, undefined);
  }
});

test('rules version is independent of the requested years and years are bounded', async () => {
  assert.equal((await generateVietnam([2026])).rules_version, (await all).rules_version);
  assert.equal((await all).gospel_entry_schema, 4);
  for (const years of [[], [2020], [2051], [2026.5]]) {
    await assert.rejects(generateVietnam(years), /years must be integers in 2021..2050/);
  }
});

test('the committed 2026 reviews agree with the table except seven known dates', async () => {
  const days = await byDate();
  const differing = [];
  for (const [date, ref] of baseline.reviews) {
    const key = days.get(date).gospel_entry;
    assert.ok(key, date);
    const entry = entries[key];
    if (!entry || entry.ref !== ref) differing.push(date);
  }
  assert.deepEqual(differing, ['2026-02-02', '2026-02-18', '2026-02-19', '2026-02-20', '2026-05-25', '2026-09-08', '2026-09-25']);
});

test('filling an uncovered Sunday entry does not change the rules version', async () => {
  const before = (await all).rules_version;
  entries['temporal:ordinary_time_12_sunday:YEAR_C'] = { ref: 'Lc 9,18-24', evidence: [] };
  try {
    assert.equal((await generateVietnam([2026])).rules_version, before);
  } finally {
    delete entries['temporal:ordinary_time_12_sunday:YEAR_C'];
  }
});
