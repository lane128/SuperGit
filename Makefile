GO ?= go
SOURCE_PKGS ?= ./cmd/... ./internal/... ./tools/...
APP := build/SuperGit.app
APP_BINARY := build/supergit
DESKTOP_APP := build/SuperGit Desktop.app
LOBE_APP := build/SuperGit Lobe Preview.app
FOLDER_DOCUMENT_TYPES := [{"CFBundleTypeName":"Repository Folder","CFBundleTypeRole":"Editor","LSHandlerRank":"Alternate","LSItemContentTypes":["public.folder"]}]

.PHONY: build app release-pack run demo test check e2e-build native-check desktop-app desktop-e2e-build desktop-native-check lobe-build lobe-app lobe-run lobe-e2e-build icns icons

UPDATE_PUBKEY ?=
BUILD_LDFLAGS := $(if $(UPDATE_PUBKEY),-X supergit/internal/updater.DefaultPublicKeyBase64=$(UPDATE_PUBKEY),)

build:
	@mkdir -p "$(dir $(APP_BINARY))"
	$(GO) build $(if $(BUILD_LDFLAGS),-ldflags "$(BUILD_LDFLAGS)",) -o "$(APP_BINARY)" ./cmd/supergit

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
	cp "$(APP_BINARY)" "$(APP)/Contents/MacOS/supergit"
	$(GO) run ./tools/appversion -in packaging/Info.plist -out "$(APP)/Contents/Info.plist"
	plutil -insert CFBundleDocumentTypes -json '$(FOLDER_DOCUMENT_TYPES)' "$(APP)/Contents/Info.plist"
	$(if $(UPDATE_PUBKEY),plutil -insert SuperGitUpdatePublicKey -string "$(UPDATE_PUBKEY)" "$(APP)/Contents/Info.plist",)
	cp packaging/SuperGit.icns "$(APP)/Contents/Resources/SuperGit.icns"
	cp internal/lobedesk/fonts/OFL.txt "$(APP)/Contents/Resources/Font-Licenses.txt"
	codesign --force --sign - "$(APP)"
	codesign --verify --verbose=2 "$(APP)"

# Package a signed update archive and update.json manifest. Pass UPDATE_PRIVKEY
# pointing to the local Ed25519 private key file (never commit private keys).
RELEASE_OUT ?= build/release
RELEASE_BASE_URL ?=
RELEASE_NOTES_ZH ?=
RELEASE_NOTES_EN ?=
release-pack: app
	@test -n "$(UPDATE_PRIVKEY)" || (echo "UPDATE_PRIVKEY is required (generate with: go run ./tools/releasepack -genkey -priv path/to/update.key -pub path/to/update.pub)" >&2; exit 2)
	$(GO) run ./tools/releasepack -app "$(APP)" -out "$(RELEASE_OUT)" -priv "$(UPDATE_PRIVKEY)" \
		$(if $(RELEASE_BASE_URL),-base-url "$(RELEASE_BASE_URL)",) \
		$(if $(RELEASE_NOTES_ZH),-notes-zh "$(RELEASE_NOTES_ZH)",) \
		$(if $(RELEASE_NOTES_EN),-notes-en "$(RELEASE_NOTES_EN)",)

desktop-app:
	@mkdir -p "$(DESKTOP_APP)/Contents/MacOS" "$(DESKTOP_APP)/Contents/Resources"
	$(GO) build -tags legacydesktop -o "$(DESKTOP_APP)/Contents/MacOS/supergit" ./cmd/supergit
	$(GO) run ./tools/appversion -in packaging/Info.plist -out "$(DESKTOP_APP)/Contents/Info.plist"
	plutil -replace CFBundleIdentifier -string dev.supergit.desktop.local "$(DESKTOP_APP)/Contents/Info.plist"
	plutil -replace CFBundleName -string "SuperGit Desktop" "$(DESKTOP_APP)/Contents/Info.plist"
	plutil -replace CFBundleDisplayName -string "SuperGit Desktop" "$(DESKTOP_APP)/Contents/Info.plist"
	cp packaging/SuperGit.icns "$(DESKTOP_APP)/Contents/Resources/SuperGit.icns"
	codesign --force --sign - "$(DESKTOP_APP)"
	codesign --verify --verbose=2 "$(DESKTOP_APP)"

