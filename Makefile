# Canonical verification gate for kubecom (D17). Agents and CI run `make check`;
# "green" means exactly this passing.
# (Replaces the legacy Travis/protoc Makefile; pb/ codegen is gone per D3/D14.)
.PHONY: check build test vet lint test-envtest keys-doc screencast dist release

check: build test vet lint

# Release build metadata (follows engram/margin). A build has a name and an
# identity: the name is CalVer — `YY.MM.DD` for a stable release (the tag's own
# name), `YY.MM.DD-dev.<sha7>` for a build of main — and the identity is the full
# commit. Channel is stamped only by `dist`, so every local build is on no
# channel and `kubecom update` sends it to the dev channel. The launcher rides
# the same version as the seed the package ships (D295): it is rebuilt and
# version-stamped with every release, so `kubecom --launcher-version` names the
# package the user installed, while `kubecom version` names the build it runs.
SHORT    := $(shell git rev-parse --short=7 HEAD 2>/dev/null)
COMMIT   ?= $(shell git rev-parse HEAD 2>/dev/null)
MODIFIED ?= $(if $(shell git status --porcelain 2>/dev/null),true,)
RELEASE  ?=
CHANNEL  ?= $(if $(RELEASE),stable,dev)
DIST_VERSION = $(or $(RELEASE),$(shell TZ=UTC git show -s --date=format-local:%y.%m.%d --format=%cd HEAD)-$(CHANNEL).$(SHORT))

VERSION_PKG  := github.com/neuroplastio/kubecom/internal/version
LDFLAGS      := -X $(VERSION_PKG).Commit=$(COMMIT) -X $(VERSION_PKG).modified=$(MODIFIED)
DIST_LDFLAGS := -s -w -X $(VERSION_PKG).Commit=$(COMMIT) \
	-X $(VERSION_PKG).Version=$(DIST_VERSION) -X $(VERSION_PKG).Channel=$(CHANNEL) \
	-X $(VERSION_PKG).Date=$(shell TZ=UTC git show -s --format=%cI HEAD)
LAUNCHER_FLAGS := -buildvcs=false -ldflags '$(DIST_LDFLAGS)'
DIST_PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

build:
	go build ./...

