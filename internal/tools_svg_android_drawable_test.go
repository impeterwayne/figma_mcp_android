package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestSVGVectorCache_LRUTouchAndEvict(t *testing.T) {
	cache := newSVGVectorCache()
	cache.Set("a", "1")
	cache.Set("b", "2")
	cache.Set("c", "3")
	if value, ok := cache.Get("a"); !ok || value != "1" {
		t.Fatalf("expected cache hit for a")
	}
	for i := 0; i < svgVectorCacheMaxEntries; i++ {
		cache.Set(strings.Repeat("x", i+1), "v")
	}
	if _, ok := cache.Get("b"); ok {
		t.Fatal("expected least-recently-used entry b to be evicted")
	}
}

func TestComputeSVGVectorCacheKey_CoversRunnerOptions(t *testing.T) {
	svg := `<svg viewBox="0 0 24 24"><path d="M0 0 L10 0 Z"/></svg>`
	params := svgVectorToolParams{FloatPrecision: 2, FillBlack: true, XMLTag: true, Tint: "#112233", Cache: true}
	got := computeSVGVectorCacheKey(svg, params)

	encoded, err := json.Marshal(map[string]interface{}{
		"floatPrecision": 2,
		"fillBlack":      true,
		"xmlTag":         true,
		"tint":           "#112233",
	})
	if err != nil {
		t.Fatalf("marshal options: %v", err)
	}
	sum := sha256.Sum256(append([]byte(svg), encoded...))
	want := hex.EncodeToString(sum[:])
	if got != want {
		t.Fatalf("cache key mismatch: got %s want %s", got, want)
	}

	// Every option that changes the XML must change the key, or a later call with
	// different options gets served a stale conversion.
	for name, other := range map[string]svgVectorToolParams{
		"floatPrecision": {FloatPrecision: 3, FillBlack: true, XMLTag: true, Tint: "#112233", Cache: true},
		"fillBlack":      {FloatPrecision: 2, XMLTag: true, Tint: "#112233", Cache: true},
		"xmlTag":         {FloatPrecision: 2, FillBlack: true, Tint: "#112233", Cache: true},
		"tint":           {FloatPrecision: 2, FillBlack: true, XMLTag: true, Cache: true},
	} {
		if computeSVGVectorCacheKey(svg, other) == got {
			t.Fatalf("expected %s to change the cache key", name)
		}
	}
}

func TestConvertSVGToVectorDrawable_UsesSourceJSBehavior(t *testing.T) {
	xml, err := convertSVGToVectorDrawable(`<svg viewBox="0 0 24 24"><path d="M2 2 L22 2 L22 22 Z" fill="rgba(255,0,0,0.5)"/></svg>`, svgVectorToolParams{FloatPrecision: 2})
	if err != nil {
		t.Fatalf("convertSVGToVectorDrawable: %v", err)
	}
	if !strings.Contains(xml, `<vector xmlns:android="http://schemas.android.com/apk/res/android"`) {
		t.Fatalf("expected vector root, got %s", xml)
	}
	if !strings.Contains(xml, `android:fillColor="#000080"`) {
		t.Fatalf("expected source rgba conversion behavior, got %s", xml)
	}
	if !strings.Contains(xml, `android:pathData="M2 2h20v20Z"`) {
		t.Fatalf("expected source path rounding/serialization, got %s", xml)
	}
}

func TestConvertSVGToVectorDrawable_SupportsGradient(t *testing.T) {
	xml, err := convertSVGToVectorDrawable(`<svg viewBox="0 0 24 24"><defs><linearGradient id="g"><stop offset="0%" stop-color="#ff0000"/><stop offset="100%" stop-color="#00ff00"/></linearGradient></defs><path d="M0 0H24V24H0Z" fill="url(#g)"/></svg>`, svgVectorToolParams{FloatPrecision: 2})
	if err != nil {
		t.Fatalf("convertSVGToVectorDrawable: %v", err)
	}
	if !strings.Contains(xml, `<aapt:attr name="android:fillColor">`) {
		t.Fatalf("expected gradient aapt attr, got %s", xml)
	}
	if !strings.Contains(xml, `<gradient`) {
		t.Fatalf("expected gradient element, got %s", xml)
	}
}

