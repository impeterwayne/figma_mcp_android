package internal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
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

type svgVectorToolParams struct {
	SVG            string
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
	CacheHit       bool     `json:"cacheHit"`
	CacheKey       string   `json:"cacheKey"`
	OutputPath     string   `json:"outputPath,omitempty"`
	VectorDrawable string   `json:"vectorDrawable"`
	Warnings       []string `json:"warnings,omitempty"`
	Metrics        struct {
		InputBytes  int   `json:"inputBytes"`
		OutputBytes int   `json:"outputBytes"`
		ConvertMs   int64 `json:"convertMs"`
	} `json:"metrics"`
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
		mcp.WithDescription("Convert inline SVG markup, base64-encoded SVG, or an SVG file into Android VectorDrawable XML for Android projects (res/drawable/ic_*.xml). Workflow: 1) Export vector node from Figma via get_screenshot(format='SVG') or save_screenshots(format='SVG'); 2) Pass the SVG markup, base64 data, or svgPath to this tool to generate VectorDrawable XML."),
		mcp.WithString("svg",
			mcp.Description("Inline SVG markup, base64-encoded SVG, or data URI to convert. Provide either svg or svgPath."),
		),
		mcp.WithString("svgPath",
			mcp.Description("Path to an SVG file on disk to convert into VectorDrawable XML. Provide either svgPath or svg."),
		),
		mcp.WithString("outputPath",
			mcp.Description("Optional file path to write generated VectorDrawable XML directly (e.g. 'app/src/main/res/drawable/ic_star.xml')."),
		),
		mcp.WithNumber("floatPrecision",
			mcp.Description("Decimal precision when serializing coordinates, integer 0-6. Default 2."),
		),
		mcp.WithBoolean("fillBlack",
			mcp.Description("Force fill color black when missing. Default false."),
		),
		mcp.WithBoolean("xmlTag",
			mcp.Description("Include XML declaration. Default false."),
		),
		mcp.WithString("tint",
			mcp.Description("Android tint color, for example #FF000000."),
		),
		mcp.WithBoolean("cache",
			mcp.Description("Reuse cached result for identical inputs within this process. Default true."),
		),
	), executeConvertSVGToAndroidDrawable)
}

func executeConvertSVGToAndroidDrawable(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params, err := parseSVGVectorToolParams(req.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	svgCode, source, err := loadSVGSource(params)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
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
			return mcp.NewToolResultError(err.Error()), nil
		}
		if converted == "" {
			return mcp.NewToolResultError("Conversion did not produce XML"), nil
		}
		vectorXML = converted
		if params.Cache {
			globalSVGVectorCache.Set(cacheKey, vectorXML)
		}
	}

	result := svgVectorResult{
		Success:        true,
		Source:         source,
		CacheHit:       cacheHit,
		CacheKey:       cacheKey,
		VectorDrawable: vectorXML,
	}
	result.Metrics.InputBytes = len(svgCode)
	result.Metrics.OutputBytes = len(vectorXML)
	result.Metrics.ConvertMs = time.Since(started).Milliseconds()

	if params.OutputPath != "" {
		resolvedPath := filepath.Clean(params.OutputPath)
		if !filepath.IsAbs(resolvedPath) {
			resolvedPath, err = filepath.Abs(resolvedPath)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("resolve outputPath: %v", err)), nil
			}
		}
		if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("mkdir: %v", err)), nil
		}
		if err := os.WriteFile(resolvedPath, []byte(vectorXML), 0o644); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("write file: %v", err)), nil
		}
		result.OutputPath = resolvedPath
	}

	out, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func parseSVGVectorToolParams(args map[string]interface{}) (svgVectorToolParams, error) {
	params := svgVectorToolParams{FloatPrecision: 2, Cache: true}

	if raw, ok := args["svg"].(string); ok {
		params.SVG = raw
	}
	if raw, ok := args["svgPath"].(string); ok {
		params.SVGPath = raw
	}
	if raw, ok := args["outputPath"].(string); ok {
		params.OutputPath = raw
	}
	if raw, ok := args["floatPrecision"].(float64); ok {
		params.FloatPrecision = int(raw)
	}
	if raw, ok := args["fillBlack"].(bool); ok {
		params.FillBlack = raw
	}
	if raw, ok := args["xmlTag"].(bool); ok {
		params.XMLTag = raw
	}
	if raw, ok := args["tint"].(string); ok {
		params.Tint = raw
	}
	if raw, ok := args["cache"].(bool); ok {
		params.Cache = raw
	}

	hasSVG := strings.TrimSpace(params.SVG) != ""
	hasPath := strings.TrimSpace(params.SVGPath) != ""
	if !hasSVG && !hasPath {
		return params, fmt.Errorf("either svg or svgPath is required")
	}
	if params.FloatPrecision < 0 || params.FloatPrecision > 6 {
		return params, fmt.Errorf("floatPrecision must be between 0 and 6")
	}
	return params, nil
}

func loadSVGSource(params svgVectorToolParams) (string, string, error) {
	if strings.TrimSpace(params.SVG) != "" {
		return normalizeSVGContent(params.SVG), "inline", nil
	}
	resolvedPath, err := filepath.Abs(params.SVGPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve svgPath: %w", err)
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return "", "", fmt.Errorf("read svgPath: %w", err)
	}
	return normalizeSVGContent(string(data)), "file", nil
}

func normalizeSVGContent(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw
	}
	if strings.HasPrefix(trimmed, "data:") {
		parts := strings.SplitN(trimmed, ",", 2)
		if len(parts) == 2 {
			header := parts[0]
			dataPart := parts[1]
			if strings.Contains(header, ";base64") {
				if decoded, err := base64.StdEncoding.DecodeString(dataPart); err == nil {
					return string(decoded)
				}
				if decoded, err := base64.RawStdEncoding.DecodeString(dataPart); err == nil {
					return string(decoded)
				}
			}
			if unescaped, err := url.QueryUnescape(dataPart); err == nil {
				return unescaped
			}
			return dataPart
		}
	}
	if !strings.HasPrefix(trimmed, "<") {
		if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil && strings.Contains(string(decoded), "<svg") {
			return string(decoded)
		}
		if decoded, err := base64.RawStdEncoding.DecodeString(trimmed); err == nil && strings.Contains(string(decoded), "<svg") {
			return string(decoded)
		}
	}
	return raw
}

func computeSVGVectorCacheKey(svgCode string, params svgVectorToolParams) string {
	cacheOptions := map[string]interface{}{
		"floatPrecision": params.FloatPrecision,
		"fillBlack":      params.FillBlack,
		"xmlTag":         params.XMLTag,
		"tint":           params.Tint,
	}
	encoded, _ := json.Marshal(cacheOptions)
	sum := sha256.Sum256([]byte(svgCode + string(encoded)))
	return hex.EncodeToString(sum[:])
}

func convertSVGToVectorDrawable(svgCode string, params svgVectorToolParams) (string, error) {
	runtimeDir, err := svgVectorRuntimeDir()
	if err != nil {
		return "", err
	}
	payload := svgVectorRunnerPayload{
		SVG: svgCode,
		Options: map[string]interface{}{
			"floatPrecision": params.FloatPrecision,
			"fillBlack":      params.FillBlack,
			"xmlTag":         params.XMLTag,
			"tint":           params.Tint,
		},
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
