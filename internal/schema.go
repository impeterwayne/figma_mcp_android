package internal

import (
	"fmt"
	"regexp"
	"strings"
)

// nodeIDPattern matches Figma node IDs:
//
//	simple:   "4029:12345"
//	compound: "I2167:9091;186:1579;186:1745" (instances/variants)
var nodeIDPattern = regexp.MustCompile(`^I?\d+:\d+(;\d+:\d+)*$`)

// NormalizeNodeID converts hyphen-format node IDs (LLM output artifact) to colon format.
// "4029-12345" → "4029:12345". No-ops for already-valid or unrecognized strings.
func NormalizeNodeID(s string) string {
	if strings.Contains(s, "-") && !strings.Contains(s, ":") {
		normalized := strings.ReplaceAll(s, "-", ":")
		if nodeIDPattern.MatchString(normalized) {
			return normalized
		}
	}
	return s
}

// ValidNodeID reports whether s is a valid Figma node ID.
func ValidNodeID(s string) bool {
	return nodeIDPattern.MatchString(s)
}

// ValidateRPC validates an incoming RPC request against the tool's expected
// input shape. Returns an error string on failure, empty string if valid.
func ValidateRPC(tool string, nodeIDs []string, params map[string]interface{}) string {
	switch tool {
	case "get_node":
		if len(nodeIDs) == 0 || nodeIDs[0] == "" {
			return "nodeId is required"
		}
		if !ValidNodeID(nodeIDs[0]) {
			return fmt.Sprintf("nodeId must use colon format e.g. 4029:12345, got: %s", nodeIDs[0])
		}
		if msg := validateDetail(params); msg != "" {
			return msg
		}

	case "get_nodes_info":
		if len(nodeIDs) == 0 {
			return "nodeIds is required and must not be empty"
		}
		for _, id := range nodeIDs {
			if !ValidNodeID(id) {
				return fmt.Sprintf("invalid nodeId: %s — must use colon format e.g. 4029:12345", id)
			}
		}
		if msg := validateDetail(params); msg != "" {
			return msg
		}

	case "get_selection":
		if msg := validateDetail(params); msg != "" {
			return msg
		}

	case "get_screenshot":
		for _, id := range nodeIDs {
			if !ValidNodeID(id) {
				return fmt.Sprintf("invalid nodeId: %s — must use colon format e.g. 4029:12345", id)
			}
		}
		if format, ok := params["format"].(string); ok {
			if !validExportFormat(format) {
				return fmt.Sprintf("format must be PNG, SVG, JPG, or PDF, got: %s", format)
			}
		}

	case "save_screenshots":
		items, ok := params["items"]
		if !ok {
			return "items is required"
		}
		itemList, ok := items.([]interface{})
		if !ok || len(itemList) == 0 {
			return "items must be a non-empty array"
		}
		for i, item := range itemList {
			m, ok := item.(map[string]interface{})
			if !ok {
				return fmt.Sprintf("items[%d] must be an object", i)
			}
			nodeID, _ := m["nodeId"].(string)
			if !ValidNodeID(nodeID) {
				return fmt.Sprintf("items[%d].nodeId must use colon format e.g. 4029:12345", i)
			}
			outputPath, _ := m["outputPath"].(string)
			if outputPath == "" {
				return fmt.Sprintf("items[%d].outputPath is required", i)
			}
		}

	case "get_design_context":
		if depth, ok := params["depth"].(float64); ok {
			if depth < 0 {
				return "depth must be a non-negative number"
			}
		}
		if msg := validateDetail(params); msg != "" {
			return msg
		}

	case "search_nodes":
		query, _ := params["query"].(string)
		if query == "" {
			return "query is required"
		}
		if nodeID, ok := params["nodeId"].(string); ok && nodeID != "" {
			if !ValidNodeID(nodeID) {
				return fmt.Sprintf("nodeId must use colon format e.g. 4029:12345, got: %s", nodeID)
			}
		}
		if limit, ok := params["limit"].(float64); ok && limit <= 0 {
			return "limit must be a positive number"
		}

	case "get_reactions":
		if len(nodeIDs) == 0 || nodeIDs[0] == "" {
			return "nodeId is required"
		}
		if !ValidNodeID(nodeIDs[0]) {
			return fmt.Sprintf("nodeId must use colon format e.g. 4029:12345, got: %s", nodeIDs[0])
		}

	case "scan_text_nodes", "scan_nodes_by_types":
		nodeID, _ := params["nodeId"].(string)
		if nodeID == "" {
			return "nodeId is required"
		}
		if !ValidNodeID(nodeID) {
			return fmt.Sprintf("nodeId must use colon format e.g. 4029:12345, got: %s", nodeID)
		}
		if tool == "scan_nodes_by_types" {
			types, ok := params["types"].([]interface{})
			if !ok || len(types) == 0 {
				return "types must be a non-empty array"
			}
		}

	case "export_tokens":
		if format, ok := params["format"].(string); ok && format != "" {
			switch format {
			case "json", "css":
			default:
				return fmt.Sprintf("format must be json or css, got: %s", format)
			}
		}

	case "convert_svg_to_android_drawable":
		svgPath, _ := params["svgPath"].(string)
		items, hasItems := params["items"].([]interface{})
		if strings.TrimSpace(svgPath) == "" && !hasItems {
			return "svgPath or items is required: write the SVG to a file (e.g. save_screenshots with format='SVG') and pass its path"
		}
		if strings.TrimSpace(svgPath) != "" && hasItems {
			return "svgPath and items cannot be combined: pass every asset as an entry in items"
		}
		if hasItems && len(items) == 0 {
			return "items must not be empty"
		}
		for i, entry := range items {
			fields, ok := entry.(map[string]interface{})
			if !ok {
				return fmt.Sprintf("items[%d] must be an object with an svgPath", i)
			}
			if path, _ := fields["svgPath"].(string); strings.TrimSpace(path) == "" {
				return fmt.Sprintf("items[%d].svgPath is required", i)
			}
			if msg := validateSVGFloatPrecision(fields); msg != "" {
				return fmt.Sprintf("items[%d]: %s", i, msg)
			}
		}
		if msg := validateSVGFloatPrecision(params); msg != "" {
			return msg
		}
	}

	return ""
}


func validateSVGFloatPrecision(fields map[string]interface{}) string {
	precision, ok := fields["floatPrecision"].(float64)
	if !ok {
		return ""
	}
	if precision < 1 || precision > 6 || precision != float64(int(precision)) {
		return "floatPrecision must be an integer between 1 and 6"
	}
	return ""
}

// validateDetail checks the shared `detail` verbosity param. The node-lookup
// tools accept depth -1 for "unlimited", so depth range checks stay per-tool.
func validateDetail(params map[string]interface{}) string {
	detail, ok := params["detail"].(string)
	if !ok || detail == "" {
		return ""
	}
	switch detail {
	case "minimal", "compact", "full":
		return ""
	}
	return fmt.Sprintf("detail must be minimal, compact, or full, got: %s", detail)
}

func validExportFormat(f string) bool {
	switch f {
	case "PNG", "SVG", "JPG", "PDF":
		return true
	}
	return false
}
