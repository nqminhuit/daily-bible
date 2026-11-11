#!/usr/bin/env python3
"""Group dated page Gospels by year-independent entry. Offline; output is a candidate table, not approvals."""
import json, re, sys, collections
cal = json.load(open(sys.argv[1])); gospels = sys.argv[2]; ledger = json.load(open(sys.argv[3])); exc = json.load(open(sys.argv[4])); out = sys.argv[5]
refs = {}; url = None
for line in open(gospels, encoding="utf-8"):
    m = re.match(r"URL: .*/(\d{4})/(\d{2})/(\d{2})\.html", line)
    if m: url = "-".join(m.groups())
    elif line.startswith("__ref__: ") and url: refs[url] = line[9:].strip(); url = None
refs.update(exc["page_read"]["refs"])
excluded = {}
for k in ("tet", "martyrs_external", "optional"):
    for d in exc[k]["dates"]: excluded[d] = k
for d, why in exc["other"].items(): excluded[d] = "other"
days = {d["date"]: d for d in cal["days"]}
SKIP = {"holy_thursday", "holy_saturday", "palm_sunday_of_the_passion_of_the_lord", "friday_of_the_passion_of_the_lord"}
COND = exc["conditional"]["entries"]; DEC = exc["decisions"]["entries"]

def is_temporal(d): return d["observance"] == d["temporal"] and d["rank"] in ("WEEKDAY", "SUNDAY")
def temporal_key(d):
    t = d["temporal"]
    return "temporal:%s:%s" % (t, d["sunday_cycle"]) if t.endswith("_sunday") or t.startswith("sunday_") else "temporal:" + t
def unanimous(rows):  # rows: {date: ref}; excluded dates are not evidence
    good = {d: r for d, r in rows.items() if d not in excluded}
    vals = set(good.values())
    return (next(iter(vals)), sorted(good)) if len(vals) == 1 else (None, sorted(good))

table, review = {}, []
groups = collections.defaultdict(dict)
for date, d in days.items():
    if date in refs and is_temporal(d) and d["observance"] not in SKIP: groups[temporal_key(d)][date] = refs[date]
for k, g in sorted(groups.items()):
    if k in COND:
        rule = COND[k]; bad = {}
        for date, r in g.items():
            want = rule.get(days[date]["sunday_cycle"], rule["default"])
            if date not in excluded and r != want: bad[date] = r
        for cyc, r in rule.items():
            if cyc == "why": continue
            table[k if cyc == "default" else k + ":" + cyc] = {"ref": r, "evidence": sorted(dt for dt, x in g.items() if x == r and dt not in excluded), "rule": rule["why"]}
        if bad: review.append({"entry": k, "kind": "conditional-rule-mismatch", "dates": bad})
        continue
    ref, ev = unanimous(g)
    if k in DEC: table[k] = {"ref": DEC[k]["ref"], "evidence": [dt for dt in ev if g[dt] == DEC[k]["ref"]], "decision": DEC[k]["why"]}
    elif ref: table[k] = {"ref": ref, "evidence": ev}
    elif ev: review.append({"entry": k, "kind": "temporal-disagreement", "dates": g})
for k, v in DEC.items():
    if k.startswith("temporal:") and k not in table: table[k] = {"ref": v["ref"], "evidence": [], "decision": v["why"]}
def temporal_ref(d):
    k = temporal_key(d); c = table.get(k + ":" + d["sunday_cycle"]) or table.get(k)
    return c["ref"] if c else None

cele = collections.defaultdict(list); via = collections.defaultdict(dict)
for date, d in days.items():
    if date in refs and d["observance"] not in SKIP and not is_temporal(d) and date not in excluded:
        cele[d["observance"]].append((date, refs[date], d["sunday_cycle"], d["rank"], temporal_ref(d) if d["temporal"] != d["observance"] else None))
for obs, rows in sorted(cele.items()):
    key = "observance:" + obs
    if key in DEC:
        table[key] = {"ref": DEC[key]["ref"], "evidence": [r[0] for r in rows if r[1] == DEC[key]["ref"]], "decision": DEC[key]["why"]}; continue
    allrefs = {r[1] for r in rows}; known = [r for r in rows if r[4] is not None]
    ferial = [r for r in known if r[1] == r[4]]
    rec = {"observance": obs, "ranks": sorted({r[3] for r in rows}), "rows": [list(r) for r in rows]}
    memorial = rows[0][3] in ("MEMORIAL", "OPTIONAL_MEMORIAL", "COMMEMORATION")
    proper = collections.Counter(r[1] for r in rows if r[4] is None or r[1] != r[4])
    if memorial and len(ferial) == len(known):
        table[key] = {"ferial": True, "evidence": [r[0] for r in known]}
        for r in rows: via[temporal_key(days[r[0]])][r[0]] = r[1]
    elif len(allrefs) == 1 and (len(rows) > 1 or not memorial):
        table[key] = {"ref": rows[0][1], "evidence": [r[0] for r in rows]}
    else:
        bycycle = collections.defaultdict(set)
        for r in rows: bycycle[r[2]].add(r[1])
        if not memorial and all(len(s) == 1 for s in bycycle.values()):
            for c, s in bycycle.items(): table["%s:%s" % (key, c)] = {"ref": next(iter(s)), "evidence": [r[0] for r in rows if r[2] == c]}
        else:
            rec["kind"] = "mixed"; review.append(rec)
# weekdays never seen undisplaced: take them from ferial memorial days when unanimous
for k, g in sorted(via.items()):
    if k not in table and len(g) >= 2 and len(set(g.values())) == 1:
        table[k] = {"ref": next(iter(g.values())), "evidence": sorted(g), "via": "memorial days that printed the weekday"}

def lookup(d):
    if d["observance"] in SKIP: return None
    if not is_temporal(d):
        o = table.get("observance:" + d["observance"]) or table.get("observance:%s:%s" % (d["observance"], d["sunday_cycle"]))
        if not o: return None
        if "ref" in o: return o["ref"]
    return temporal_ref(d)
test = {"agree": 0, "uncovered": [], "contradict": []}
for row in ledger["reviews"]:
    date, ref = row[0], row[1]; got = lookup(days[date])
    if got is None: test["uncovered"].append([date, ref, days[date]["observance"]])
    elif got == ref: test["agree"] += 1
    else: test["contradict"].append([date, ref, got, days[date]["observance"], excluded.get(date, "")])
cover = collections.Counter(); unc = collections.Counter()
for date, d in days.items():
    if lookup(d): cover[date[:4]] += 1
    else: unc[d["observance"] if not is_temporal(d) else temporal_key(d)] += 1
single = sorted(k for k, v in table.items() if len(v["evidence"]) < 2)
json.dump({"rules_version": cal["rules_version"], "table": table, "review": review, "test": test, "single_evidence_entries": single,
           "uncovered_entries_2021_2026": dict(unc), "excluded_dates": excluded}, open(out, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
print("refs", len(refs), "| entries", len(table), "| single-evidence", len(single), "| review", len(review),
      "| test agree/uncovered/contradict", test["agree"], len(test["uncovered"]), len(test["contradict"]), "| covered dates", dict(cover), "| uncovered entries", len(unc))
