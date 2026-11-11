#!/usr/bin/env python3
"""Per-verse variants across all replayed pages, compared with a verse store. Offline, read-only."""
import re, sys, json, sqlite3, collections, unicodedata
gospels, db, out = sys.argv[1:4]
BOOK = {"mt": "Mt", "mc": "Mc", "lc": "Lc", "ga": "Ga"}
variants = collections.defaultdict(lambda: collections.defaultdict(list))  # key -> text -> [dates]
stray = collections.Counter(); book = chap = date = None
for line in open(gospels, encoding="utf-8"):
    line = line.strip()
    m = re.match(r"URL: .*/(\d{4})/(\d{2})/(\d{2})\.html", line)
    if m: date = "-".join(m.groups()); continue
    if line.startswith("__ref__: "):
        m = re.match(r"\s*([A-Za-z]{1,3})\s*(\d+)\s*,", line[9:]); book, chap = (BOOK.get(m.group(1).lower()), int(m.group(2))) if m else (None, None); continue
    m = re.fullmatch(r"\[\[chapter:(\d+)\]\]", line)
    if m and chap and int(m.group(1)) == chap + 1: chap += 1; continue
    m = re.match(r"\{\{(\d+)([A-Za-z]?)\}\}\s*(.*)", line)
    if m and book:
        variants[(book, chap, int(m.group(1)), m.group(2))][" ".join(m.group(3).split())].append(date)
    elif line and line != "-------" and "Tin Mừng" not in line: stray[date] += 1
def norm(t):  # wording comparison ignores spacing around punctuation and quote style
    t = unicodedata.normalize("NFC", t); t = re.sub(r"[“”\"‘’']", "'", t); t = re.sub(r"\s*([:;!?,.])\s*", r"\1", t); return " ".join(t.split()).strip()
con = sqlite3.connect("file:%s?mode=ro" % db, uri=True)
store = {(b, c, v, s or ""): t for b, c, v, s, t in con.execute("select book,chapter,verse,verse_suffix,text from verses")}
rep = {"store_rows": len(store), "crawl_keys": len(variants), "same": 0, "punctuation_only": 0, "wording": [], "missing_in_store": [], "store_only": [], "multi_variant_keys": 0}
best = {}
for k, vs in variants.items():
    groups = collections.defaultdict(list)
    for t, ds in vs.items(): groups[norm(t)] += [(d, t) for d in ds]
    if len(groups) > 1: rep["multi_variant_keys"] += 1
    top = max(groups.values(), key=len); text = collections.Counter(t for _, t in top).most_common(1)[0][0]
    best[k] = {"text": text, "pages": len(top), "total": sum(len(g) for g in groups.values()), "variants": len(groups)}
    name = "%s %d,%d%s" % k
    if k not in store: rep["missing_in_store"].append(name)
    elif store[k] == text: rep["same"] += 1
    elif norm(store[k]) == norm(text) or norm(store[k]) in groups: rep["punctuation_only" if norm(store[k]) == norm(text) else "minority_variant_in_store"] = rep.get("punctuation_only" if norm(store[k]) == norm(text) else "minority_variant_in_store", 0) + 1
    else: rep["wording"].append({"ref": name, "store": store[k], "crawl": text, "pages": len(top), "of": best[k]["total"]})
rep["store_only"] = sorted("%s %d,%d%s" % k for k in store if k not in variants)
rep["pages_with_unnumbered_lines"] = len(stray)
json.dump({"report": rep, "best": {"%s\t%d\t%d\t%s" % k: v for k, v in sorted(best.items())}}, open(out, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
print({k: (v if isinstance(v, int) else len(v)) for k, v in rep.items()})
