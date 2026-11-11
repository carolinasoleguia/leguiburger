"""Check tracked source, never print a secret value or inspect ignored local env files."""
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
    if any(pattern.search(data) for pattern in patterns):
        print(f"Potential credential in {name}; value redacted")
        failed = True
    if name.startswith("database/migrations/") and b"\r\n" in data:
        print(f"Migration must use LF to preserve its remote checksum: {name}")
        failed = True
if failed:
    sys.exit(1)
print("Source checks passed: no detected credentials, ignored local env excluded, migration LF preserved.")
