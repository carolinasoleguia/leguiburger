"""Check tracked source, never print a secret value or inspect ignored local env files."""
import hashlib
import pathlib
import re
import subprocess
import sys

root = pathlib.Path(__file__).resolve().parent.parent
paths = subprocess.check_output(["git", "ls-files", "--cached", "--others", "--exclude-standard", "-z"], cwd=root).decode().split("\0")
patterns = [
    re.compile(rb"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----"),
    re.compile(rb"\b(?:gh[pousr]_[A-Za-z0-9]{36,}|github_pat_[A-Za-z0-9_]{50,})\b"),
    re.compile(rb"\bAKIA[0-9A-Z]{16}\b"),
    re.compile(rb"postgres(?:ql)?://[^\s:/]+:[^\s@]{8,}@[^\s]+supabase\.(?:co|com)"),
    re.compile(rb"\bAPP_USR-[0-9]{8,}-[A-Za-z0-9-]{20,}"),
]
# These preserved security guides list PEM header patterns, without key material.
# Pin the whole file: any edit removes the exception, and every other secret
# pattern is still scanned even when the documentation matches its known hash.
documented_pem_patterns = {
    "skills/originals/first-import/security/security-senior-secops.md": "80f83d40f631134ab458247433b616e8fbdfdc0c7bdc568e356d4e6902385359",
    "skills/security/security-senior-secops/SKILL.md": "ae542f06bec50b876d3d3b67022712800e9bcbc0af1e0c57b5f591a912d4e8e4",
    "skills/security/security-senior-secops/references/original-profile.md": "80f83d40f631134ab458247433b616e8fbdfdc0c7bdc568e356d4e6902385359",
}
failed = False
for name in sorted(set(paths)):
    if not name:
        continue
    path = root / name
    if not path.is_file():
        continue
    if pathlib.PurePosixPath(name).name == ".env" or (name.startswith(".env.") and name != ".env.example"):
        print(f"Local environment file must not be tracked: {name}")
        failed = True
        continue
    data = path.read_bytes()
    scan_data = data
    if hashlib.sha256(data).hexdigest() == documented_pem_patterns.get(name):
        scan_data = re.sub(rb"(?m)^-----BEGIN (?:RSA |EC )PRIVATE KEY-----\r?$", b"[documented PEM header pattern]", data)
    if any(pattern.search(scan_data) for pattern in patterns):
        print(f"Potential credential in {name}; value redacted")
        failed = True
    if name.startswith("database/migrations/") and b"\r\n" in data:
        print(f"Migration must use LF to preserve its remote checksum: {name}")
        failed = True
if failed:
    sys.exit(1)
print("Source checks passed: no detected credentials, ignored local env excluded, migration LF preserved.")
