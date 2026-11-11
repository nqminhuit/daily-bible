#!/usr/bin/env node
'use strict';
// Maintenance-only Vietnam national daytime calendar: General Roman baseline plus fixed rules.
// It selects Gospel entry keys only; no Gospel mapping is approved here.
const fs = require('node:fs');
const crypto = require('node:crypto');
const { generate } = require('./export.cjs');
const { entries } = require('../../data/lectionary-table.json');

const FIRST_YEAR = 2021;
const LAST_YEAR = 2050;
const SOURCE_URL = 'https://github.com/romcal/romcal';
const TRANSFERS = { epiphanyOnSunday: true, ascensionOnSunday: true, corpusChristiOnSunday: true };
const MARTYRS = 'andrew_dung_lac_priest_and_companions_martyrs';
const REVIEW_NOTE = 'Vietnam national daytime calendar: General Roman baseline with Epiphany, Ascension and Corpus Christi on Sunday and the Vietnamese Martyrs solemnity on November 24 (owner-approved 2026-11-11). No diocesan overlay; Tết Masses out of scope.';

function martyrsDate(year) {
  const nov24 = new Date(Date.UTC(year, 10, 24));
  return nov24.getUTCDay() === 0 ? `${year}-11-25` : `${year}-11-24`;
}

function rulesVersion(base) {
  // Independent of the requested years, so importing more years keeps reviews valid.
  const kinds = Object.keys(entries).sort().map(key => [key, entries[key].ref ? 'ref' : 'weekday']);
  const hash = crypto.createHash('sha256')
    .update(fs.readFileSync(__filename)).update(fs.readFileSync(require.resolve('./export.cjs')))
    .update(JSON.stringify({ base: base.rules_version, kinds })).digest('hex').slice(0, 20);
  return 'vietnam-general-' + hash;
}

async function generateVietnam(years) {
  if (!years.length || years.some(y => !Number.isInteger(y) || y < FIRST_YEAR || y > LAST_YEAR)) {
    throw new Error(`years must be integers in ${FIRST_YEAR}..${LAST_YEAR}`);
  }
  const base = await generate('roman-general', years, TRANSFERS);
  const martyrs = new Set(years.map(martyrsDate));
  const days = base.days.map(day => {
    let out = { ...day };
    if (martyrs.has(day.date)) {
      out = { ...out, observance: MARTYRS, rank: 'SOLEMNITY', celebration: 'Các Thánh Tử Đạo Việt Nam', gospel_entry: 'observance:' + MARTYRS };
      delete out.unsupported_reason;
    }
    if (out.unsupported_reason) return out;
    return { ...out, source_url: SOURCE_URL, review_note: REVIEW_NOTE };
  });
  return { gospel_entry_schema: base.gospel_entry_schema, calendar_id: 'vietnam', rules_version: rulesVersion(base),
    source_url: SOURCE_URL, base_rules_version: base.rules_version, days };
}

function parseYears(args) {
  return args.flatMap(arg => {
    const range = /^(\d+)-(\d+)$/.exec(arg);
    if (!range) return [Number(arg)];
    const [from, to] = [Number(range[1]), Number(range[2])];
    return Array.from({ length: Math.max(0, to - from + 1) }, (_, i) => from + i);
  });
}

if (require.main === module) {
  generateVietnam(parseYears(process.argv.slice(2)))
    .then(manifest => process.stdout.write(JSON.stringify(manifest) + '\n'))
    .catch(error => { console.error(error.message); process.exitCode = 1; });
}
module.exports = { generateVietnam };
