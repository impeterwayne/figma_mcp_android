.PHONY: test test-go test-ts coverage coverage-go coverage-go-html coverage-ts build build-go build-ts build-npm stage-npm-binaries stage-npm-runtime validate-npm-pack clean-npm-bin

NPM_VERSION := $(shell node -p "require('./npm/package.json').version")
GO_LDFLAGS := -X main.version=$(NPM_VERSION)
NPM_BIN := npm/bin
NPM_PLATFORMS := darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 windows-amd64 windows-arm64

ifeq ($(OS),Windows_NT)
MKDIR_P = powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path $(subst /,\\,$(1)) | Out-Null"
RM_RF = powershell -NoProfile -Command "if (Test-Path -LiteralPath '$(subst /,\\,$(1))') { Remove-Item -LiteralPath '$(subst /,\\,$(1))' -Recurse -Force }"
COPY_DIR = powershell -NoProfile -Command "Copy-Item -LiteralPath '$(subst /,\\,$(1))' -Destination '$(subst /,\\,$(2))' -Recurse -Force"
SET_GO_ENV = set GOOS=$(1)&& set GOARCH=$(2)&&
else
MKDIR_P = mkdir -p $(1)
RM_RF = rm -rf $(1)
COPY_DIR = cp -R $(1) $(2)
SET_GO_ENV = GOOS=$(1) GOARCH=$(2)
endif

build: build-go build-ts

build-go:
	go build -ldflags "$(GO_LDFLAGS)" -o bin/figma-mcp-android.exe ./cmd/figma-mcp-android

build-npm: stage-npm-binaries stage-npm-runtime

stage-npm-binaries:
	$(call MKDIR_P,$(NPM_BIN)/darwin-amd64)
	$(call MKDIR_P,$(NPM_BIN)/darwin-arm64)
	$(call MKDIR_P,$(NPM_BIN)/linux-amd64)
	$(call MKDIR_P,$(NPM_BIN)/linux-arm64)
	$(call MKDIR_P,$(NPM_BIN)/windows-amd64)
	$(call MKDIR_P,$(NPM_BIN)/windows-arm64)
	$(call SET_GO_ENV,darwin,amd64) go build -ldflags "$(GO_LDFLAGS)" -o $(NPM_BIN)/darwin-amd64/figma-mcp-android ./cmd/figma-mcp-android
	$(call SET_GO_ENV,darwin,arm64) go build -ldflags "$(GO_LDFLAGS)" -o $(NPM_BIN)/darwin-arm64/figma-mcp-android ./cmd/figma-mcp-android
	$(call SET_GO_ENV,linux,amd64) go build -ldflags "$(GO_LDFLAGS)" -o $(NPM_BIN)/linux-amd64/figma-mcp-android ./cmd/figma-mcp-android
	$(call SET_GO_ENV,linux,arm64) go build -ldflags "$(GO_LDFLAGS)" -o $(NPM_BIN)/linux-arm64/figma-mcp-android ./cmd/figma-mcp-android
	$(call SET_GO_ENV,windows,amd64) go build -ldflags "$(GO_LDFLAGS)" -o $(NPM_BIN)/windows-amd64/figma-mcp-android.exe ./cmd/figma-mcp-android
	$(call SET_GO_ENV,windows,arm64) go build -ldflags "$(GO_LDFLAGS)" -o $(NPM_BIN)/windows-arm64/figma-mcp-android.exe ./cmd/figma-mcp-android

stage-npm-runtime:
	$(call RM_RF,$(NPM_BIN)/svg2vectordrawable_runtime)
	$(call MKDIR_P,$(NPM_BIN))
	$(call COPY_DIR,internal/svg2vectordrawable_runtime,$(NPM_BIN)/svg2vectordrawable_runtime)

validate-npm-pack: build-npm
	cd npm && npm pack --dry-run

clean-npm-bin:
	$(foreach platform,$(NPM_PLATFORMS),$(call RM_RF,$(NPM_BIN)/$(platform))
	)$(call RM_RF,$(NPM_BIN)/svg2vectordrawable_runtime)

build-ts:
	cd plugin && bun run build

test: test-go test-ts

test-go:
	go test ./...

test-ts:
	cd plugin && bun test

coverage: coverage-go coverage-ts

coverage-go:
	go test -coverprofile=bin/coverage.out ./... && go tool cover -func=bin/coverage.out

coverage-ts:
	cd plugin && bun test --coverage
