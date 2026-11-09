const assert = require('node:assert/strict');
const test = require('node:test');
const { generate } = require('./export.cjs');

const calendar = generate('roman-general', [2023, 2024, 2025, 2026]);
test('proper celebrations replace Ordinary Sundays, not protected Sundays', async () => {
  const { days } = await calendar;
  const byDate = new Map(days.map(day => [day.date, day]));
  for (const date of ['2025-11-09', '2026-11-09']) {
    assert.equal(byDate.get(date).observance, 'dedication_of_the_lateran_basilica');
    assert.notEqual(byDate.get(date).temporal, byDate.get(date).observance);
  }
  assert.equal(byDate.get('2026-11-01').observance, 'all_saints');
  assert.equal(byDate.get('2025-02-02').observance, 'presentation_of_the_lord');
  assert.equal(byDate.get('2024-12-08').rank, 'SUNDAY');
  assert.equal(byDate.get('2024-12-09').observance, 'immaculate_conception_of_the_blessed_virgin_mary');
});
test('solemnities transfer around Sundays and Holy Week', async () => {
  const { days } = await calendar;
  const byDate = new Map(days.map(day => [day.date, day]));
  assert.equal(byDate.get('2024-04-08').observance, 'annunciation_of_the_lord');
  assert.notEqual(byDate.get('2024-03-25').observance, 'annunciation_of_the_lord');
  assert.notEqual(byDate.get('2023-03-19').observance, 'saint_joseph_spouse_of_the_blessed_virgin_mary');
  assert.match(byDate.get('2023-03-20').observance, /joseph/);
  assert.equal(byDate.get('2026-11-11').observance, 'martin_of_tours_bishop');
});
test('versions are deterministic, scope-specific, and output is complete', async () => {
  const roman = await generate('roman-general', [2026]);
  const again = await generate('roman-general', [2026, 2026]);
  const vietnam = await generate('vietnam', [2026]);
  assert.deepEqual(roman, again);
  assert.equal(roman.days.length, 365);
  assert.equal((await calendar).days.length, 365 * 4 + 1);
  assert.notEqual(roman.rules_version, vietnam.rules_version);
  assert.equal(roman.options.ascensionOnSunday, false);
  assert.equal(vietnam.options.ascensionOnSunday, true);
  // A national plugin is only a candidate policy, not proof of the particular
  // parish's celebration. November 15 still requires the deployment decision.
  assert.equal(vietnam.calendar_id, 'vietnam');
});
test('unsupported profiles/years are rejected instead of falling back', async () => {
  await assert.rejects(generate('unknown', [2026]));
  await assert.rejects(generate('roman-general', [NaN]));
  await assert.rejects(generate('roman-general', []));
});
