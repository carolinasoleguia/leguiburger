"""Verify the preserved skill import without running any imported helper."""
import hashlib
import json
from pathlib import Path

root = Path(__file__).resolve().parent.parent / "skills"
manifest = json.loads((root / "import-manifest.json").read_text(encoding="utf-8"))
failures = []
for name, expected in manifest["sha256"].items():
    path = (root / name).resolve()
    if not path.is_relative_to(root.resolve()) or not path.is_file():
        failures.append(name)
    elif hashlib.sha256(path.read_bytes()).hexdigest() != expected:
        failures.append(name)
if failures:
    raise SystemExit("Skill import differs from its manifest: " + ", ".join(failures))
print(f"Skills verified: {len(manifest['sha256'])} preserved files, including unslop.")
