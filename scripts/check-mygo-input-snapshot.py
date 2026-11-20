#!/usr/bin/env python3
"""Verify the pinned MyGo compatibility snapshot before checks/build review."""
import hashlib
import json
from pathlib import Path

root = Path(__file__).resolve().parent.parent
sdk = root / "third_party/mygo"
manifest = json.loads((sdk / "SUPERGIT_UPSTREAM.json").read_text())
assert manifest["version"] == "v0.8.4-0.20261108082913-9233663736ca", "review a new upstream version explicitly"
assert "replace github.com/egoist/mygo => ./third_party/mygo" in (root / "go.mod").read_text(), "SDK compatibility replacement missing"
expected = manifest["files"]
actual = {str(p.relative_to(sdk)) for p in sdk.rglob("*") if p.is_file()}
assert actual == set(expected) | {"SUPERGIT_UPSTREAM.json", "SUPERGIT_PATCH.md"}, "SDK snapshot file set changed"
for name, digest in expected.items():
    assert hashlib.sha256((sdk / name).read_bytes()).hexdigest() == digest, f"SDK snapshot drift: {name}"
print(f"PASS: pinned MyGo snapshot ({len(expected)} files; two runtime compatibility edits)")