func TestExecuteConvertSVGToAndroidDrawable_Cache(t *testing.T) {
	globalSVGVectorCache = newSVGVectorCache()
	markup := `<svg viewBox="0 0 24 24"><path d="M2 2 L22 2 L22 22 Z" fill="#123456"/></svg>`
	// Two files rather than one path twice: the source SVG is deleted on success, so
	// the cache is only ever exercised by a second file with identical content.
	firstArgs := map[string]any{"svgPath": writeTestSVG(t, markup)}
	secondArgs := map[string]any{"svgPath": writeTestSVG(t, markup)}

	first, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: firstArgs}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if first.IsError {
		t.Fatalf("unexpected tool error: %+v", first.Content)
	}
	firstPayload := decodeSVGVectorResult(t, first)
	if firstPayload.CacheHit {
		t.Fatal("expected first call to miss cache")
	}

	second, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: secondArgs}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if second.IsError {
		t.Fatalf("unexpected tool error on second call: %+v", second.Content)
	}
	secondPayload := decodeSVGVectorResult(t, second)
	if !secondPayload.CacheHit {
		t.Fatal("expected second call to hit cache")
	}
}

func TestExecuteConvertSVGToAndroidDrawable_CacheDisabled(t *testing.T) {
	globalSVGVectorCache = newSVGVectorCache()
	markup := `<svg viewBox="0 0 24 24"><path d="M2 2 L22 2 L22 22 Z" fill="#654321"/></svg>`

	for i := 0; i < 2; i++ {
		result, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
			"svgPath": writeTestSVG(t, markup),
			"cache":   false,
		}}})
		if err != nil {
			t.Fatalf("unexpected Go error: %v", err)
		}
		if result.IsError {
			t.Fatalf("unexpected tool error: %+v", result.Content)
		}
		if decodeSVGVectorResult(t, result).CacheHit {
			t.Fatalf("call %d: expected no cache hit with cache=false", i)
		}
	}
}

func TestExecuteConvertSVGToAndroidDrawable_FromFileAndWriteOutput(t *testing.T) {
	globalSVGVectorCache = newSVGVectorCache()
	inputDir := t.TempDir()
	outputDir := t.TempDir()
	inputPath := filepath.Join(inputDir, "icon.svg")
	outputPath := filepath.Join(outputDir, "icon.xml")
	if err := os.WriteFile(inputPath, []byte(`<svg viewBox="0 0 24 24"><path d="M0 0 H24 V24 Z" fill="rgb(1, 2, 3)"/></svg>`), 0o644); err != nil {
		t.Fatalf("write input svg: %v", err)
	}

	result, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"svgPath":    inputPath,
		"outputPath": outputPath,
	}}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	payload := decodeSVGVectorResult(t, result)
	if payload.Source != "file" {
		t.Fatalf("expected source=file, got %s", payload.Source)
	}
	if payload.OutputPath == "" {
		t.Fatal("expected output path in payload")
	}
	written, err := os.ReadFile(payload.OutputPath)
	if err != nil {
		t.Fatalf("read written output: %v", err)
	}
	if !strings.Contains(string(written), `android:fillColor="#010203"`) {
		t.Fatalf("expected converted rgb color in output, got %s", string(written))
	}
	if !payload.SVGDeleted {
		t.Fatal("expected source svg to be deleted")
	}
	if payload.SourcePath == "" {
		t.Fatal("expected sourcePath in payload")
	}
	if _, err := os.Stat(inputPath); !os.IsNotExist(err) {
		t.Fatalf("expected source svg removed from disk, stat err = %v", err)
	}
}

func TestExecuteConvertSVGToAndroidDrawable_DefaultsToMaxPrecision(t *testing.T) {
	globalSVGVectorCache = newSVGVectorCache()
	markup := `<svg viewBox="0 0 1 1"><path d="M0.123456 0.234567 L0.876543 0.234567 L0.876543 0.876543 Z" fill="#123456"/></svg>`

	result, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"svgPath": writeTestSVG(t, markup),
	}}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	payload := decodeSVGVectorResult(t, result)
	// Default is max precision, so a tiny viewBox keeps its fractional detail instead
	// of collapsing to two decimals.
	if !strings.Contains(payload.VectorDrawable, "0.123456") {
		t.Fatalf("expected max-precision coordinates by default, got %s", payload.VectorDrawable)
	}
	if strings.HasPrefix(payload.VectorDrawable, "<?xml") {
		t.Fatalf("expected no xml declaration by default, got %s", payload.VectorDrawable)
	}

	// ...and it is still overridable.
	lowered, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"svgPath":        writeTestSVG(t, markup),
		"floatPrecision": float64(2),
	}}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if lowered.IsError {
		t.Fatalf("unexpected tool error: %+v", lowered.Content)
	}
	if strings.Contains(decodeSVGVectorResult(t, lowered).VectorDrawable, "0.123456") {
		t.Fatal("expected floatPrecision=2 to drop coordinate detail")
	}
}

