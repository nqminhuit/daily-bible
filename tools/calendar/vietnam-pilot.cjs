#!/usr/bin/env node
'use strict';
// Explicit, partial, maintenance-only national pilot. This is NOT the disabled
// Vietnam Romcal plugin and is deliberately not wired into production export jobs.
const fs = require('node:fs');
const crypto = require('node:crypto');
const { generate } = require('./export.cjs');
const rules = require('../../data/vietnam-pilot-2026.json');
const dossier = require('../../data/vietnam-national-sources.json');

async function generatePilot(years) {
  if (!years.length || years.some(year => year !== 2026)) {
    throw new Error('reviewed national pilot supports only 2026; historical/future applicability is not assumed');
  }
  const base = await generate('roman-general', [2026]);
  const sources = new Map(dossier.sources.map(source => [source.id, source]));
  const decisions = new Map(rules.decisions.map(decision => [decision.date, decision]));
  if (decisions.size !== rules.decisions.length) throw new Error('duplicate reviewed decision');
  const proof = [];
  const days = base.days.map(day => {
    const decision = decisions.get(day.date);
    if (!decision) return { ...day, unsupported_reason: rules.unsupported[day.date] || rules.unsupported.default };
    const source = sources.get(decision.source_id);
    if (!source || new URL(source.url).origin !== 'https://hdgmvietnam.com' || !decision.note) throw new Error('missing national decision evidence');
    const fact = [...dossier.mass_assignments, ...dossier.proclaimed_dates].find(fact =>
      fact.date === day.date && fact.source_id === source.id &&
      (fact.formulary || fact.observance) === decision.identity);
    if (!fact) throw new Error('reviewed decision lacks the captured primary-source fact: ' + day.date);
    proof.push(source);
    const selected = { ...day, source_url: source.url, review_note: `${rules.scope_note} ${decision.note}` };
    if (decision.mode === 'mass') {
      selected.mass = { formulary: decision.identity, celebration: decision.celebration };
    } else if (decision.mode === 'observance' && day.observance === decision.identity) {
      selected.celebration = decision.celebration;
    } else throw new Error('underlying provider disagrees with reviewed observance: ' + day.date);
    return selected;
  });
  if (days.filter(day => !day.unsupported_reason).length !== decisions.size) throw new Error('decision outside supported year');
  const hash = crypto.createHash('sha256').update(fs.readFileSync(__filename))
    .update(JSON.stringify({ base: base.rules_version, rules, proof })).digest('hex').slice(0, 20);
  return { gospel_entry_schema: base.gospel_entry_schema, calendar_id: 'vietnam',
    rules_version: `vietnam-national-pilot-2026-${hash}`,
    // Issuer landing page only. Each supported day retains its precise directive
    // URL/note; the issuer is NOT presented as a complete annual-calendar source.
    source_url: 'https://hdgmvietnam.com/', scope: rules.scope_note,
    base_rules_version: base.rules_version, days };
}

if (require.main === module) {
  generatePilot(process.argv.slice(2).map(Number))
    .then(manifest => process.stdout.write(JSON.stringify(manifest) + '\n'))
    .catch(error => { console.error(error.message); process.exitCode = 1; });
}
module.exports = { generatePilot };
