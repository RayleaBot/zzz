"""Converts the pinned GachaClock banner history. Only text and dates are
kept; uncertain start times are marked, and no images are copied.

Usage: python scripts/import-banner-data.py <参考项目/2026-09-15>
Writes internal/assets/data/resources.json.
"""
import datetime as dt
import json
import pathlib
import re
import shutil
import sys

reference = pathlib.Path(sys.argv[1]).resolve() / "GachaClock-data"
plugin = pathlib.Path(__file__).resolve().parents[1]
source = json.loads((reference / "SOURCE.json").read_text(encoding="utf-8"))
raw = json.loads((reference / "zzz-history.json").read_text(encoding="utf-8"))


def timestamp(value):
    value = value.strip().replace("/", "-")
    for layout in ("%Y-%m-%d %H:%M:%S", "%Y-%m-%d %H:%M", "%Y-%m-%d"):
        try:
            return dt.datetime.strptime(value, layout).strftime("%Y-%m-%d %H:%M:%S")
        except ValueError:
            pass
    return ""


periods = []
for p in raw:
    parts = p["timer"].split("~")
    end, start = timestamp(parts[-1]), timestamp(parts[0])
    match = re.match(r"([\d.]+)(.*)", p["version"])
    if not end or not match:
        raise ValueError("Unsupported banner interval")
    periods.append((end, start, p, match.groups()))
periods.sort(key=lambda period: period[0])

pools = []
for end, start, p, (version, half) in periods:
    # A banner without a start opens at launch, or the day after the one
    # before it ends at 11:00; the latter is marked as estimated.
    estimated = False
    if not start:
        previous = [e for e, _, _, _ in periods if e < end]
        if p["timer"].startswith("公测开启后"):
            start = "2024-07-04 10:00:00"
        elif previous:
            day = dt.datetime.strptime(max(previous), "%Y-%m-%d %H:%M:%S") + dt.timedelta(days=1)
            start = day.replace(hour=11, minute=0, second=0).strftime("%Y-%m-%d %H:%M:%S")
            estimated = True
        else:
            estimated = True
    kind = "character" if p["type"] == "角色" else "weapon"
    row = {"version": version, "half": half, "from": start, "to": end, "kind": kind, "estimated_start": estimated,
           "characters5": [], "characters4": [], "weapons5": [], "weapons4": []}
    key = "characters" if kind == "character" else "weapons"
    row[key + "5"] = [p["s"]] if p.get("s") else []
    row[key + "4"] = p.get("a") or []
    pools.append(row)

data = {"version": "GachaClock-" + source["commit"][:12], "pools": pools}
text = json.dumps(data, ensure_ascii=False, separators=(",", ":")) + "\n"
(plugin / "internal/assets/data/resources.json").write_bytes(text.encode("utf-8"))
shutil.copyfile(reference / "LICENSE", plugin / "LICENSES/GachaClock-MIT.txt")
print(json.dumps({"pools": len(pools), "estimated_starts": sum(p["estimated_start"] for p in pools), "source": source["commit"]}))
