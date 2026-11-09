#!/usr/bin/env node
// Maintenance-only adapter. Go servers and offline lookups do not need Node.
const { Romcal } = require('romcal');
const { GeneralRoman_En } = require('@romcal/calendar.general-roman');
const { Vietnam_En } = require('@romcal/calendar.vietnam');
const crypto = require('node:crypto');
const { dependencies } = require('./package.json');

async function generate(calendar, years) {
  if (!['roman-general', 'vietnam'].includes(calendar)) throw new Error('calendar must be roman-general or vietnam');
  if (!years.length || years.some(y => !Number.isInteger(y) || y < 1970 || y > 2199)) throw new Error('years must be integers in 1970..2199');
  const options = calendar === 'roman-general'
    ? { epiphanyOnSunday: false, ascensionOnSunday: false, corpusChristiOnSunday: false }
    : { epiphanyOnSunday: true, ascensionOnSunday: true, corpusChristiOnSunday: true };
  const version = 'romcal-' + dependencies.romcal + '-' + crypto.createHash('sha256')
    .update(JSON.stringify({ adapter: 1, calendar, dependencies, options })).digest('hex').slice(0, 16);
  const engine = new Romcal({ localizedCalendar: calendar === 'roman-general' ? GeneralRoman_En : Vietnam_En, ...options });
  const days = [];
  for (const year of [...new Set(years)].sort()) {
    const result = await engine.generateCalendar(year);
    for (const [date, alternatives] of Object.entries(result).sort()) {
      if (!date.startsWith(year + '-')) continue;
      const day = alternatives[0];
      if (!day || day.date !== date) throw new Error('invalid calendar day ' + date);
      days.push({ date, observance: day.id, rank: day.rank,
        temporal: day.weekday?.id || day.id,
        sunday_cycle: day.cycles.sundayCycle, weekday_cycle: day.cycles.weekdayCycle,
        alternatives: alternatives.slice(1).map(d => d.id) });
    }
  }
  return { calendar_id: calendar, rules_version: version,
    source_url: 'https://github.com/romcal/romcal', options, days };
}
if (require.main === module) {
  const [calendar, ...years] = process.argv.slice(2);
  generate(calendar, years.map(Number)).then(value => process.stdout.write(JSON.stringify(value) + '\n'))
    .catch(error => { console.error(error.message); process.exitCode = 1; });
}
module.exports = { generate };
