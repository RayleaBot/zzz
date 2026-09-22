"""Imports ZZZ-Plugin's agent aliases (defSet/alias.yaml) into
internal/assets/catalog.json, with each agent's short name first. Only
built-in aliases are replaced; custom aliases stay in plugin settings.

Usage: python scripts/import-aliases.py <参考项目/2026-09-15>
Needs PyYAML.
"""
import json
import pathlib
import sys

import yaml

refs = pathlib.Path(sys.argv[1]).resolve()
root = pathlib.Path(__file__).resolve().parents[1]


def words(value):
    if isinstance(value, list):
        return [str(item).strip() for item in value if str(item).strip()]
    return [item.strip() for item in str(value or "").split(",") if item.strip()]


def unique(values, exclude):
    seen, out = {exclude}, []
    for value in values:
        if value and value not in seen:
            seen.add(value)
            out.append(value)
    return out


path = root / "internal/assets/catalog.json"
raw = path.read_bytes().decode("utf-8")
catalog = json.loads(raw)
if json.dumps(catalog, ensure_ascii=False, indent=2) + "\n" != raw:
    raise SystemExit(f"{path} is not in the expected layout")

table = yaml.safe_load((refs / "ZZZ-Plugin-dev/defSet/alias.yaml").read_text(encoding="utf-8"))
short_names = json.loads((refs / "ZZZ-Plugin-dev/resources/map/PartnerId2Data.json").read_text(encoding="utf-8"))
# alias.yaml names agents by their short or full names, some in 「」.
by_name = {}
for entry in catalog["entries"]:
    if entry["kind"] == "character":
        by_name[short_names.get(entry["id"], {}).get("name") or entry["name"]] = entry
        by_name[entry["name"]] = entry
unmatched = []
for key, values in table.items():
    entry = by_name.get(str(key)) or by_name.get(str(key).strip("「」"))
    if entry is None:
        unmatched.append(str(key))
        continue
    short = short_names.get(entry["id"], {}).get("name", "")
    entry["aliases"] = unique([short] + words(values), entry["name"])
counted = sum(len(entry.get("aliases", [])) for entry in catalog["entries"])
path.write_bytes((json.dumps(catalog, ensure_ascii=False, indent=2) + "\n").encode("utf-8"))
print(json.dumps({"aliases": counted, "unmatched": unmatched}, ensure_ascii=False))
