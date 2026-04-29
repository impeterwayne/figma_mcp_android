# Source Tree Analysis

## Overview

This document outlines the directory structure of the **figma-mcp-go** monorepo, covering only version-controlled files.

## Directory Tree

```text
figma-mcp-go/
├── cmd/
│   └── figma-mcp-go/
│       └── main.go              # Go entry point (flag parsing, startup)
├── internal/
│   ├── bridge.go                # WebSocket bridge to Figma Plugin
│   ├── bridge_test.go
│   ├── election.go              # Leader/Follower election logic
│   ├── election_test.go
│   ├── follower.go              # Follower HTTP proxy to Leader
│   ├── follower_test.go
│   ├── helpers_test.go
│   ├── leader.go                # Leader HTTP server (/ws, /ping, /rpc)
│   ├── leader_test.go
│   ├── node.go                  # Role-based request router
│   ├── node_test.go
│   ├── schema.go                # MCP tool definitions (names, schemas)
│   ├── schema_test.go
│   ├── tools.go                 # Tool registration + save_screenshots logic
│   ├── tools_handler_test.go
│   ├── tools_read.go            # Read tool registration aggregator
│   ├── tools_read_document.go   # Document/node inspection tools
│   ├── tools_read_export.go     # Screenshot/PDF export tools
│   ├── tools_read_export_test.go
│   ├── tools_read_styles.go     # Style/variable/component read tools
│   ├── tools_schema_test.go
│   ├── tools_test.go
│   ├── tools_write.go           # Write tool registration aggregator
│   ├── tools_write_components.go
│   ├── tools_write_create.go
│   ├── tools_write_modify.go
│   ├── tools_write_page.go
│   ├── tools_write_prototype.go
│   ├── tools_write_styles.go
│   ├── tools_write_variables.go
│   ├── types.go                 # Shared types (BridgeRequest/Response, Role)
│   ├── types_test.go
│   └── prompts/                 # MCP prompt strategies
│       ├── prompts.go
│       ├── prompts_test.go
│       ├── read_design_strategy.go
│       ├── design_strategy.go
│       ├── text_replacement_strategy.go
│       ├── annotation_conversion_strategy.go
│       ├── swap_overrides_instances.go
│       ├── reaction_to_connector_strategy.go
│       ├── style_audit_strategy.go
│       ├── bulk_rename_strategy.go
│       ├── design_token_generation_strategy.go
│       ├── generate_color_palette.go
│       ├── generate_type_scale.go
│       └── generate_component_variants.go
├── plugin/
│   ├── manifest.json            # Figma plugin manifest
│   ├── package.json             # Plugin Node dependencies
│   ├── bun.lock                 # Bun lockfile
│   ├── vite.config.ts           # Vite config for UI (→ index.html)
│   ├── vite.config.main.ts      # Vite config for sandbox (→ code.js)
│   ├── svelte.config.js         # Svelte configuration
│   ├── tsconfig.json            # TypeScript configuration
│   └── src/
│       ├── main.ts              # Plugin sandbox entry (request dispatch)
│       ├── global.d.ts          # Global type declarations
│       ├── read-handlers.ts     # Read handler chain aggregator
│       ├── read-document.ts     # Document/node read handlers
│       ├── read-styles.ts       # Style/variable read handlers
│       ├── read-styles.test.ts
│       ├── read-export.ts       # Export handlers
│       ├── write-handlers.ts    # Write handler chain aggregator
│       ├── write-create.ts      # Node creation handlers
│       ├── write-create.test.ts
│       ├── write-modify.ts      # Node modification handlers
│       ├── write-modify.test.ts
│       ├── write-styles.ts      # Style CRUD handlers
│       ├── write-variables.ts   # Variable handlers
│       ├── write-components.ts  # Component handlers
│       ├── write-components.test.ts
│       ├── write-prototype.ts   # Prototype handlers
│       ├── write-prototype.test.ts
│       ├── write-page.ts        # Page handlers
│       ├── write-page.test.ts
│       ├── write-helpers.ts     # Shared write utilities
│       ├── write-helpers.test.ts
│       ├── write-effects.test.ts
│       ├── serializers.ts       # Figma node → JSON serializers
│       ├── serializers.test.ts
│       └── ui/
│           ├── App.svelte       # Main UI component
│           ├── index.html       # HTML scaffold for Vite
│           └── main.ts          # Svelte app bootstrap
├── npm/
│   ├── package.json             # NPM distribution manifest
│   └── bin/
│       └── run.js               # Platform-aware binary launcher
├── docs/                        # Project documentation
├── openspec/                    # OpenSpec change management
├── go.mod                       # Go module definition
├── go.sum                       # Go dependency checksums
├── Makefile                     # Build/test/coverage commands
├── server.json                  # MCP server metadata for registries
├── glama.json                   # Glama MCP registry metadata
├── LICENSE                      # MIT license
└── .gitignore
```

## Entry Points

| Part          | Entry Point                | Build Output |
|---------------|----------------------------|--------------|
| Go Backend    | `cmd/figma-mcp-go/main.go` | `bin/figma-mcp-go` binary (git-ignored) |
| Plugin UI     | `plugin/src/ui/main.ts`    | `plugin/dist/index.html` (git-ignored) |
| Plugin Core   | `plugin/src/main.ts`       | `plugin/dist/code.js` (git-ignored) |
| NPM CLI       | `npm/bin/run.js`           | Spawns Go binary |

## File Size Highlights

| File | Size | Notes |
|------|------|-------|
| `internal/schema.go` | ~30 KB | All MCP tool definitions |
| `internal/schema_test.go` | ~51 KB | Largest test file |
| `internal/tools_write_modify.go` | ~21 KB | Largest tool implementation |
| `plugin/src/write-modify.test.ts` | ~26 KB | Largest plugin test |
| `plugin/src/serializers.test.ts` | ~20 KB | Serializer tests |
| `plugin/src/write-modify.ts` | ~18 KB | Largest plugin handler |
| `plugin/src/read-document.ts` | ~17 KB | Document read handler |
| `plugin/src/ui/App.svelte` | ~13 KB | UI component (includes styles) |
