package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

func TestComputeSVGVectorCacheKey_MatchesSourceWrapper(t *testing.T) {
	svg := `<svg viewBox="0 0 24 24"><path d="M0 0 L10 0 Z"/></svg>`
	params := svgVectorToolParams{FloatPrecision: 2, FillBlack: true, XMLTag: true, Tint: "#112233", Cache: true}
	got := computeSVGVectorCacheKey(svg, params)

	options := map[string]interface{}{
		"floatPrecision": 2,
		"fillBlack":      true,
		"xmlTag":         true,
		"tint":           "#112233",
	}
	encoded, err := json.Marshal(options)
	if err != nil {
		t.Fatalf("marshal options: %v", err)
	}
	sum := sha256.Sum256(append([]byte(svg), encoded...))
	want := hex.EncodeToString(sum[:])
	if got != want {
		t.Fatalf("cache key mismatch: got %s want %s", got, want)
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

func TestExecuteConvertSVGToAndroidDrawable_InlineAndCache(t *testing.T) {
	globalSVGVectorCache = newSVGVectorCache()
	args := map[string]any{
		"svg":            `<svg viewBox="0 0 24 24"><path d="M2 2 L22 2 L22 22 Z" fill="#123456"/></svg>`,
		"floatPrecision": float64(2),
	}

	first, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}})
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

	second, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}})
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
}

func TestExecuteConvertSVGToAndroidDrawable_XmlTagAndTint(t *testing.T) {
	result, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"svg":     `<svg viewBox="0 0 24 24"><path d="M0 0 H24 V24 Z"/></svg>`,
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

func TestExecuteConvertSVGToAndroidDrawable_AllowsBothSvgAndSvgPathLikeSource(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "icon.svg")
	if err := os.WriteFile(inputPath, []byte(`<svg viewBox="0 0 24 24"><path d="M0 0 H24 V24 Z" fill="#000"/></svg>`), 0o644); err != nil {
		t.Fatalf("write input svg: %v", err)
	}
	result, err := executeConvertSVGToAndroidDrawable(nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"svg":     `<svg viewBox="0 0 24 24"><path d="M0 0 H10 V10 Z" fill="#f00"/></svg>`,
		"svgPath": inputPath,
	}}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	payload := decodeSVGVectorResult(t, result)
	if !strings.Contains(payload.VectorDrawable, `android:fillColor="#F00"`) && !strings.Contains(payload.VectorDrawable, `android:fillColor="#FF0000"`) {
		t.Fatalf("expected inline svg to win over svgPath, got %s", payload.VectorDrawable)
	}
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