# The release artifacts (D18/D20): static (CGO off) bare binaries, one per
# platform — kubecom_<os>_<arch>, what the channel publishes and the launcher
# installs, and kubecom-launcher_<os>_<arch>, the thin wrapper a package installs
# as kubecom. Refuses a dirty tree and a RELEASE that is not a tag at HEAD.
dist:
	@test -z "$(MODIFIED)" || { echo "make dist: the tree has uncommitted changes" >&2; exit 1; }
	@test -z "$(RELEASE)" || test "$$(git rev-parse -q --verify 'refs/tags/$(RELEASE)^{commit}')" = "$(COMMIT)" || \
		{ echo "make dist: RELEASE=$(RELEASE) is not a tag at HEAD" >&2; exit 1; }
	rm -rf dist
	@set -e; for p in $(DIST_PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		echo "dist/kubecom_$${os}_$${arch}"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags '$(DIST_LDFLAGS)' \
			-o dist/kubecom_$${os}_$${arch} ./cmd/kubecom; \
		echo "dist/kubecom-launcher_$${os}_$${arch}"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath $(LAUNCHER_FLAGS) \
			-o dist/kubecom-launcher_$${os}_$${arch} ./cmd/kubecom-launcher; \
	done

# Cut a release: tag HEAD with today's date in UTC, YY.MM.DD, and push it. The
# tag is the whole act — the release workflow builds it, publishes it to the
# stable channel, and makes the GitHub release. Refuses a dirty tree, a HEAD not
# on origin/main, and a day already released (a day has one release).
release:
	@test -z "$(MODIFIED)" || { echo "make release: the tree has uncommitted changes" >&2; exit 1; }
	@git fetch -q --tags origin
	@git merge-base --is-ancestor HEAD origin/main || { echo "make release: HEAD is not on origin/main" >&2; exit 1; }
	@other=$$(git tag --points-at HEAD --list '[0-9][0-9].[0-9][0-9].[0-9][0-9]'); \
	test -z "$$other" || { echo "make release: $(SHORT) is already released as $$other" >&2; exit 1; }
	@set -e; tag=$$(date -u +%y.%m.%d); \
	if git ls-remote --exit-code --tags origin "refs/tags/$$tag" >/dev/null; then \
		echo "make release: $$tag is already released; a day has one release" >&2; exit 1; \
	fi; \
	git tag -a "$$tag" -m "kubecom $$tag"; \
	git push origin "refs/tags/$$tag"; \
	echo "kubecom $$tag is $(SHORT); the release workflow takes it from here"

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run

# Regenerate the committed keybindings reference (docs/keybindings.md) from the
# default keymap (D11). `make check` fails if the doc drifts (TestKeybindingsDoc).
keys-doc:
	go test ./internal/tui/keymap -run TestKeybindingsDoc -update

# Opt-in envtest integration tests (D18): stand up a real kube-apiserver + etcd.
# Not part of `check` — needs control-plane binaries fetched via setup-envtest.
# CI runs this target in its own workflow (.github/workflows/envtest.yml), never
# as part of the `make check` gate: a failed control-plane download is not a code
# failure (D190). Locally:
#   go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.19
ENVTEST_K8S_VERSION ?= 1.31.x
# setup-envtest is a tool, not a module dependency, so it is wherever `go install`
# put it — which is on PATH only if GOPATH/bin is. Look there too rather than
# failing at a command substitution.
SETUP_ENVTEST ?= $(shell command -v setup-envtest 2>/dev/null || printf '%s/bin/setup-envtest' "$$(go env GOPATH)")
test-envtest:
	@set -e; \
	if [ -n "$$KUBEBUILDER_ASSETS" ]; then \
		echo "using KUBEBUILDER_ASSETS=$$KUBEBUILDER_ASSETS"; \
	elif [ -x "$(SETUP_ENVTEST)" ]; then \
		KUBEBUILDER_ASSETS="$$($(SETUP_ENVTEST) use -p path $(ENVTEST_K8S_VERSION))"; \
		export KUBEBUILDER_ASSETS; \
		echo "using KUBEBUILDER_ASSETS=$$KUBEBUILDER_ASSETS"; \
	else \
		echo "make: setup-envtest not found (looked on PATH and at $(SETUP_ENVTEST))." >&2; \
		echo "  go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.19" >&2; \
		echo "or set KUBEBUILDER_ASSETS to a control-plane binary directory." >&2; \
		exit 1; \
	fi; \
	KUBECOM_TEST_ENVTEST=1 go test ./internal/kube/... -count=1

# Record the README screencast (docs/screencast.gif) from the committed tape.
# Not part of `check`: it needs vhs (https://github.com/charmbracelet/vhs), a real
# terminal and a real cluster in the current kubeconfig context — see the header of
# docs/screencast/screencast.tape for what the tour assumes, and D181 for why the recording is
# a human's and not an agent's.
#
# The tape redirects kubecom's XDG config/cache dirs to a throwaway /tmp dir and
# wipes it before every launch, so reruns start from the same welcome screen (the
# remembered pane and menu pins never leak across recordings).
#
# The binary is built here and put first on PATH so the recording is always of this
# checkout, never of a stale `kubecom` someone installed months ago.
#
# The same run writes docs/screencast.cast (CAST-01, D290): tmux pipes kubecom's pane to
# castrec, built here too, and the tour's captions reach it as markers.
BIN_DIR ?= $(CURDIR)/bin
screencast:
	go build -o $(BIN_DIR)/kubecom ./cmd/kubecom
	go build -o $(BIN_DIR)/castrec ./docs/screencast/castrec
	PATH="$(BIN_DIR):$$PATH" vhs docs/screencast/screencast.tape