func TestExecuteConvertSVGToAndroidDrawable_XmlTagAndTint(t *testing.T) {
	result, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"svgPath": writeTestSVG(t, `<svg viewBox="0 0 24 24"><path d="M0 0 H24 V24 Z"/></svg>`),
		"xmlTag":  true,
		"tint":    "#aa112233",
		"cache":   false,
	}}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	payload := decodeSVGVectorResult(t, result)
	if !strings.HasPrefix(payload.VectorDrawable, "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n") {
		t.Fatalf("expected xml declaration, got %s", payload.VectorDrawable)
	}
	if !strings.Contains(payload.VectorDrawable, `android:tint="#AA112233"`) {
		t.Fatalf("expected uppercased tint in output, got %s", payload.VectorDrawable)
	}
}

func TestExecuteConvertSVGToAndroidDrawable_RejectsFloatPrecisionZero(t *testing.T) {
	// 0 is in range for the JS converter but silently means 2, so the tool rejects it.
	result, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"svgPath":        writeTestSVG(t, `<svg viewBox="0 0 24 24"><path d="M0 0 H24 V24 Z"/></svg>`),
		"floatPrecision": float64(0),
	}}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected floatPrecision=0 to be rejected, got %+v", result.Content)
	}
}

func TestExecuteConvertSVGToAndroidDrawable_RejectsNonFileInput(t *testing.T) {
	base64SVG := "PHN2ZyB2aWV3Qm94PSIwIDAgMjQgMjQiPjxwYXRoIGQ9Ik0yIDIgTDIyIDIgTDIyIDIyIFoiIGZpbGw9IiMxMjM0NTYiLz48L3N2Zz4="

	cases := map[string]map[string]any{
		"inline markup": {"svg": `<svg viewBox="0 0 24 24"><path d="M0 0 H24 V24 Z"/></svg>`},
		"raw base64":    {"svg": base64SVG},
		"data uri":      {"svg": "data:image/svg+xml;base64," + base64SVG},
		"no input":      {},
	}

	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}})
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			if !result.IsError {
				t.Fatalf("expected svgPath-required error, got success: %+v", result.Content)
			}
		})
	}
}

func TestExecuteConvertSVGToAndroidDrawable_RejectsNonSVGFile(t *testing.T) {
	notSVG := filepath.Join(t.TempDir(), "icon.svg")
	if err := os.WriteFile(notSVG, []byte("just some text"), 0o644); err != nil {
		t.Fatalf("write input file: %v", err)
	}
	result, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"svgPath": notSVG,
	}}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected error for file without SVG markup, got success: %+v", result.Content)
	}
}

func TestExecuteConvertSVGToAndroidDrawable_BatchWritesEveryItem(t *testing.T) {
	globalSVGVectorCache = newSVGVectorCache()
	outputDir := t.TempDir()
	sources := []string{
		writeTestSVG(t, `<svg viewBox="0 0 24 24"><path d="M0 0 H24 V24 Z" fill="rgb(1, 2, 3)"/></svg>`),
		writeTestSVG(t, `<svg viewBox="0 0 24 24"><path d="M2 2 H22 V22 Z" fill="rgb(4, 5, 6)"/></svg>`),
		writeTestSVG(t, `<svg viewBox="0 0 24 24"><path d="M4 4 H20 V20 Z" fill="rgb(7, 8, 9)"/></svg>`),
	}
	items := make([]any, 0, len(sources))
	for i, source := range sources {
		items = append(items, map[string]any{
			"svgPath":    source,
			"outputPath": filepath.Join(outputDir, fmt.Sprintf("ic_%d.xml", i)),
		})
	}

	result, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"items": items,
	}}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}

	payload := decodeSVGVectorBatch(t, result)
	if !payload.Success || !payload.Batch {
		t.Fatalf("expected successful batch payload, got %+v", payload)
	}
	if payload.Total != 3 || payload.Converted != 3 || payload.Failed != 0 {
		t.Fatalf("expected 3/3 converted, got total=%d converted=%d failed=%d", payload.Total, payload.Converted, payload.Failed)
	}

	wantColors := []string{"#010203", "#040506", "#070809"}
	for i, item := range payload.Results {
		// Results keep request order, so the caller can pair them with its items.
		if item.SourcePath != sources[i] {
			t.Fatalf("result %d: expected sourcePath %s, got %s", i, sources[i], item.SourcePath)
		}
		if !item.Success || item.Error != "" {
			t.Fatalf("result %d: expected success, got %+v", i, item)
		}
		// The XML is on disk; repeating it in the response is what makes a
		// 40-icon batch expensive.
		if item.VectorDrawable != "" {
			t.Fatalf("result %d: expected XML to be omitted when written to disk, got %s", i, item.VectorDrawable)
		}
		written, err := os.ReadFile(item.OutputPath)
		if err != nil {
			t.Fatalf("result %d: read written output: %v", i, err)
		}
		if !strings.Contains(string(written), `android:fillColor="`+wantColors[i]+`"`) {
			t.Fatalf("result %d: expected %s in output, got %s", i, wantColors[i], string(written))
		}
		if !item.SVGDeleted {
			t.Fatalf("result %d: expected source svg to be deleted", i)
		}
		if _, err := os.Stat(sources[i]); !os.IsNotExist(err) {
			t.Fatalf("result %d: expected source svg removed from disk, stat err = %v", i, err)
		}
	}
}

