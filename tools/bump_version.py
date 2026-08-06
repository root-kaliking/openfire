#!/usr/bin/env python3
"""Bump project.godot's config/version and print the new version.

Usage: bump_version.py [patch|minor|major]   (default: patch)

Shared by the release CI workflow. The Stop-hook rebuild.sh bumps the patch
component inline; this script adds minor/major bumps and is the single source of
truth for the version string format.

Also syncs export_presets.cfg Android preset version/code + version/name so the
APK's android:versionCode and android:versionName stay in sync with the game.
"""
import re
import sys

KIND = sys.argv[1] if len(sys.argv) > 1 else "patch"
PATH = "project.godot"
PRESETS = "export_presets.cfg"

src = open(PATH).read()
m = re.search(r'config/version="(\d+)\.(\d+)\.(\d+)"', src)
if not m:
    sys.exit("bump_version: no config/version in project.godot")

major, minor, patch = (int(x) for x in m.groups())
if KIND == "major":
    major, minor, patch = major + 1, 0, 0
elif KIND == "minor":
    minor, patch = minor + 1, 0
elif KIND == "patch":
    patch += 1
else:
    sys.exit(f"bump_version: unknown bump kind {KIND!r} (use patch|minor|major)")

new = f"{major}.{minor}.{patch}"
src = src[: m.start()] + f'config/version="{new}"' + src[m.end():]
open(PATH, "w").write(src)

# --- Sync Android export preset version ---
# Android versionCode = MAJOR*10000 + MINOR*100 + PATCH
version_code = major * 10000 + minor * 100 + patch
try:
    ps = open(PRESETS).read()
    # preset.3 is Android (name="Android"); update version/code and version/name
    ps_new, n1 = re.subn(
        r'(\[preset\.3\.options\][\s\S]*?version/code=)\d+',
        lambda mm: mm.group(1) + str(version_code),
        ps, count=1,
    )
    ps_new, n2 = re.subn(
        r'(\[preset\.3\.options\][\s\S]*?version/name=")[^"]*(")',
        lambda mm: mm.group(1) + new + mm.group(2),
        ps_new, count=1,
    )
    if n1 or n2:
        open(PRESETS, "w").write(ps_new)
except FileNotFoundError:
    pass

print(new)
