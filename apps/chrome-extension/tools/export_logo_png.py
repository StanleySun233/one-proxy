from pathlib import Path
import subprocess
import sys
import tempfile

import cv2


ROOT = Path(__file__).resolve().parents[1]
ASSET_DIR = ROOT / "assets" / "logo"
SVG_PATH = ASSET_DIR / "logo.svg"
PNG_PATH = ASSET_DIR / "logo.png"
ICON_SIZES = [16, 32, 48, 128]


def main() -> int:
    if not SVG_PATH.exists():
        raise FileNotFoundError(SVG_PATH)
    ASSET_DIR.mkdir(parents=True, exist_ok=True)

    with tempfile.TemporaryDirectory() as tmpdir:
        tmp_png = Path(tmpdir) / "logo-raster.png"
        subprocess.run(
            [
                "convert",
                "-background",
                "none",
                str(SVG_PATH),
                str(tmp_png),
            ],
            check=True,
        )
        image = cv2.imread(str(tmp_png), cv2.IMREAD_UNCHANGED)
        if image is None:
            raise RuntimeError("failed to load rasterized logo")
        if not cv2.imwrite(str(PNG_PATH), image):
            raise RuntimeError("failed to write png")
        for size in ICON_SIZES:
            resized = cv2.resize(image, (size, size), interpolation=cv2.INTER_AREA)
            icon_path = ASSET_DIR / f"logo-{size}.png"
            if not cv2.imwrite(str(icon_path), resized):
                raise RuntimeError(f"failed to write {icon_path}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
