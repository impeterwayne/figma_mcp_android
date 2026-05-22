# API Contracts - Root (Backend)

## Overview
This document details the complete set of tools exposed by the Go Model Context Protocol (MCP) server. These tools allow LLMs to directly interact with the Figma environment via the bridging plugin.

## Document & Node Inspection
- `get_document`: Get the full node tree of the current page (not the whole file — only the active page). Returns all nodes recursively and can be very large. Prefer get_design_context for exploration or when token efficiency matters.
- `get_pages`: List all pages in the document with their IDs and names. Lightweight alternative to get_document.
- `get_metadata`: Get metadata about the current Figma document: file name, pages, current page.
- `get_selection`: Get the nodes currently selected in Figma. Returns an empty array if nothing is selected.
- `get_node`: Get a single node by ID with full detail.
- `get_nodes_info`: Get full details for multiple nodes by ID in one round-trip.
- `get_design_context`: Get a depth-limited, token-efficient tree of the current selection or page. Use this instead of get_document when exploring large files.

## Search & Scan
- `search_nodes`: Search for nodes by name substring and/or type within a subtree.
- `scan_text_nodes`: Scan all TEXT nodes in a subtree and return their content.
- `scan_nodes_by_types`: Find all nodes of specific types in a subtree, regardless of name.
- `find_replace_text`: Find and replace text content across all TEXT nodes in a subtree.

## Context & Reactions
- `get_reactions`: Get the prototype reactions defined on a node. Returns an array of reaction objects.
- `set_reactions`: Set or replace prototype reactions on a node (triggers, actions, navigation).
- `remove_reactions`: Remove all or specific prototype reactions from a node.
- `get_viewport`: Get the current Figma viewport: scroll center, zoom level, and visible bounds.
- `get_fonts`: List all fonts used in the current page, sorted by usage frequency.
- `get_annotations`: Get dev-mode annotations in the current document or scoped to a specific node.

## Styles & Variables
- `get_styles`: Get all local styles in the document (paint, text, effect, and grid).
- `get_variable_defs`: Get all local variable definitions: collections, modes, and values.
- `get_local_components`: Get all components defined in the current Figma file.
- `export_tokens`: Export all design tokens (variables and paint styles) as JSON or CSS custom properties.
- `apply_style_to_node`: Apply an existing local style (paint, text, effect, or grid) to a node, linking the node to that style.
- `bind_variable_to_node`: Bind a local variable to a node property so the property is driven by the variable's value.
- `create_effect_style`: Create a new local effect style (drop shadow, inner shadow, or blur).
- `create_grid_style`: Create a new local layout grid style.
- `create_paint_style`: Create a new local paint style with a solid fill color.
- `create_text_style`: Create a new local text style (typography preset).
- `create_variable`: Create a new variable (design token) inside an existing collection.
- `create_variable_collection`: Create a new local variable collection with an optional initial mode name.
- `add_variable_mode`: Add a new mode to an existing variable collection.
- `delete_style`: Delete a style (paint, text, effect, or grid) by its ID.
- `delete_variable`: Delete a single variable or an entire collection.
- `set_variable_value`: Set a variable's value for a specific mode.
- `update_paint_style`: Update an existing paint style's name, color, or description.

## Exports
- `get_screenshot`: Export a screenshot of one or more nodes as base64-encoded image data (held in memory).
- `save_screenshots`: Export screenshots for multiple nodes and write them to the local filesystem.

## Node Creation
- `create_frame`: Create a new frame on the current page or inside a parent node.
- `create_rectangle`: Create a new rectangle on the current page or inside a parent node.
- `create_ellipse`: Create a new ellipse (circle/oval) on the current page or inside a parent node.
- `create_text`: Create a new text node on the current page or inside a parent node.
- `create_component`: Convert an existing FRAME node into a reusable COMPONENT.
- `create_section`: Create a Figma Section node on the current page.
- `import_image`: Import a base64-encoded image into Figma as a rectangle with an image fill.

## Node Modification
- `set_text`: Update the text content of an existing TEXT node.
- `set_fills`: Set the fill color on a single node. Use mode='append' to stack a new fill.
- `set_strokes`: Set the stroke color and weight on a single node.
- `set_opacity`: Set the opacity of one or more nodes (0 = fully transparent, 1 = fully opaque).
- `set_corner_radius`: Set corner radius on one or more nodes.
- `set_auto_layout`: Set or update auto-layout (flex) properties on an existing frame.
- `set_blend_mode`: Set the blend mode of one or more nodes (e.g. MULTIPLY, SCREEN, OVERLAY).
- `set_visible`: Show or hide one or more nodes by setting their visibility.
- `set_constraints`: Set layout constraints (pinning behaviour) on one or more nodes relative to their parent.
- `set_effects`: Apply one or more effects (drop shadow, inner shadow, layer blur, background blur) directly to a node.

## Transform & Structure
- `move_nodes`: Move one or more nodes to an absolute canvas position.
- `resize_nodes`: Resize one or more nodes. Provide width, height, or both.
- `rotate_nodes`: Rotate one or more nodes to an absolute angle in degrees.
- `rename_node`: Rename a single node by ID.
- `batch_rename_nodes`: Rename multiple nodes using find/replace, regex substitution, or prefix/suffix addition.
- `clone_node`: Clone an existing node, optionally repositioning it or placing it in a new parent.
- `delete_nodes`: Delete one or more nodes. This cannot be undone via MCP — use with care.
- `reorder_nodes`: Change the z-order (layer stack position) of one or more nodes.
- `group_nodes`: Group two or more nodes into a GROUP.
- `ungroup_nodes`: Ungroup one or more GROUP nodes, moving their children to the parent and removing the group.
- `reparent_nodes`: Move one or more nodes to a different parent frame, group, or section.

## State & Instance Management
- `lock_nodes`: Lock one or more nodes to prevent accidental edits in Figma.
- `unlock_nodes`: Unlock one or more nodes, allowing them to be edited again.
- `swap_component`: Swap the main component of an existing INSTANCE node.
- `detach_instance`: Detach one or more component instances, converting them to plain frames.

## Page Management
- `add_page`: Add a new page to the Figma document.
- `delete_page`: Delete a page from the Figma document. Cannot delete the only remaining page.
- `rename_page`: Rename an existing page in the Figma document.
- `navigate_to_page`: Switch the active Figma page. Provide either pageId or pageName.

## Communication Protocol
- Uses standard MCP JSON-RPC over stdio for LLM communication.
- Translates MCP tool calls into custom JSON payloads sent over WebSocket to the Figma Plugin.
