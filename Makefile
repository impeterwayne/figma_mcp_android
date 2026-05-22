.PHONY: test test-go test-ts coverage coverage-go coverage-go-html coverage-ts build build-go build-ts build-npm stage-npm-binaries stage-npm-runtime validate-npm-pack clean-npm-bin

NPM_VERSION := $(shell node -p "require('./npm/package.json').version")
GO_LDFLAGS := -X main.version=$(NPM_VERSION)
NPM_BIN := npm/bin
NPM_PLATFORMS := darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 windows-amd64 windows-arm64

build: build-go build-ts

build-go:
	go build -ldflags "$(GO_LDFLAGS)" -o bin/figma-mcp-android.exe ./cmd/figma-mcp-android

build-npm: stage-npm-binaries stage-npm-runtime

stage-npm-binaries:
	mkdir -p $(NPM_BIN)/darwin-amd64 $(NPM_BIN)/darwin-arm64 $(NPM_BIN)/linux-amd64 $(NPM_BIN)/linux-arm64 $(NPM_BIN)/windows-amd64 $(NPM_BIN)/windows-arm64
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o $(NPM_BIN)/darwin-amd64/figma-mcp-android ./cmd/figma-mcp-android
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(GO_LDFLAGS)" -o $(NPM_BIN)/darwin-arm64/figma-mcp-android ./cmd/figma-mcp-android
	GOOS=linux GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o $(NPM_BIN)/linux-amd64/figma-mcp-android ./cmd/figma-mcp-android
	GOOS=linux GOARCH=arm64 go build -ldflags "$(GO_LDFLAGS)" -o $(NPM_BIN)/linux-arm64/figma-mcp-android ./cmd/figma-mcp-android
	GOOS=windows GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o $(NPM_BIN)/windows-amd64/figma-mcp-android.exe ./cmd/figma-mcp-android
	GOOS=windows GOARCH=arm64 go build -ldflags "$(GO_LDFLAGS)" -o $(NPM_BIN)/windows-arm64/figma-mcp-android.exe ./cmd/figma-mcp-android

stage-npm-runtime:
	rm -rf $(NPM_BIN)/svg2vectordrawable_runtime
	mkdir -p $(NPM_BIN)
	cp -R internal/svg2vectordrawable_runtime $(NPM_BIN)/svg2vectordrawable_runtime

validate-npm-pack: build-npm
	cd npm && npm pack --dry-run

clean-npm-bin:
	rm -rf $(foreach platform,$(NPM_PLATFORMS),$(NPM_BIN)/$(platform)) $(NPM_BIN)/svg2vectordrawable_runtime

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
