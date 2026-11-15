GO ?= go
APP := build/SuperGit.app

.PHONY: build app run demo test check e2e-build native-check lobe-build lobe-run lobe-e2e-build icns icons

build:
	@mkdir -p build
	$(GO) build -o build/supergit ./cmd/supergit

# The drawings ship as full-bleed squares; the macOS icon needs them inside the
# system's rounded-square grid. Re-run "make icons" (and then "make icns") after
# replacing the artwork. ICONS_DIR points at the folder holding
# Light.iconset/Dark.iconset.
ICONS_DIR ?= $(HOME)/Downloads/GitApp_Icons
icons:
	$(GO) run ./tools/icongen -src "$(ICONS_DIR)/Light.iconset/icon_512x512@2x.png" -out internal/appicon/icons/app-light.png
	$(GO) run ./tools/icongen -src "$(ICONS_DIR)/Dark.iconset/icon_512x512@2x.png" -out internal/appicon/icons/app-dark.png

# The .icns in packaging/ is generated from the embedded app artwork; run
# "make icns" after changing internal/appicon/icons/app-light.png.
icns:
	./scripts/make-icns.sh

# The bundle version comes from the nearest tag and the commit count (see
# tools/appversion); outside a checkout the values already in the plist stay.
# The source plist is never rewritten, so a local build leaves no diff.
app: build
	@mkdir -p "$(APP)/Contents/MacOS" "$(APP)/Contents/Resources"
	cp build/supergit "$(APP)/Contents/MacOS/supergit"
	$(GO) run ./tools/appversion -in packaging/Info.plist -out "$(APP)/Contents/Info.plist"
	cp packaging/SuperGit.icns "$(APP)/Contents/Resources/SuperGit.icns"
	codesign --force --sign - "$(APP)"
	codesign --verify --verbose=2 "$(APP)"

run:
	$(GO) run ./cmd/supergit $(if $(REPO),"$(REPO)")

demo: app
	@sh scripts/create-demo.sh > build/demo-root.txt
	@read -r repo < build/demo-root.txt; open -n "$(APP)" --args "$$repo"

test:
	$(GO) test ./...

check:
	$(GO) test ./...
	$(GO) test -race ./...
	$(GO) vet ./...

e2e-build:
	@mkdir -p build
	$(GO) build -tags e2e -o build/supergit-e2e ./cmd/supergit
	$(GO) build -o build/mygo-automation github.com/egoist/mygo/plugins/automation/cmd/mygo-automation

# Experimental Lobe UI preview. Independent of cmd/supergit; not a release build.
lobe-build:
	@mkdir -p build
	$(GO) build -o build/supergit-lobe ./cmd/supergit-lobe

lobe-run:
	$(GO) run ./cmd/supergit-lobe $(if $(REPO),"$(REPO)")

lobe-e2e-build:
	@mkdir -p build
	$(GO) build -tags e2e -o build/supergit-lobe-e2e ./cmd/supergit-lobe
	$(GO) build -o build/mygo-automation github.com/egoist/mygo/plugins/automation/cmd/mygo-automation

native-check: e2e-build
	python3 -I scripts/check-native-workflows.py
	python3 -I scripts/check-native-workflows.py --diff
	python3 -I scripts/check-native-workflows.py --watch
	python3 -I scripts/check-native-workflows.py --history
	python3 -I scripts/check-native-workflows.py --keyboard
	python3 -I scripts/check-native-workflows.py --files
	python3 -I scripts/check-native-workflows.py --commit-files
	python3 -I scripts/check-native-workflows.py --stash
	python3 -I scripts/check-native-workflows.py --tags