func TestExecuteConvertSVGToAndroidDrawable_BatchKeepsXMLWithoutOutputPath(t *testing.T) {
	globalSVGVectorCache = newSVGVectorCache()
	result, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"items": []any{
			map[string]any{"svgPath": writeTestSVG(t, `<svg viewBox="0 0 24 24"><path d="M0 0 H24 V24 Z" fill="#123456"/></svg>`)},
		},
	}}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	// With nowhere to write it, the XML is the only way the caller gets the drawable.
	if xml := decodeSVGVectorBatch(t, result).Results[0].VectorDrawable; !strings.Contains(xml, "<vector") {
		t.Fatalf("expected inline vector XML when no outputPath, got %q", xml)
	}
}

func TestExecuteConvertSVGToAndroidDrawable_BatchSurvivesFailingItem(t *testing.T) {
	globalSVGVectorCache = newSVGVectorCache()
	outputDir := t.TempDir()
	good := writeTestSVG(t, `<svg viewBox="0 0 24 24"><path d="M0 0 H24 V24 Z" fill="#123456"/></svg>`)
	broken := writeTestSVG(t, "not an svg at all")
	missing := filepath.Join(t.TempDir(), "gone.svg")

	result, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"items": []any{
			map[string]any{"svgPath": broken, "outputPath": filepath.Join(outputDir, "ic_broken.xml")},
			map[string]any{"svgPath": good, "outputPath": filepath.Join(outputDir, "ic_good.xml")},
			map[string]any{"svgPath": missing, "outputPath": filepath.Join(outputDir, "ic_missing.xml")},
		},
	}}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	// A partial batch is not a tool-level error: the caller needs the per-item detail.
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}

	payload := decodeSVGVectorBatch(t, result)
	if payload.Success {
		t.Fatal("expected batch success=false when an item fails")
	}
	if payload.Converted != 1 || payload.Failed != 2 {
		t.Fatalf("expected 1 converted / 2 failed, got converted=%d failed=%d", payload.Converted, payload.Failed)
	}
	for _, i := range []int{0, 2} {
		if payload.Results[i].Success || payload.Results[i].Error == "" {
			t.Fatalf("result %d: expected failure with an error message, got %+v", i, payload.Results[i])
		}
		if payload.Results[i].SourcePath == "" {
			t.Fatalf("result %d: failed item must name its svgPath", i)
		}
	}
	// The good icon still landed, which is the whole point of not aborting.
	if !payload.Results[1].Success {
		t.Fatalf("expected the valid item to convert, got %+v", payload.Results[1])
	}
	if _, err := os.Stat(payload.Results[1].OutputPath); err != nil {
		t.Fatalf("expected the valid item to be written: %v", err)
	}
	if _, err := os.Stat(broken); err != nil {
		t.Fatalf("expected the failing source svg to be left on disk for a retry: %v", err)
	}
}

