"""Copies the upstream images, fonts and id maps the templates read from the
pinned reference snapshots into assets/<source>/. The plugin build ships the
top-level assets/ directory, and the artwork store serves these files until an
administrator downloads a newer copy of the source. assets/<source>.json
records the upstream commit and the totals the status shows.

Usage: python scripts/bundle-artwork.py --references <参考项目/2026-09-15>

Run it before building a package. Agent, W-Engine, Bangboo and Drive Disc
images from the ZZZeroUID mirrors, official images and nanoka Mindscape art are
fetched file by file at render time as ZZZ-Plugin does, and the Atlas is
downloaded on request, so none of them are bundled.
"""
import argparse
import fnmatch
import json
import pathlib
import shutil

# Source id: the reference snapshot and the files the Go code reads from it.
# A pattern's * also matches "/".
BUNDLES = {
    "zzz-plugin": ("ZZZ-Plugin", [
        "resources/*/images/*",
        "resources/common/fonts/inpinhongmengti.ttf",
        "resources/map/PartnerId2Data.json",
        "resources/map/WeaponId2Data.json",
        "resources/map/SuitData.json",
        "resources/map/ElementData.json",
        "resources/map/BangbooId2Data.json",
    ]),
    # The UID list page, drawn with miao's frame like Miao-Yunzai's.
    "miao-plugin": ("miao-plugin", [
        "resources/common/font/tttgbnumber.woff",
        "resources/common/font/NZBZ.woff",
        "resources/common/font/HYWH-65W.woff",
        "resources/common/bg/bg-hydro.webp",
        "resources/common/item/face.webp",
    ]),
    # The UID list page and the news pages.
    "yunzai-genshin": ("Yunzai-genshin", [
        "resources/img/icon/check.webp",
        "resources/ZZZero/img/other/banner.png",
        "resources/html/mysNews/iconfont.fb3712d.woff2",
        "resources/html/mysNews/mys.png",
        "resources/html/mysNews-list/蒙德.png",
        "resources/font/tttgbnumber.ttf",
    ]),
}

parser = argparse.ArgumentParser()
parser.add_argument("--references", type=pathlib.Path, required=True)
args = parser.parse_args()
refs = args.references.resolve()
root = pathlib.Path(__file__).resolve().parents[1]
game = json.loads((root / "internal/assets/game.json").read_text(encoding="utf-8"))
extensions = {source["id"]: tuple(source.get("extensions", [])) for source in game["artwork"]}

for source_id, (snapshot, patterns) in BUNDLES.items():
    base = refs / snapshot
    pinned = json.loads((refs / f"{snapshot}.source.json").read_text(encoding="utf-8"))
    names = [file.relative_to(base).as_posix() for file in base.rglob("*") if file.is_file() and ".git" not in file.relative_to(base).parts]
    # The download keeps the same extensions, so an update never drops a
    # bundled file type.
    names = [name for name in names if name.lower().endswith(extensions[source_id])]
    chosen = set()
    for pattern in patterns:
        matched = [name for name in names if fnmatch.fnmatchcase(name, pattern)]
        if not matched:
            raise SystemExit(f"{source_id}: {pattern} matches no file in {snapshot}")
        chosen.update(matched)
    target = root / "assets" / source_id
    shutil.rmtree(target, ignore_errors=True)
    size = 0
    for name in sorted(chosen):
        file = target / name
        file.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(base / name, file)
        size += file.stat().st_size
    record = {"commit": pinned["commit"], "archive": pinned["archive_url"], "files": len(chosen), "bytes": size}
    (root / "assets" / f"{source_id}.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    print(f"assets/{source_id}: {len(chosen)} files, {size / 1e6:.1f} MB at {pinned['commit'][:7]}")
