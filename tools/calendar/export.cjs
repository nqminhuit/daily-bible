#!/usr/bin/env node
// Maintenance-only adapter. Go servers and offline lookups do not need Node.
const { Romcal } = require('romcal');
const { GeneralRoman_En } = require('@romcal/calendar.general-roman');
const crypto = require('node:crypto');
const { dependencies } = require('./package.json');
// Keep in agreement with internal/lectionary.GospelEntrySchemaVersion.
const GOSPEL_ENTRY_SCHEMA = 4;
const table = require('../../data/lectionary-table.json').entries;

const PASSION_UNSUPPORTED = new Set(['palm_sunday_of_the_passion_of_the_lord', 'friday_of_the_passion_of_the_lord']);
const NO_DAYTIME_GOSPEL = new Set(['holy_thursday', 'holy_saturday']);
const CYCLE_SUFFIX = /_sunday$|^sunday_/;
const SUNDAY_CYCLE_WEEKDAYS = { ordinary_time_18_monday: 1, ordinary_time_18_tuesday: 2 };
const MEMORIAL_RANKS = new Set(['MEMORIAL', 'OPTIONAL_MEMORIAL', 'COMMEMORATION']);

function temporalKey(day, previousDays) {
  const id = day.temporal;
  if (CYCLE_SUFFIX.test(id)) return `temporal:${id}:${day.sunday_cycle}`;
  const cycleKey = `temporal:${id}:${day.sunday_cycle}`;
  if (!(cycleKey in table)) return `temporal:${id}`;
  const back = SUNDAY_CYCLE_WEEKDAYS[id];
  if (back) {
    // The cycle Gospel applies only when the preceding Sunday was the 18th Sunday itself.
    const sunday = previousDays.get(addDays(day.date, -back));
    if (!sunday || sunday.observance !== 'ordinary_time_18_sunday') return `temporal:${id}`;
  }
  return cycleKey;
}

function addDays(date, n) {
  const d = new Date(date + 'T00:00:00Z');
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
}

// previousDays: Map of date -> day, used to look back at the preceding Sunday.
function gospelEntry(day, previousDays = new Map()) {
  if (NO_DAYTIME_GOSPEL.has(day.observance)) return { unsupported_reason: 'no daytime Gospel entry is defined for this day' };
  if (PASSION_UNSUPPORTED.has(day.observance)) return { unsupported_reason: 'Passion reading crosses chapters; not supported yet' };
  const cycle = day.sunday_cycle;
  if (day.observance === day.temporal && (day.rank === 'WEEKDAY' || day.rank === 'SUNDAY')) {
    return { gospel_entry: temporalKey(day, previousDays) };
  }
  const plain = `observance:${day.observance}`;
  const cycled = `${plain}:${cycle}`;
  if (table[plain]?.ref) return { gospel_entry: plain };
  if (table[cycled]?.ref) return { gospel_entry: cycled };
  if (['YEAR_A', 'YEAR_B', 'YEAR_C'].some(c => `${plain}:${c}` in table)) return { gospel_entry: cycled };
  if (table[plain]?.weekday || MEMORIAL_RANKS.has(day.rank)) return { gospel_entry: temporalKey(day, previousDays) };
  return { gospel_entry: cycled };
}

// transfers: explicit Sunday-transfer options for a reviewed caller; the default profile and its version are unchanged.
async function generate(calendar, years, transfers = {}) {
  if (calendar === 'vietnam') throw new Error('Vietnam Romcal export disabled: unreviewed Vietnamese Martyrs date/rank discrepancy; provide reviewed national definitions (see data/calendar-research.md)');
  if (calendar !== 'roman-general') throw new Error('calendar must be roman-general; other policies require reviewed definitions');
  if (!years.length || years.some(y => !Number.isInteger(y) || y < 1970 || y > 2199)) throw new Error('years must be integers in 1970..2199');
  const options = { epiphanyOnSunday: false, ascensionOnSunday: false, corpusChristiOnSunday: false };
  for (const [name, value] of Object.entries(transfers)) {
    if (!(name in options) || typeof value !== 'boolean') throw new Error('unknown transfer option ' + name);
    options[name] = value;
  }
  const version = 'romcal-' + dependencies.romcal + '-' + crypto.createHash('sha256')
    .update(JSON.stringify({ adapter: 4, gospel_entry_schema: GOSPEL_ENTRY_SCHEMA, calendar, dependencies, options })).digest('hex').slice(0, 16);
  const engine = new Romcal({ localizedCalendar: GeneralRoman_En, ...options });
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
        alternatives: alternatives.slice(1).map(d => d.id),
        seasons: day.seasons.map(s => s.toLowerCase()) });
    }
  }
  const byDate = new Map(days.map(day => [day.date, day]));
  for (const day of days) Object.assign(day, gospelEntry(day, byDate));
  return { gospel_entry_schema: GOSPEL_ENTRY_SCHEMA, calendar_id: calendar, rules_version: version,
    source_url: 'https://github.com/romcal/romcal', options, days };
}
if (require.main === module) {
  const [calendar, ...years] = process.argv.slice(2);
  generate(calendar, years.map(Number)).then(value => process.stdout.write(JSON.stringify(value) + '\n'))
    .catch(error => { console.error(error.message); process.exitCode = 1; });
}
module.exports = { generate, gospelEntry };
