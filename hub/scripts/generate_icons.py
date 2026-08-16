#!/usr/bin/env python3
"""Generate Hub's PNG icons from the same geometry as public/favicon.svg.

Draws at 4x supersampling then downscales with LANCZOS for smooth edges.
Outputs (committed to the repo, no build-step dependency):
  public/icon-192.png         - PNG fallback for rel="icon"
  public/apple-touch-icon.png - 180x180, iOS home screen

Usage: python3 scripts/generate_icons.py  (requires Pillow)
"""

from PIL import Image, ImageDraw

ACCENT = "#D97757"
PAPER = "#F5F1EA"
GRID = 64  # matches favicon.svg viewBox


def draw(size: int) -> Image.Image:
    scale = 4  # supersampling factor
    s = size * scale / GRID
    img = Image.new("RGBA", (size * scale, size * scale), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)

    def box(x0, y0, x1, y1):
        return [x0 * s, y0 * s, x1 * s, y1 * s]

    d.rounded_rectangle(box(4, 4, 60, 60), radius=14 * s, fill=ACCENT)
    # geometric "H": two stems + crossbar, same coordinates as favicon.svg
    d.rectangle(box(20, 18, 27, 46), fill=PAPER)
    d.rectangle(box(37, 18, 44, 46), fill=PAPER)
    d.rectangle(box(20, 30, 44, 34), fill=PAPER)
    return img.resize((size, size), Image.LANCZOS)


def main() -> None:
    for name, size in [("icon-192", 192), ("apple-touch-icon", 180)]:
        path = f"public/{name}.png"
        draw(size).save(path)
        print(f"wrote {path} ({size}x{size})")


if __name__ == "__main__":
    main()
