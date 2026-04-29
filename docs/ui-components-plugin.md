# UI Components — Plugin

## Overview

The plugin UI consists of a single Svelte component (`App.svelte`) with no sub-components. See [Component Inventory — Plugin](./component-inventory-plugin.md) for the full breakdown of its responsibilities, layout, and visual design.

## Build Pipeline

The UI is built via `vite.config.ts`:

1. **Entry**: `plugin/src/ui/main.ts` (bootstraps `App.svelte`)
2. **Scaffold**: `plugin/src/ui/index.html` (standard Vite HTML entry)
3. **Plugins**: `@sveltejs/vite-plugin-svelte` + `vite-plugin-singlefile`
4. **Output**: `plugin/dist/index.html` — a single self-contained HTML file with all CSS and JS inlined

This singlefile output is required by Figma's plugin architecture, which loads the UI from a `data:` URL (no separate asset loading).

## Styling

- All styles are scoped within `App.svelte` using Svelte's `<style>` block.
- Global resets applied via `:global(*)` and `:global(body)`.
- Dark theme throughout (`#1e1e1e` background).
- No external CSS frameworks or design systems.
