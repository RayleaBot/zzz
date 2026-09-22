"""Converts ZZZ-Plugin's pinned id maps into internal/assets/catalog.json:
agents, W-Engines and Bangboo with their names, rarities, elements,
specialties, descriptions and base stats.

Usage: python scripts/import-reference-data.py --references <参考项目/2026-09-15>

This reads data only. It never evaluates reference JavaScript or installs its
dependencies. The catalog carries its source version and attribution.
"""
import argparse
import html
import json
import pathlib
import re
import shutil

parser = argparse.ArgumentParser()
parser.add_argument("--references", type=pathlib.Path, required=True)
args = parser.parse_args()
refs = args.references.resolve()
root = pathlib.Path(__file__).resolve().parents[1]


def clean(value):
    if not isinstance(value, str):
        return ""
    value = re.sub(r"<br\s*/?>", "\n", value, flags=re.I)
    value = html.unescape(re.sub(r"<[^>]*>", "", value))
    return value.encode("utf-8", errors="replace").decode("utf-8").strip()


def load(file):
    return json.loads(file.read_text(encoding="utf-8"))


specialties = {"1": "强攻", "2": "击破", "3": "异常", "4": "支援", "5": "防护", "6": "命破"}
elements = {"200": "物理", "201": "火", "202": "冰", "203": "电", "205": "以太"}
entries = []
base = refs / "ZZZ-Plugin-dev/resources/map"
for kind, filename in (("character", "PartnerId2Data.json"), ("weapon", "WeaponId2Data.json"), ("buddy", "BangbooId2Data.json")):
    for item_id, raw in load(base / filename).items():
        name = clean(raw.get("full_name") or raw.get("name") or raw.get("Name"))
        if not name:
            continue
        weapon = raw.get("WeaponType", "")
        if isinstance(weapon, dict):
            weapon = "、".join(clean(v) for v in weapon.values())
        elif kind == "character":
            weapon = specialties.get(str(weapon), "")
        else:
            weapon = ""
        entries.append({
            "id": str(item_id),
            "name": name,
            "aliases": [clean(raw["name"])] if raw.get("name") and raw["name"] != name else [],
            "kind": kind,
            "rarity": {"S": 4, "A": 3, "B": 2}.get(raw.get("Rarity"), 0),
            "element": elements.get(str(raw.get("ElementType")), ""),
            "weapon": weapon,
            "description": clean(raw.get("Desc") or raw.get("desc")),
            "stats": {key: raw[key] for key in ("Attack", "Defence", "HpMax", "Crit", "CritDamage", "ElementMystery") if key in raw},
        })
catalog = {"version": "zzz-fb66219cec02", "source": "https://github.com/ZZZure/ZZZ-Plugin/tree/fb66219cec0294e1834bacdf0033b2d43a9ccaf4/resources/map", "entries": sorted(entries, key=lambda item: item["id"])}
(root / "internal/assets/catalog.json").write_bytes((json.dumps(catalog, ensure_ascii=False, indent=2) + "\n").encode("utf-8"))
shutil.copyfile(refs / "ZZZ-Plugin-dev/LICENSE", root / "LICENSES/ZZZ-Plugin-AGPL-3.0.txt")
print(json.dumps({"entries": len(entries)}, ensure_ascii=True))
