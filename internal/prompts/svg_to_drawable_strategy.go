package prompts

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func addSVGToDrawableStrategy(s *server.MCPServer) {
	s.AddPrompt(mcp.NewPrompt("svg_to_drawable_strategy",
		mcp.WithPromptDescription("Export Figma vector assets and convert them into Android VectorDrawable XML"),
	), func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return mcp.NewGetPromptResult(
			"Export Figma vector assets and convert them into Android VectorDrawable XML",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleUser,
					mcp.NewTextContent(`# SVG to Android VectorDrawable Strategy

Turn icons and vector art from Figma into `+"`res/drawable/ic_*.xml`"+` VectorDrawable files.

## Decide first: vector or raster?

Convert to VectorDrawable when the asset is flat vector art — icons, logos, simple
illustrations, solid or linear/radial gradient fills.

Export PNG/WebP into `+"`res/drawable-{m,h,xh,xxh,xxxh}dpi/`"+` instead when the node contains:
- raster image fills or exported bitmaps
- blur / drop-shadow / any layer effect (VectorDrawable has no effect support)
- blend modes other than normal
- dozens of paths for a detailed illustration (large pathData inflates on every draw)

Say which route you picked and why before exporting.

## Steps

1. **Find the assets**
   - `+"`search_nodes`"+` by name (e.g. "ic", "icon") or `+"`scan_nodes_by_types`"+` with
     types ["COMPONENT", "INSTANCE", "VECTOR"] to list icon nodes in a subtree.
   - Export the *icon container* (the COMPONENT/FRAME, usually 24x24), not the individual
     VECTOR children — child paths lose the artboard viewport and come out mis-scaled.

2. **Export SVG to disk** — the converter takes a file path only
   - `+"`save_screenshots({items: [{nodeId, outputPath, format: 'SVG'}, ...]})`"+` — one call for
     all icons — writes the .svg files and returns their paths.
   - `+"`get_screenshot(format='SVG')`"+` is the wrong tool here: it returns base64 that
     `+"`convert_svg_to_android_drawable`"+` will not accept, and it burns tokens on markup nobody
     reads. If an SVG reaches you as text or base64, write it to a file yourself first.

3. **Convert** — one call for the whole set
   - Pass every asset in `+"`items`"+`, each with `+"`outputPath`"+` set, so the XML is written
     straight to the project and never round-trips through the model:
     `+"`{items: [{svgPath: '<tmp>/ic_star.svg', outputPath: 'app/src/main/res/drawable/ic_star.xml'}, ...]}`"+`
     Items convert in parallel and the response lists per-item results in the same order,
     so a batch is both faster and cheaper than one call per icon. Use the top-level
     `+"`svgPath`"+`/`+"`outputPath`"+` form only for a single asset; the two forms cannot be mixed.
   - One failing item does not stop the others: the response reports
     `+"`converted`/`failed`"+` counts and an `+"`error`"+` string on each item that failed. Re-export
     and retry just those, and say which icons did not make it.
   - Every `+"`svgPath`"+` in a batch must be distinct, and so must every `+"`outputPath`"+` — duplicates
     are rejected before anything is written rather than silently overwriting a drawable.
   - Name files `+"`ic_<snake_case>.xml`"+`: lowercase a-z, 0-9 and _ only, first char a letter.
     Sanitize the Figma layer name ("Icon / Arrow-Left 2" -> ic_arrow_left_2.xml); an invalid
     resource name is an aapt build error, not a warning.
   - The exported .svg is always deleted once the conversion succeeds — it is an intermediate,
     and the drawable XML is the artifact that ships. There is no opt-out, so export to a temp
     path rather than over an .svg you want to keep, and never call the converter twice on the
     same svgPath: re-export from Figma if you need another pass.

4. **Options that actually matter**
   - `+"`floatPrecision`"+` (default 6, the maximum): leave it alone. The default is correct for
     every viewBox size, including normalized `+"`viewBox=\"0 0 1 1\"`"+` art where a low precision
     visibly distorts curves. Lower it only to shrink a large illustration whose pathData is
     genuinely too long. Curves that still look wrong after conversion are a bad export (a
     child VECTOR instead of the container, or an effect VectorDrawable cannot represent),
     not a precision problem.
   - `+"`fillBlack: true`"+`: use when the source SVG has paths with no fill attribute — otherwise
     they convert to an invisible drawable. Symptom: valid XML, nothing renders. Leave it false
     for stroke-only artwork, or the strokes pick up a spurious black fill.
   - `+"`tint`"+`: sets `+"`android:tint`"+` on the root. Use for single-color icons that must follow a
     fixed color. Leave unset if the icon is tinted at the call site.
   - `+"`xmlTag`"+`: leave false; res XML does not need the declaration.
   - `+"`cache`"+`: leave true — repeated identical icons convert once.
   - In batch mode these are the defaults for the whole set; override per item only where
     an icon genuinely differs (e.g. one stroke-only asset that must not get `+"`fillBlack`"+`).

5. **Verify every output** — the tool returns XML, not correctness:
   - Root is `+"`<vector>`"+` with non-zero `+"`viewportWidth`/`viewportHeight`"+` and
     `+"`android:width`/`android:height`"+` in dp (typically 24dp for icons).
   - At least one `+"`<path android:pathData>`"+` and each visible path has a `+"`fillColor`"+`.
   - Report any `+"`warnings`"+` from the response instead of silently accepting them.
   - Gradients need minSdk 24+; `+"`<clip-path>`"+` is supported but masks/blur are dropped
     silently — compare against `+"`get_screenshot(format='PNG')`"+` if the result looks off.

6. **Theming**
   - Single-color icons: export the shape, then replace the literal
     `+"`android:fillColor=\"#FF000000\"`"+` with a theme attribute (`+"`?attr/colorControlNormal`"+`)
     or a color resource, so the icon follows light/dark theme.
   - Multi-color brand marks: keep the literal colors; do not tint them.

## Rules
- A file path is the only accepted input; there is no inline/base64 parameter. Do not paste SVG
  markup into the call — save the file and pass its path.
- One item per asset; never concatenate several SVGs into one file.
- Do not paste generated XML into your reply — write it with `+"`outputPath`"+` and report the paths.
- Never overwrite an existing drawable without saying so; check the path first.
- If the converter errors, report the message as-is (a missing Node.js runtime and a malformed
  SVG are different problems) rather than retrying with random option changes.
`),
				),
			},
		), nil
	})
}
