from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[4]
SMOKE = ROOT / "apps" / "extension" / "chrome" / "tools" / "service_worker_smoke.mjs"


def fail(message: str) -> None:
    print(message, file=sys.stderr)
    raise SystemExit(1)


source = SMOKE.read_text(encoding="utf-8")
lines = source.splitlines()

for line in lines:
    if re.search(r"\b(accessToken|refreshToken)\s*:\s*headers\.get\(", line):
        fail("chrome_service_worker_smoke_sensitive_ak_exposed")

for pattern in (
    r"messageResult\.session\.(accessToken|refreshToken|proxyToken)\b",
    r"\bbootstrap\.(accessToken|refreshToken)\b",
    r"\blatestBootstrap\.(accessToken|refreshToken)\b",
):
    if re.search(pattern, source):
        fail("chrome_service_worker_smoke_sensitive_ak_exposed")

print("chrome_service_worker_smoke_sensitive_ak_ok")
