"""Rebuild visitor delivery variants from the preserved original assets (requires Pillow)."""

from pathlib import Path

from PIL import Image


assets = Path(__file__).resolve().parents[1] / "packages/ui/src/assets"
with Image.open(assets / "nusantara.jpg") as original:
    for width in (640, 960, 1440, 1920):
        image = original.resize(
            (width, round(width * original.height / original.width)), Image.Resampling.LANCZOS
        )
        image.save(assets / f"nusantara-{width}.avif", quality=45, speed=6)
        image.save(assets / f"nusantara-{width}.webp", quality=50, method=6)

with Image.open(assets / "ikn-logo.png") as original:
    for size in (48, 96, 144):
        image = original.convert("RGBA").resize((size, size), Image.Resampling.LANCZOS)
        image.save(assets / f"ikn-logo-{size}.webp", quality=75, method=6)
