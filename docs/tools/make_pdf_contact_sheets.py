#!/usr/bin/env python3
"""Create contact sheets from rendered PDF page PNGs for quick visual QA."""

from __future__ import annotations

import argparse
import math
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


def natural_key(path: Path) -> tuple[int, str]:
    digits = "".join(ch for ch in path.stem if ch.isdigit())
    return (int(digits) if digits else 0, path.name)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("input_dir", type=Path)
    parser.add_argument("output_dir", type=Path)
    parser.add_argument("--per-sheet", type=int, default=12)
    args = parser.parse_args()

    pages = sorted(args.input_dir.glob("page-*.png"), key=natural_key)
    args.output_dir.mkdir(parents=True, exist_ok=True)
    if not pages:
        raise SystemExit("No rendered page PNGs found")

    thumb_w = 255
    label_h = 24
    gap = 18
    cols = 3
    rows = math.ceil(args.per_sheet / cols)
    font = ImageFont.load_default()

    for sheet_index in range(math.ceil(len(pages) / args.per_sheet)):
        batch = pages[sheet_index * args.per_sheet : (sheet_index + 1) * args.per_sheet]
        thumbs: list[tuple[Path, Image.Image]] = []
        thumb_h = 0
        for page in batch:
            image = Image.open(page).convert("RGB")
            ratio = thumb_w / image.width
            thumb = image.resize((thumb_w, int(image.height * ratio)))
            thumb_h = max(thumb_h, thumb.height)
            thumbs.append((page, thumb))

        sheet_w = cols * thumb_w + (cols + 1) * gap
        sheet_h = rows * (thumb_h + label_h) + (rows + 1) * gap
        sheet = Image.new("RGB", (sheet_w, sheet_h), "white")
        draw = ImageDraw.Draw(sheet)

        for idx, (page, thumb) in enumerate(thumbs):
            row = idx // cols
            col = idx % cols
            x = gap + col * (thumb_w + gap)
            y = gap + row * (thumb_h + label_h + gap)
            sheet.paste(thumb, (x, y + label_h))
            draw.rectangle(
                [x, y + label_h, x + thumb.width - 1, y + label_h + thumb.height - 1],
                outline=(180, 180, 180),
                width=1,
            )
            draw.text((x, y), page.stem.replace("page-", "strona "), fill=(20, 20, 20), font=font)

        sheet.save(args.output_dir / f"contact-sheet-{sheet_index + 1:02d}.png")


if __name__ == "__main__":
    main()
