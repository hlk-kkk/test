"""Batch generate thumbnails for existing images in storage."""

import sys
from pathlib import Path


def main() -> None:
    from ..core.config import settings
    from ..services.thumbnail import generate_thumbnail, thumbnail_path_from_original

    storage_path = Path(settings.storage_path)
    if not storage_path.exists():
        print(f"Storage path not found: {storage_path}")
        sys.exit(1)

    extensions = {".jpg", ".jpeg", ".png", ".webp"}
    image_files = [
        f
        for f in storage_path.iterdir()
        if f.is_file() and f.suffix.lower() in extensions and "_thumb" not in f.stem
    ]

    total = len(image_files)
    skipped = 0
    created = 0
    failed = 0

    print(f"Found {total} images in {storage_path}")

    for i, img_path in enumerate(image_files, 1):
        thumb_path = thumbnail_path_from_original(img_path)
        if thumb_path.exists():
            skipped += 1
            continue

        result = generate_thumbnail(str(img_path))
        if result:
            created += 1
        else:
            failed += 1

        if i % 50 == 0:
            print(f"  Progress: {i}/{total} (created={created}, skipped={skipped}, failed={failed})")

    print(f"\nDone! Total={total}, Created={created}, Skipped={skipped}, Failed={failed}")


if __name__ == "__main__":
    main()
