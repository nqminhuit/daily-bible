#!/usr/bin/env node
'use strict';
// Explicit, maintenance-only national pilot. This is NOT the disabled Vietnam
// Romcal plugin and is deliberately not wired into production export jobs.
const fs = require('node:fs');
const crypto = require('node:crypto');
const { generate } = require('./export.cjs');
const rules = require('../../data/vietnam-pilot-2026.json');
const dossier = require('../../data/vietnam-national-sources.json');

const NATIONAL_ORIGIN = 'https://hdgmvietnam.com';

// A national decision needs its captured HĐGMVN directive fact.
function nationalEvidence(decision) {
  const source = dossier.sources.find(candidate => candidate.id === decision.source_id);
  if (!source || new URL(source.url).origin !== NATIONAL_ORIGIN) throw new Error('missing national decision evidence');
  const fact = [...dossier.mass_assignments, ...dossier.proclaimed_dates].find(candidate =>
    candidate.date === decision.date && candidate.source_id === source.id &&
    (candidate.formulary || candidate.observance) === decision.identity);
  if (!fact) throw new Error('reviewed decision lacks the captured primary-source fact: ' + decision.date);
  return [source];
}

// Owner-approved 2026-11-11: agreeing official calendars of distinct dioceses.
function diocesanEvidence(decision) {
  const fact = dossier.diocesan_facts.find(candidate =>
    candidate.date === decision.date && candidate.identity === decision.identity);
  const ids = decision.source_ids || [];
  if (!fact || ids.length !== fact.source_ids.length || ids.some(id => !fact.source_ids.includes(id))) {
    throw new Error('diocesan decision lacks its recorded fact: ' + decision.date);
  }
  const sources = ids.map(id => dossier.diocesan_sources.find(candidate => candidate.id === id));
  if (sources.some(source => !source || new URL(source.url).protocol !== 'https:')) throw new Error('missing diocesan evidence: ' + decision.date);
  if (new Set(sources.map(source => source.diocese)).size < rules.diocesan_evidence_rule.minimum_dioceses) {
    throw new Error('diocesan evidence needs agreeing calendars of distinct dioceses: ' + decision.date);
  }
  if (decision.owner_decision && !rules.owner_decisions[decision.owner_decision]) throw new Error('unknown owner decision: ' + decision.date);
  return sources;
}

function decide(day, decision) {
  if (!decision.note) throw new Error('decision lacks a note: ' + day.date);
  const sources = decision.evidence === 'diocesan' ? diocesanEvidence(decision) : nationalEvidence(decision);
  const selected = { ...day, source_url: sources[0].url, review_note: `${rules.scope_note} ${decision.note}` };
  if (decision.mode === 'mass') {
    selected.mass = { formulary: decision.identity, celebration: decision.celebration };
  } else if (day.observance !== decision.identity) {
    throw new Error('underlying provider disagrees with reviewed observance: ' + day.date);
  } else if (decision.mode === 'observance') {
    selected.celebration = decision.celebration;
  } else if (decision.mode === 'rank') {
    // A national proper outranks the universal memorial; the day keeps its identity.
    selected.rank = decision.rank;
    selected.celebration = decision.celebration;
  } else throw new Error('unknown decision mode: ' + day.date);
  return { selected, sources };
}

async function generatePilot(years) {
  if (!years.length || years.some(year => year !== 2026)) {
    throw new Error('reviewed national pilot supports only 2026; historical/future applicability is not assumed');
  }
  const base = await generate(rules.baseline.provider, [2026], rules.baseline.transfers);
  const decisions = new Map(rules.decisions.map(decision => [decision.date, decision]));
  if (decisions.size !== rules.decisions.length) throw new Error('duplicate reviewed decision');
  const proof = [];
  let decided = 0;
  const days = base.days.map(day => {
    const decision = decisions.get(day.date);
    if (rules.unsupported[day.date]) {
      if (decision) throw new Error('date is both decided and unsupported: ' + day.date);
      return { ...day, unsupported_reason: rules.unsupported[day.date] };
    }
    if (!decision) {
      const note = [rules.scope_note, rules.baseline.note, rules.baseline_notes[day.date]].filter(Boolean).join(' ');
      return { ...day, source_url: rules.baseline.source_url, review_note: note };
    }
    const { selected, sources } = decide(day, decision);
    proof.push(...sources);
    decided++;
    return selected;
  });
  if (decided !== decisions.size) throw new Error('decision outside supported year');
  const hash = crypto.createHash('sha256').update(fs.readFileSync(__filename))
    .update(JSON.stringify({ base: base.rules_version, rules, proof })).digest('hex').slice(0, 20);
  return { gospel_entry_schema: base.gospel_entry_schema, calendar_id: 'vietnam',
    rules_version: `vietnam-national-pilot-2026-${hash}`,
    // Issuer landing page only. Each day retains its own directive, diocesan
    // or baseline source; the issuer is NOT a complete annual-calendar source.
    source_url: NATIONAL_ORIGIN + '/', scope: rules.scope_note,
    base_rules_version: base.rules_version, days };
}

if (require.main === module) {
  generatePilot(process.argv.slice(2).map(Number))
    .then(manifest => process.stdout.write(JSON.stringify(manifest) + '\n'))
    .catch(error => { console.error(error.message); process.exitCode = 1; });
}
module.exports = { generatePilot };
