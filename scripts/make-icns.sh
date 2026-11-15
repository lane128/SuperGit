#!/bin/sh

# Builds packaging/SuperGit.icns from the embedded 1024px app icon, so the
# packaged .app shows the same artwork in Finder that the running app shows in
# the Dock. The .icns carries the light drawing: a bundle icon is one file, and
# macOS cannot switch it by appearance; the running app swaps the Dock icon
# itself (the shared state machine lives in internal/appicon, driven by
# internal/desktop/dockicon.go and internal/lobedesk/dockicon.go).
set -eu

cd "$(dirname "$0")/.."
source_png=internal/appicon/icons/app-light.png
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

iconset="$work/SuperGit.iconset"
mkdir -p "$iconset"
for size in 16 32 128 256 512; do
	sips -z "$size" "$size" "$source_png" --out "$iconset/icon_${size}x${size}.png" >/dev/null
	double=$((size * 2))
	sips -z "$double" "$double" "$source_png" --out "$iconset/icon_${size}x${size}@2x.png" >/dev/null
done

mkdir -p packaging
iconutil -c icns "$iconset" -o packaging/SuperGit.icns
echo "wrote packaging/SuperGit.icns"
