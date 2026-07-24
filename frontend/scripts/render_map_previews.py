"""Render full map previews from Tiled .tmj files, scaled to ~400x250."""

import json
import math
import sys
from pathlib import Path

from PIL import Image

PROJECT_ROOT = Path(__file__).resolve().parent.parent.parent
ASSETS_DIR = PROJECT_ROOT / "frontend" / "public" / "assets"
MAPS_DIR = ASSETS_DIR / "maps"
TILES_DIR = ASSETS_DIR / "tiles"
OUTPUT_DIR = MAPS_DIR  # same folder as .tmj files
PREVIEW_W = 400
PREVIEW_H = 250
COLLISION_LAYER_NAME = "Collision"


def parse_tilesets(tilesets_raw: list[dict]) -> list[dict]:
    """Parse tileset definitions, adding `columns` computed from image dims."""
    result = []
    for i, ts in enumerate(tilesets_raw):
        firstgid = ts["firstgid"]
        image_path_rel = ts["image"]  # e.g. "../tiles/Grass_Middle.png"
        image_name = Path(image_path_rel).name
        columns = ts.get("columns") or (ts["imagewidth"] // ts["tilewidth"])
        # Determine firstgid of next tileset
        if i + 1 < len(tilesets_raw):
            next_gid = tilesets_raw[i + 1]["firstgid"]
        else:
            next_gid = 2_000_000_000  # sentinel
        result.append({
            "firstgid": firstgid,
            "next_gid": next_gid,
            "image_name": image_name,
            "image_width": ts["imagewidth"],
            "image_height": ts["imageheight"],
            "tile_width": ts["tilewidth"],
            "tile_height": ts["tileheight"],
            "columns": columns,
        })
    return result


def find_tileset(gid: int, tilesets: list[dict]) -> dict | None:
    """Return the tileset that owns `gid`."""
    for ts in tilesets:
        if ts["firstgid"] <= gid < ts["next_gid"]:
            return ts
    return None


def get_tile_src_rect(gid: int, ts: dict) -> tuple[int, int, int, int]:
    """Return (sx, sy, sw, sh) in tileset image pixels for `gid`."""
    local_idx = gid - ts["firstgid"]
    cols = ts["columns"]
    col = local_idx % cols
    row = local_idx // cols
    sx = col * ts["tile_width"]
    sy = row * ts["tile_height"]
    return (sx, sy, ts["tile_width"], ts["tile_height"])


def render_map(tmj_path: Path) -> Image.Image | None:
    """Render all visible tile layers (skipping Collision/objectgroup) into one PIL image."""
    data = json.loads(tmj_path.read_text(encoding="utf-8"))

    map_w = data["width"]
    map_h = data["height"]
    tile_w = data["tilewidth"]
    tile_h = data["tileheight"]

    full_w = map_w * tile_w
    full_h = map_h * tile_h

    # Parse tilesets
    tilesets = parse_tilesets(data.get("tilesets", []))

    # Pre-load tileset images
    tileset_images: dict[str, Image.Image] = {}
    for ts in tilesets:
        img_path = TILES_DIR / ts["image_name"]
        if img_path.exists():
            tileset_images[ts["image_name"]] = Image.open(img_path).convert("RGBA")
        else:
            print(f"  WARNING: tileset image not found: {img_path}")

    # Identify tile layers (skip "objectgroup" and "Collision")
    layers = []
    for layer in data.get("layers", []):
        if layer.get("type") != "tilelayer":
            continue
        if layer.get("name") == COLLISION_LAYER_NAME:
            continue
        if not layer.get("visible", True):
            continue
        layers.append(layer)

    if not layers:
        print("  No renderable tile layers found.")
        return None

    print(f"  Rendering {len(layers)} layers onto {full_w}x{full_h} canvas...")

    canvas = Image.new("RGBA", (full_w, full_h), (0, 0, 0, 0))

    for layer in layers:
        data_vals = layer.get("data", [])
        if not data_vals:
            continue
        layer_opacity = layer.get("opacity", 1.0)

        for idx, gid in enumerate(data_vals):
            if gid == 0:
                continue
            ts = find_tileset(gid, tilesets)
            if ts is None or ts["image_name"] not in tileset_images:
                continue

            src_img = tileset_images[ts["image_name"]]
            sx, sy, sw, sh = get_tile_src_rect(gid, ts)

            col = idx % map_w
            row = idx // map_w
            dx = col * tile_w
            dy = row * tile_h

            # Crop tile from tileset
            tile = src_img.crop((sx, sy, sx + sw, sy + sh))
            if tile.size != (tile_w, tile_h):
                # Protect against tileset image boundary mismatches
                tile = tile.resize((tile_w, tile_h))

            if layer_opacity < 1.0:
                # Apply layer opacity
                r, g, b, a = tile.split()
                a = a.point(lambda x: int(x * layer_opacity))
                tile = Image.merge("RGBA", (r, g, b, a))

            canvas.paste(tile, (dx, dy), tile)

    # Close tileset images
    for img in tileset_images.values():
        img.close()

    return canvas


def scale_to_fit(img: Image.Image, max_w: int, max_h: int) -> Image.Image:
    """Scale `img` to fit within max_w x max_h, maintaining aspect ratio."""
    w, h = img.size
    scale = min(max_w / w, max_h / h)
    if scale >= 1.0:
        return img.copy()
    new_w = max(1, int(w * scale))
    new_h = max(1, int(h * scale))
    return img.resize((new_w, new_h), Image.LANCZOS)


def main():
    map_files = [
        MAPS_DIR / "village_adventure.tmj",
        MAPS_DIR / "winter.tmj",
        MAPS_DIR / "dark_village.tmj",
    ]

    for tmj_path in map_files:
        if not tmj_path.exists():
            print(f"SKIP: {tmj_path.name} not found")
            continue

        print(f"Processing: {tmj_path.name}")
        full_img = render_map(tmj_path)
        if full_img is None:
            continue

        print(f"  Full size: {full_img.size}")
        preview = scale_to_fit(full_img, PREVIEW_W, PREVIEW_H)
        print(f"  Preview size: {preview.size}")

        out_name = tmj_path.stem + "_preview.png"
        out_path = OUTPUT_DIR / out_name
        preview.save(out_path, "PNG")
        print(f"  Saved: {out_path}")

        full_img.close()
        preview.close()


if __name__ == "__main__":
    main()
