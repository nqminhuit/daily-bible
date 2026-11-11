#!/usr/bin/env python3
"""One-pass capture of dated Vatican News Gospel pages. 1 req/s, resumable, no retries/bypass."""
import datetime, hashlib, json, os, shutil, subprocess, sys, time
ROOT = os.path.dirname(os.path.abspath(__file__))
PAGES = os.path.join(ROOT, "pages")
LOG = os.path.join(ROOT, "requests.jsonl")
OLD = os.path.join(ROOT, "..", "vietnam-baseline-2026-11-11c", "captures")
URL = "https://www.vaticannews.va/vi/loi-chua-hang-ngay/%04d/%02d/%02d.html"
UA = "Mozilla/5.0 (compatible; research-client)"
STOP_AFTER = 15  # consecutive non-200 responses: assume blocking and stop

def sha(p):
    return hashlib.sha256(open(p, "rb").read()).hexdigest()

def log(rec):
    with open(LOG, "a") as f:
        f.write(json.dumps(rec, ensure_ascii=False) + "\n")

done = set()
if os.path.exists(LOG):
    for line in open(LOG):
        r = json.loads(line)
        if r["status"] == 200:
            done.add(r["date"])

# reuse earlier captures with their original request facts
old_log = os.path.join(OLD, "requests.jsonl")
if os.path.exists(old_log):
    for line in open(old_log):
        r = json.loads(line)
        if not r["id"].startswith("vatican-") or r.get("status") != 200:
            continue
        d = r["id"][len("vatican-"):]
        src = os.path.join(OLD, r["id"] + ".html")
        if d in done or len(d) != 10 or not os.path.exists(src) or sha(src) != r["html_sha256"]:
            continue
        shutil.copyfile(src, os.path.join(PAGES, d + ".html"))
        log({"date": d, "url": r["url"], "requested_at": r["requested_at"], "status": 200,
             "html_sha256": r["html_sha256"], "bytes": os.path.getsize(src), "reused_from": "vietnam-baseline-2026-11-11c"})
        done.add(d)

start = datetime.date.fromisoformat(sys.argv[1]); end = datetime.date.fromisoformat(sys.argv[2])
bad = 0; d = start
while d <= end:
    ds = d.isoformat(); d += datetime.timedelta(days=1)
    if ds in done:
        continue
    url = URL % (int(ds[:4]), int(ds[5:7]), int(ds[8:]))
    tmp = os.path.join(PAGES, ds + ".tmp")
    t = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    p = subprocess.run(["curl", "-sS", "-L", "--max-time", "60", "-A", UA, "-o", tmp,
                        "-w", "%{http_code} %{url_effective}", url], capture_output=True, text=True)
    out = p.stdout.split(" ", 1)
    code = int(out[0]) if out and out[0].isdigit() else 0
    rec = {"date": ds, "url": url, "requested_at": t, "status": code,
           "effective_url": out[1] if len(out) > 1 else "", "curl_exit": p.returncode}
    if code == 200 and os.path.exists(tmp):
        rec["html_sha256"] = sha(tmp); rec["bytes"] = os.path.getsize(tmp)
        os.replace(tmp, os.path.join(PAGES, ds + ".html")); bad = 0
    else:
        if os.path.exists(tmp): os.remove(tmp)
        rec["error"] = p.stderr.strip()[:200]; bad += 1
    log(rec)
    if bad >= STOP_AFTER:
        print("stopped: %d consecutive failures at %s" % (bad, ds)); sys.exit(2)
    time.sleep(1)
print("finished")