run:
	$(GO) run ./cmd/supergit $(if $(REPO),"$(REPO)")

demo: app
	@sh scripts/create-demo.sh > build/demo-root.txt
	@read -r repo < build/demo-root.txt; open -n "$(APP)" --args "$$repo"

test:
	$(GO) test $(SOURCE_PKGS)

# Rendering-heavy lobedesk tests exceed Go's default 10-minute per-package
# race timeout on a developer laptop; the limit is for a hang, not a budget.
RACE_TIMEOUT ?= 20m

check:
	python3 scripts/check-mygo-input-snapshot.py
	$(GO) test github.com/egoist/mygo/ui $(SOURCE_PKGS)
	$(GO) test -race -timeout $(RACE_TIMEOUT) $(SOURCE_PKGS)
# Exact allocation counts are measured by the full ordinary SDK suite above.
# Race instrumentation changes those counts; keep all other SDK race tests.
	$(GO) test -race -timeout $(RACE_TIMEOUT) github.com/egoist/mygo/ui -skip '^(TestDiffFrameAllocs|TestPathContains)$$'
	$(GO) vet github.com/egoist/mygo/ui $(SOURCE_PKGS)

e2e-build:
	@mkdir -p build
	$(GO) build -tags e2e -o build/supergit-e2e ./cmd/supergit
	$(GO) build -o build/mygo-automation github.com/egoist/mygo/plugins/automation/cmd/mygo-automation

# Experimental Lobe UI preview. Independent of cmd/supergit; not a release build.
lobe-build:
	@mkdir -p build
	$(GO) build -o build/supergit-lobe ./cmd/supergit-lobe

# Bundle the same preview for Finder without reusing the installed app's identity.
lobe-app: lobe-build
	@mkdir -p "$(LOBE_APP)/Contents/MacOS" "$(LOBE_APP)/Contents/Resources"
	cp build/supergit-lobe "$(LOBE_APP)/Contents/MacOS/supergit-lobe"
	$(GO) run ./tools/appversion -in packaging/Info.plist -out "$(LOBE_APP)/Contents/Info.plist"
	plutil -replace CFBundleExecutable -string supergit-lobe "$(LOBE_APP)/Contents/Info.plist"
	plutil -replace CFBundleIdentifier -string dev.supergit.lobe.local-preview "$(LOBE_APP)/Contents/Info.plist"
	plutil -replace CFBundleName -string "SuperGit Lobe Preview" "$(LOBE_APP)/Contents/Info.plist"
	plutil -replace CFBundleDisplayName -string "SuperGit Lobe Preview" "$(LOBE_APP)/Contents/Info.plist"
	plutil -insert CFBundleDocumentTypes -json '$(FOLDER_DOCUMENT_TYPES)' "$(LOBE_APP)/Contents/Info.plist"
	cp packaging/SuperGit.icns "$(LOBE_APP)/Contents/Resources/SuperGit.icns"
	codesign --force --sign - "$(LOBE_APP)"
	codesign --verify --verbose=2 "$(LOBE_APP)"

lobe-run:
	$(GO) run ./cmd/supergit-lobe $(if $(REPO),"$(REPO)")

lobe-e2e-build:
	@mkdir -p build
	$(GO) build -tags e2e -o build/supergit-lobe-e2e ./cmd/supergit-lobe
	$(GO) build -o build/mygo-automation github.com/egoist/mygo/plugins/automation/cmd/mygo-automation

native-check:
	python3 -I scripts/check-native-supergit.py

desktop-e2e-build:
	@mkdir -p build
	$(GO) build -tags 'e2e legacydesktop' -o build/supergit-desktop-e2e ./cmd/supergit
	$(GO) build -o build/mygo-automation github.com/egoist/mygo/plugins/automation/cmd/mygo-automation

desktop-native-check: desktop-e2e-build
	python3 -I scripts/check-native-workflows.py
	python3 -I scripts/check-native-workflows.py --diff
	python3 -I scripts/check-native-workflows.py --watch
	python3 -I scripts/check-native-workflows.py --history
	python3 -I scripts/check-native-workflows.py --keyboard
	python3 -I scripts/check-native-workflows.py --files
	python3 -I scripts/check-native-workflows.py --commit-files
	python3 -I scripts/check-native-workflows.py --stash
	python3 -I scripts/check-native-workflows.py --tags