func TestExecuteConvertSVGToAndroidDrawable_BatchItemOverridesDefaults(t *testing.T) {
	globalSVGVectorCache = newSVGVectorCache()
	markup := `<svg viewBox="0 0 1 1"><path d="M0.123456 0.234567 L0.876543 0.234567 L0.876543 0.876543 Z" fill="#123456"/></svg>`

	result, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"floatPrecision": float64(2),
		"tint":           "#aa112233",
		"items": []any{
			map[string]any{"svgPath": writeTestSVG(t, markup)},
			map[string]any{"svgPath": writeTestSVG(t, markup), "floatPrecision": float64(6), "tint": "#ff445566"},
		},
	}}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}

	payload := decodeSVGVectorBatch(t, result)
	inherited := payload.Results[0].VectorDrawable
	overridden := payload.Results[1].VectorDrawable
	if strings.Contains(inherited, "0.123456") {
		t.Fatalf("expected item 0 to inherit floatPrecision=2, got %s", inherited)
	}
	if !strings.Contains(inherited, `android:tint="#AA112233"`) {
		t.Fatalf("expected item 0 to inherit the batch tint, got %s", inherited)
	}
	if !strings.Contains(overridden, "0.123456") {
		t.Fatalf("expected item 1 to override floatPrecision, got %s", overridden)
	}
	if !strings.Contains(overridden, `android:tint="#FF445566"`) {
		t.Fatalf("expected item 1 to override the tint, got %s", overridden)
	}
}

func TestExecuteConvertSVGToAndroidDrawable_BatchRejectsBadRequests(t *testing.T) {
	markup := `<svg viewBox="0 0 24 24"><path d="M0 0 H24 V24 Z" fill="#123456"/></svg>`
	sharedOutput := filepath.Join(t.TempDir(), "ic_dup.xml")

	cases := map[string]map[string]any{
		"svgPath combined with items": {
			"svgPath": writeTestSVG(t, markup),
			"items":   []any{map[string]any{"svgPath": writeTestSVG(t, markup)}},
		},
		"top-level outputPath with items": {
			"outputPath": filepath.Join(t.TempDir(), "ic.xml"),
			"items":      []any{map[string]any{"svgPath": writeTestSVG(t, markup)}},
		},
		"empty items":       {"items": []any{}},
		"items not objects": {"items": []any{"icon.svg"}},
		"item without svgPath": {
			"items": []any{map[string]any{"outputPath": filepath.Join(t.TempDir(), "ic.xml")}},
		},
		"item with bad precision": {
			"items": []any{map[string]any{"svgPath": writeTestSVG(t, markup), "floatPrecision": float64(0)}},
		},
	}

	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}})
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			if !result.IsError {
				t.Fatalf("expected request to be rejected, got success: %+v", result.Content)
			}
		})
	}

	// Duplicates get their own case: nothing may be written or deleted, because the
	// rejection has to happen before the batch starts converting.
	duplicateSource := writeTestSVG(t, markup)
	survivor := writeTestSVG(t, markup)
	for name, args := range map[string]map[string]any{
		"duplicate svgPath": {"items": []any{
			map[string]any{"svgPath": duplicateSource, "outputPath": filepath.Join(t.TempDir(), "a.xml")},
			map[string]any{"svgPath": duplicateSource, "outputPath": filepath.Join(t.TempDir(), "b.xml")},
		}},
		"duplicate outputPath": {"items": []any{
			map[string]any{"svgPath": survivor, "outputPath": sharedOutput},
			map[string]any{"svgPath": writeTestSVG(t, markup), "outputPath": sharedOutput},
		}},
	} {
		t.Run(name, func(t *testing.T) {
			result, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}})
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			if !result.IsError {
				t.Fatalf("expected duplicate paths to be rejected, got success: %+v", result.Content)
			}
		})
	}
	for _, path := range []string{duplicateSource, survivor, sharedOutput} {
		if _, err := os.Stat(path); path == sharedOutput && err == nil {
			t.Fatal("expected no drawable written for a rejected batch")
		} else if path != sharedOutput && err != nil {
			t.Fatalf("expected source svg %s untouched by a rejected batch: %v", path, err)
		}
	}
}

func writeTestSVG(t *testing.T, markup string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "icon.svg")
	if err := os.WriteFile(path, []byte(markup), 0o644); err != nil {
		t.Fatalf("write input svg: %v", err)
	}
	return path
}

func decodeSVGVectorResult(t *testing.T, result *mcp.CallToolResult) svgVectorResult {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("expected tool content")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}
	var payload svgVectorResult
	if err := json.Unmarshal([]byte(textContent.Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func decodeSVGVectorBatch(t *testing.T, result *mcp.CallToolResult) svgVectorBatchResult {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("expected tool content")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}
	var payload svgVectorBatchResult
	if err := json.Unmarshal([]byte(textContent.Text), &payload); err != nil {
		t.Fatalf("unmarshal batch payload: %v", err)
	}
	if len(payload.Results) != payload.Total {
		t.Fatalf("expected %d results, got %d", payload.Total, len(payload.Results))
	}
	return payload
}
