package internal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const svgVectorCacheMaxEntries = 32
const svgVectorConverterVersion = "android-mcp-toolkit-js-v1"
const svgVectorRuntimeDirName = "svg2vectordrawable_runtime"
const svgVectorToolName = "convert_svg_to_android_drawable"

// Each conversion spawns a Node process, so a batch of icons is dominated by
// process startup rather than CPU. A handful of workers hides most of that
// latency without swamping the machine on a 100-icon export.
const svgVectorBatchMaxWorkers = 8

// svgVectorDefaultFloatPrecision is the maximum the converter supports. An icon is
// exported once, and the extra pathData bytes are cheaper than a drawable whose
// curves are visibly off on a small viewBox, so max quality is the default.
const svgVectorDefaultFloatPrecision = 6

type svgVectorToolParams struct {
	SVGPath        string
	OutputPath     string
	FloatPrecision int
	FillBlack      bool
	XMLTag         bool
	Tint           string
	Cache          bool
}

type svgVectorResult struct {
	Success        bool     `json:"success"`
	Source         string   `json:"source"`
	SourcePath     string   `json:"sourcePath,omitempty"`
	Error          string   `json:"error,omitempty"`
	SVGDeleted     bool     `json:"svgDeleted"`
	CacheHit       bool     `json:"cacheHit"`
	CacheKey       string   `json:"cacheKey,omitempty"`
	OutputPath     string   `json:"outputPath,omitempty"`
	VectorDrawable string   `json:"vectorDrawable,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
	Metrics        struct {
		InputBytes  int   `json:"inputBytes"`
		OutputBytes int   `json:"outputBytes"`
		ConvertMs   int64 `json:"convertMs"`
	} `json:"metrics"`
}

// svgVectorBatchResult is what a batch call returns. Results are in the same
// order as the requested items, so the caller can pair them up by position.
type svgVectorBatchResult struct {
	Success   bool              `json:"success"`
	Batch     bool              `json:"batch"`
	Total     int               `json:"total"`
	Converted int               `json:"converted"`
	Failed    int               `json:"failed"`
	Results   []svgVectorResult `json:"results"`
}

// svgVectorRequest is one parsed tool call: either a single conversion or a
// batch. Both run through the same per-item path.
type svgVectorRequest struct {
	Items []svgVectorToolParams
	Batch bool
}

type svgVectorCache struct {
	mu      sync.Mutex
	entries map[string]string
	order   []string
}

type svgVectorRunnerPayload struct {
	SVG     string                 `json:"svg"`
	Options map[string]interface{} `json:"options"`
}

type svgVectorRunnerResult struct {
	XML   string `json:"xml"`
	Error string `json:"error"`
}

func newSVGVectorCache() *svgVectorCache {
	return &svgVectorCache{entries: make(map[string]string)}
}

func (c *svgVectorCache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, ok := c.entries[key]
	if !ok {
		return "", false
	}
	c.touchLocked(key)
	return value, true
}

func (c *svgVectorCache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.entries[key]; ok {
		c.entries[key] = value
		c.touchLocked(key)
		return
	}
	if len(c.order) >= svgVectorCacheMaxEntries {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}
	c.entries[key] = value
	c.order = append(c.order, key)
}

func (c *svgVectorCache) touchLocked(key string) {
	for i, existing := range c.order {
		if existing == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
	c.order = append(c.order, key)
}

var globalSVGVectorCache = newSVGVectorCache()

func registerSVGTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool(svgVectorToolName,
		mcp.WithDescription("Convert SVG files on disk into Android VectorDrawable XML, one asset or a whole batch per call. Input is a file path only — inline SVG markup and base64 are not accepted, which keeps SVG payloads out of the conversation. Typical flow: save_screenshots({format:'SVG'}) on the icon container nodes (COMPONENT/FRAME, not their VECTOR children), then call this tool once with items:[{svgPath, outputPath}, ...] so every drawable is written straight into res/drawable/ic_<snake_case>.xml. Batch conversions run in parallel and report per-item results in request order; one failing icon does not abort the rest. Each source SVG file is deleted once its conversion succeeds — it is a throwaway intermediate, so export to temp paths, not over SVGs you want to keep, and do not convert the same svgPath twice. Not for raster art, layer effects, or blend modes; export those as PNG/WebP into res/drawable-*dpi instead. See the svg_to_drawable_strategy prompt for the full workflow."),
		mcp.WithArray("items",
			mcp.Description("Batch mode: list of {svgPath, outputPath?, floatPrecision?, fillBlack?, xmlTag?, tint?} objects, one per icon. Preferred over svgPath whenever you have more than one asset — a single call converts them in parallel. Per-item fields override the top-level values, which act as defaults for the batch. Mutually exclusive with svgPath. To keep the response small, an item that has outputPath returns the written path instead of the XML itself."),
			mcp.Items(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"svgPath":        map[string]any{"type": "string", "description": "Path to an SVG file on disk to convert"},
					"outputPath":     map[string]any{"type": "string", "description": "File path to write this drawable to, e.g. 'app/src/main/res/drawable/ic_star.xml'"},
					"floatPrecision": map[string]any{"type": "number", "description": "Decimal precision for path coordinates, integer 1-6"},
					"fillBlack":      map[string]any{"type": "boolean", "description": "Force black fill on paths with no fill attribute"},
					"xmlTag":         map[string]any{"type": "boolean", "description": "Include the XML declaration"},
					"tint":           map[string]any{"type": "string", "description": "Sets android:tint on the root vector, e.g. '#FF000000'"},
				},
				"required": []string{"svgPath"},
			}),
		),
		mcp.WithString("svgPath",
			mcp.Description("Single-asset mode: path to an SVG file on disk to convert. A file path is the only accepted input — inline markup and base64 are not supported. Write the SVG to a file first (save_screenshots with format='SVG' does this directly from Figma) and pass that path here. Use items instead for more than one asset; svgPath and items cannot be combined."),
		),
		mcp.WithString("outputPath",
			mcp.Description("File path to write the generated VectorDrawable XML to, e.g. 'app/src/main/res/drawable/ic_star.xml'. Set this whenever the target is an Android project. Resource file names must be lowercase a-z, 0-9 or _ and start with a letter — sanitize the Figma layer name first. Single-asset mode only; in batch mode set outputPath per item."),
		),
		mcp.WithNumber("floatPrecision",
			mcp.Description("Decimal precision for path coordinates, integer 1-6. Default 6 (maximum quality), which is correct for any viewBox size. Lower it only to shrink a large illustration whose pathData is too long — at 24dp there is nothing to gain. In batch mode this is the default for every item."),
		),
		mcp.WithBoolean("fillBlack",
			mcp.Description("Force black fill on paths that have no fill attribute. Default false. Set true when a converted drawable renders as blank — unfilled paths are otherwise invisible on Android. Leave false when the artwork is stroke-only, or the strokes get a spurious black fill. In batch mode this is the default for every item."),
		),
		mcp.WithBoolean("xmlTag",
			mcp.Description("Include the XML declaration. Default false, which is what res/ XML files normally use. In batch mode this is the default for every item."),
		),
		mcp.WithString("tint",
			mcp.Description("Sets android:tint on the root vector, e.g. '#FF000000'. Use for single-color icons with a fixed color; leave unset when the icon is tinted at the call site or is multi-color. In batch mode this is the default for every item."),
		),
		mcp.WithBoolean("cache",
			mcp.Description("Reuse the cached result for identical SVG + options within this process. Default true."),
		),
	), executeConvertSVGToAndroidDrawable)
}

func executeConvertSVGToAndroidDrawable(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	request, err := parseSVGVectorToolRequest(req.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if !request.Batch {
		result, err := runSVGVectorConversion(request.Items[0])
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return marshalSVGVectorPayload(result)
	}

	batch := svgVectorBatchResult{
		Batch:   true,
		Total:   len(request.Items),
		Results: runSVGVectorBatch(request.Items),
	}
	for _, result := range batch.Results {
		if result.Success {
			batch.Converted++
		} else {
			batch.Failed++
		}
	}
	// A batch is a success only when every icon converted: a caller that writes
	// 39 of 40 drawables and reports "done" ships a missing resource.
	batch.Success = batch.Failed == 0
	return marshalSVGVectorPayload(batch)
}

func marshalSVGVectorPayload(payload interface{}) (*mcp.CallToolResult, error) {
	out, err := json.Marshal(payload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// runSVGVectorBatch converts every item, keeping results in request order. A
// failing item is recorded and the rest of the batch still runs — re-exporting
// and re-converting 40 icons because one had a bad export is not acceptable.
func runSVGVectorBatch(items []svgVectorToolParams) []svgVectorResult {
	results := make([]svgVectorResult, len(items))

	workers := len(items)
	if workers > svgVectorBatchMaxWorkers {
		workers = svgVectorBatchMaxWorkers
	}
	slots := make(chan struct{}, workers)

	var wg sync.WaitGroup
	for i, params := range items {
		wg.Add(1)
		go func(i int, params svgVectorToolParams) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()

			result, err := runSVGVectorConversion(params)
			if err != nil {
				result.Success = false
				result.Source = "file"
				result.SourcePath = absOrCleanPath(params.SVGPath)
				result.Error = err.Error()
			} else if result.OutputPath != "" {
				// The XML is on disk; echoing it back for every icon would blow up
				// the response for no gain.
				result.VectorDrawable = ""
			}
			results[i] = result
		}(i, params)
	}
	wg.Wait()

	return results
}

func runSVGVectorConversion(params svgVectorToolParams) (svgVectorResult, error) {
	svgCode, sourcePath, err := loadSVGSource(params)
	if err != nil {
		return svgVectorResult{}, err
	}

	cacheKey := computeSVGVectorCacheKey(svgCode, params)
	started := time.Now()
	vectorXML := ""
	cacheHit := false

	if params.Cache {
		if cached, ok := globalSVGVectorCache.Get(cacheKey); ok {
			vectorXML = cached
			cacheHit = true
		}
	}

	if !cacheHit {
		converted, err := convertSVGToVectorDrawable(svgCode, params)
		if err != nil {
			return svgVectorResult{}, err
		}
		if converted == "" {
			return svgVectorResult{}, fmt.Errorf("Conversion did not produce XML")
		}
		vectorXML = converted
		if params.Cache {
			globalSVGVectorCache.Set(cacheKey, vectorXML)
		}
	}

	result := svgVectorResult{
		Success:        true,
		Source:         "file",
		SourcePath:     sourcePath,
		CacheHit:       cacheHit,
		CacheKey:       cacheKey,
		VectorDrawable: vectorXML,
	}
	result.Metrics.InputBytes = len(svgCode)
	result.Metrics.OutputBytes = len(vectorXML)
	result.Metrics.ConvertMs = time.Since(started).Milliseconds()

	if params.OutputPath != "" {
		resolvedPath, err := filepath.Abs(params.OutputPath)
		if err != nil {
			return svgVectorResult{}, fmt.Errorf("resolve outputPath: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
			return svgVectorResult{}, fmt.Errorf("mkdir: %w", err)
		}
		if err := os.WriteFile(resolvedPath, []byte(vectorXML), 0o644); err != nil {
			return svgVectorResult{}, fmt.Errorf("write file: %w", err)
		}
		result.OutputPath = resolvedPath
	}

	// The exported SVG is an intermediate; the drawable XML is the artifact that ships.
	if err := os.Remove(sourcePath); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("could not delete source SVG %s: %v", sourcePath, err))
	} else {
		result.SVGDeleted = true
	}

	return result, nil
}

func parseSVGVectorToolRequest(args map[string]interface{}) (svgVectorRequest, error) {
	defaults := svgVectorToolParams{FloatPrecision: svgVectorDefaultFloatPrecision, Cache: true}
	applySVGVectorOverrides(&defaults, args)
	if raw, ok := args["cache"].(bool); ok {
		defaults.Cache = raw
	}

	rawItems, hasItems := args["items"]
	if !hasItems {
		if strings.TrimSpace(defaults.SVGPath) == "" {
			return svgVectorRequest{}, fmt.Errorf("svgPath or items is required: write the SVG to a file (e.g. save_screenshots with format='SVG') and pass its path")
		}
		if err := validateSVGVectorParams(defaults); err != nil {
			return svgVectorRequest{}, err
		}
		return svgVectorRequest{Items: []svgVectorToolParams{defaults}}, nil
	}

	if strings.TrimSpace(defaults.SVGPath) != "" {
		return svgVectorRequest{}, fmt.Errorf("svgPath and items cannot be combined: pass every asset as an entry in items")
	}
	if defaults.OutputPath != "" {
		return svgVectorRequest{}, fmt.Errorf("outputPath cannot be used with items: set outputPath on each item")
	}

	entries, ok := rawItems.([]interface{})
	if !ok {
		return svgVectorRequest{}, fmt.Errorf("items must be an array of {svgPath, outputPath?} objects")
	}
	if len(entries) == 0 {
		return svgVectorRequest{}, fmt.Errorf("items must not be empty")
	}

	// Every item is validated before anything runs: a batch that half-converts
	// and then rejects item 30 has already deleted 29 source SVGs.
	items := make([]svgVectorToolParams, 0, len(entries))
	seenSources := make(map[string]int, len(entries))
	seenOutputs := make(map[string]int, len(entries))
	for i, entry := range entries {
		fields, ok := entry.(map[string]interface{})
		if !ok {
			return svgVectorRequest{}, fmt.Errorf("items[%d] must be an object with an svgPath", i)
		}
		params := defaults
		applySVGVectorOverrides(&params, fields)
		if strings.TrimSpace(params.SVGPath) == "" {
			return svgVectorRequest{}, fmt.Errorf("items[%d].svgPath is required", i)
		}
		if err := validateSVGVectorParams(params); err != nil {
			return svgVectorRequest{}, fmt.Errorf("items[%d]: %w", i, err)
		}

		source := absOrCleanPath(params.SVGPath)
		if first, dup := seenSources[source]; dup {
			// The source SVG is deleted on success, so the later item would fail
			// on a missing file — a confusing way to learn the list has a duplicate.
			return svgVectorRequest{}, fmt.Errorf("items[%d].svgPath duplicates items[%d] (%s); each SVG can only be converted once", i, first, params.SVGPath)
		}
		seenSources[source] = i

		if params.OutputPath != "" {
			output := absOrCleanPath(params.OutputPath)
			if first, dup := seenOutputs[output]; dup {
				return svgVectorRequest{}, fmt.Errorf("items[%d].outputPath duplicates items[%d] (%s); one of the drawables would be overwritten", i, first, params.OutputPath)
			}
			seenOutputs[output] = i
		}

		items = append(items, params)
	}

	return svgVectorRequest{Items: items, Batch: true}, nil
}

// applySVGVectorOverrides fills in whichever conversion fields the map carries,
// leaving the rest untouched — the same code serves top-level defaults and the
// per-item overrides layered on top of them.
func applySVGVectorOverrides(params *svgVectorToolParams, fields map[string]interface{}) {
	if raw, ok := fields["svgPath"].(string); ok {
		params.SVGPath = raw
	}
	if raw, ok := fields["outputPath"].(string); ok {
		params.OutputPath = raw
	}
	if raw, ok := fields["floatPrecision"].(float64); ok {
		params.FloatPrecision = int(raw)
	}
	if raw, ok := fields["fillBlack"].(bool); ok {
		params.FillBlack = raw
	}
	if raw, ok := fields["xmlTag"].(bool); ok {
		params.XMLTag = raw
	}
	if raw, ok := fields["tint"].(string); ok {
		params.Tint = raw
	}
}

func validateSVGVectorParams(params svgVectorToolParams) error {
	// 0 is rejected rather than accepted: the converter treats floatPrecision as
	// falsy and silently falls back to 2, so it would not do what the caller asked.
	if params.FloatPrecision < 1 || params.FloatPrecision > 6 {
		return fmt.Errorf("floatPrecision must be between 1 and 6")
	}
	return nil
}

func absOrCleanPath(path string) string {
	if resolved, err := filepath.Abs(path); err == nil {
		return resolved
	}
	return filepath.Clean(path)
}

func loadSVGSource(params svgVectorToolParams) (string, string, error) {
	resolvedPath, err := filepath.Abs(params.SVGPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve svgPath: %w", err)
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return "", "", fmt.Errorf("read svgPath: %w", err)
	}
	content := string(data)
	if !strings.Contains(content, "<svg") {
		return "", "", fmt.Errorf("svgPath does not contain SVG markup (no <svg> element in %s)", resolvedPath)
	}
	return content, resolvedPath, nil
}

// svgVectorRunnerOptions is the single source of truth for everything that affects
// the generated XML, so the cache key and the runner payload cannot drift apart.
func svgVectorRunnerOptions(params svgVectorToolParams) map[string]interface{} {
	return map[string]interface{}{
		"floatPrecision": params.FloatPrecision,
		"fillBlack":      params.FillBlack,
		"xmlTag":         params.XMLTag,
		"tint":           params.Tint,
	}
}

func computeSVGVectorCacheKey(svgCode string, params svgVectorToolParams) string {
	encoded, _ := json.Marshal(svgVectorRunnerOptions(params))
	sum := sha256.Sum256([]byte(svgCode + string(encoded)))
	return hex.EncodeToString(sum[:])
}

func convertSVGToVectorDrawable(svgCode string, params svgVectorToolParams) (string, error) {
	runtimeDir, err := svgVectorRuntimeDir()
	if err != nil {
		return "", err
	}
	payload := svgVectorRunnerPayload{
		SVG:     svgCode,
		Options: svgVectorRunnerOptions(params),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal runner payload: %w", err)
	}

	nodeBin, err := nodeExecutableName()
	if err != nil {
		return "", err
	}
	cmd := exec.Command(nodeBin, "runner.js")
	cmd.Dir = runtimeDir
	cmd.Stdin = bytes.NewReader(encoded)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("svg runtime: %s", message)
	}

	var result svgVectorRunnerResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return "", fmt.Errorf("decode runner output: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("%s", result.Error)
	}
	if result.XML == "" {
		return "", fmt.Errorf("Conversion did not produce XML")
	}
	return result.XML, nil
}

func svgVectorRuntimeDir() (string, error) {
	candidates := make([]string, 0, 5)

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, svgVectorRuntimeDirName),
			filepath.Join(exeDir, "internal", svgVectorRuntimeDirName),
			filepath.Join(filepath.Dir(exeDir), svgVectorRuntimeDirName),
		)
	}

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, svgVectorRuntimeDirName),
			filepath.Join(wd, "internal", svgVectorRuntimeDirName),
		)
	}

	if _, currentFile, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(currentFile), svgVectorRuntimeDirName))
	}

	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "runner.js")); err == nil {
			return dir, nil
		}
	}

	return "", fmt.Errorf("svg runtime missing (checked directories: %v)", candidates)
}

func nodeExecutableName() (string, error) {
	names := []string{"node"}
	if runtime.GOOS == "windows" {
		names = []string{"node.exe", "node"}
	}
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("Node.js runtime not found on system PATH. Please install Node.js to use convert_svg_to_android_drawable")
}
