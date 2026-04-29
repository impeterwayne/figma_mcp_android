package internal

import (
	"testing"
)

// ── ValidNodeID ──────────────────────────────────────────────────────────────

func TestValidNodeID(t *testing.T) {
	valid := []string{
		"4029:12345",
		"0:1",
		"1:1",
		"I44:9;44:3",
		"I2167:9091;186:1579;186:1745",
	}
	for _, id := range valid {
		if !ValidNodeID(id) {
			t.Errorf("expected %q to be valid", id)
		}
	}

	invalid := []string{
		"",
		"4029-12345",
		"4029:12345:6789",
		"abc:def",
		"4029:",
		":12345",
		"4029",
	}
	for _, id := range invalid {
		if ValidNodeID(id) {
			t.Errorf("expected %q to be invalid", id)
		}
	}
}

// ── NormalizeNodeID ───────────────────────────────────────────────────────────

func TestNormalizeNodeID(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"4029-12345", "4029:12345"},
		{"4029:12345", "4029:12345"},       // already valid, no-op
		{"not-a-node-id", "not-a-node-id"}, // hyphen but not a node ID
		{"", ""},
	}
	for _, c := range cases {
		got := NormalizeNodeID(c.input)
		if got != c.want {
			t.Errorf("NormalizeNodeID(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

// ── ValidateRPC ───────────────────────────────────────────────────────────────

func TestValidateRPC_GetNode(t *testing.T) {
	// missing nodeId
	if msg := ValidateRPC("get_node", nil, nil); msg == "" {
		t.Error("expected error for missing nodeId")
	}
	// hyphen format
	if msg := ValidateRPC("get_node", []string{"4029-12345"}, nil); msg == "" {
		t.Error("expected error for hyphen nodeId")
	}
	// valid
	if msg := ValidateRPC("get_node", []string{"4029:12345"}, nil); msg != "" {
		t.Errorf("unexpected error: %s", msg)
	}
}

func TestValidateRPC_GetNodesInfo(t *testing.T) {
	if msg := ValidateRPC("get_nodes_info", nil, nil); msg == "" {
		t.Error("expected error for empty nodeIds")
	}
	if msg := ValidateRPC("get_nodes_info", []string{"bad"}, nil); msg == "" {
		t.Error("expected error for invalid nodeId")
	}
	if msg := ValidateRPC("get_nodes_info", []string{"1:1", "2:2"}, nil); msg != "" {
		t.Errorf("unexpected error: %s", msg)
	}
}

func TestValidateRPC_GetScreenshot(t *testing.T) {
	// invalid format
	msg := ValidateRPC("get_screenshot", []string{"1:1"}, map[string]interface{}{"format": "GIF"})
	if msg == "" {
		t.Error("expected error for invalid format")
	}
	// valid formats
	for _, f := range []string{"PNG", "SVG", "JPG", "PDF"} {
		msg := ValidateRPC("get_screenshot", []string{"1:1"}, map[string]interface{}{"format": f})
		if msg != "" {
			t.Errorf("unexpected error for format %s: %s", f, msg)
		}
	}
}

func TestValidateRPC_SaveScreenshots(t *testing.T) {
	// missing items
	if msg := ValidateRPC("save_screenshots", nil, nil); msg == "" {
		t.Error("expected error for missing items")
	}
	// empty items array
	msg := ValidateRPC("save_screenshots", nil, map[string]interface{}{
		"items": []interface{}{},
	})
	if msg == "" {
		t.Error("expected error for empty items")
	}
	// invalid nodeId in item
	msg = ValidateRPC("save_screenshots", nil, map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"nodeId": "bad", "outputPath": "out.png"},
		},
	})
	if msg == "" {
		t.Error("expected error for bad nodeId in item")
	}
	// missing outputPath
	msg = ValidateRPC("save_screenshots", nil, map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"nodeId": "1:1"},
		},
	})
	if msg == "" {
		t.Error("expected error for missing outputPath")
	}
	// valid
	msg = ValidateRPC("save_screenshots", nil, map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"nodeId": "1:1", "outputPath": "out.png"},
		},
	})
	if msg != "" {
		t.Errorf("unexpected error: %s", msg)
	}
}

func TestValidateRPC_GetDesignContext(t *testing.T) {
	// negative depth
	msg := ValidateRPC("get_design_context", nil, map[string]interface{}{"depth": float64(-1)})
	if msg == "" {
		t.Error("expected error for negative depth")
	}
	// invalid detail
	msg = ValidateRPC("get_design_context", nil, map[string]interface{}{"detail": "huge"})
	if msg == "" {
		t.Error("expected error for invalid detail")
	}
	// valid detail values
	for _, d := range []string{"minimal", "compact", "full"} {
		msg := ValidateRPC("get_design_context", nil, map[string]interface{}{"detail": d})
		if msg != "" {
			t.Errorf("unexpected error for detail %s: %s", d, msg)
		}
	}
}

func TestValidateRPC_SearchNodes(t *testing.T) {
	// missing query
	if msg := ValidateRPC("search_nodes", nil, nil); msg == "" {
		t.Error("expected error for missing query")
	}
	// invalid nodeId
	msg := ValidateRPC("search_nodes", nil, map[string]interface{}{
		"query":  "button",
		"nodeId": "bad",
	})
	if msg == "" {
		t.Error("expected error for bad nodeId")
	}
	// non-positive limit
	msg = ValidateRPC("search_nodes", nil, map[string]interface{}{
		"query": "button",
		"limit": float64(0),
	})
	if msg == "" {
		t.Error("expected error for zero limit")
	}
	// valid
	msg = ValidateRPC("search_nodes", nil, map[string]interface{}{"query": "button"})
	if msg != "" {
		t.Errorf("unexpected error: %s", msg)
	}
}

func TestValidateRPC_GetReactions(t *testing.T) {
	if msg := ValidateRPC("get_reactions", nil, nil); msg == "" {
		t.Error("expected error for missing nodeId")
	}
	if msg := ValidateRPC("get_reactions", []string{"bad-id"}, nil); msg == "" {
		t.Error("expected error for hyphen nodeId")
	}
	if msg := ValidateRPC("get_reactions", []string{"1:1"}, nil); msg != "" {
		t.Errorf("unexpected error: %s", msg)
	}
}

func TestValidateRPC_ScanTextNodes(t *testing.T) {
	if msg := ValidateRPC("scan_text_nodes", nil, nil); msg == "" {
		t.Error("expected error for missing nodeId")
	}
	if msg := ValidateRPC("scan_text_nodes", nil, map[string]interface{}{"nodeId": "bad"}); msg == "" {
		t.Error("expected error for invalid nodeId")
	}
	if msg := ValidateRPC("scan_text_nodes", nil, map[string]interface{}{"nodeId": "1:1"}); msg != "" {
		t.Errorf("unexpected error: %s", msg)
	}
}

func TestValidateRPC_ScanNodesByTypes(t *testing.T) {
	if msg := ValidateRPC("scan_nodes_by_types", nil, nil); msg == "" {
		t.Error("expected error for missing nodeId")
	}
	// missing types
	msg := ValidateRPC("scan_nodes_by_types", nil, map[string]interface{}{"nodeId": "1:1"})
	if msg == "" {
		t.Error("expected error for missing types")
	}
	// valid
	msg = ValidateRPC("scan_nodes_by_types", nil, map[string]interface{}{
		"nodeId": "1:1",
		"types":  []interface{}{"FRAME"},
	})
	if msg != "" {
		t.Errorf("unexpected error: %s", msg)
	}
}

func TestValidateRPC_UnknownTool(t *testing.T) {
	// unknown tools pass through with no error
	msg := ValidateRPC("unknown_tool", nil, nil)
	if msg != "" {
		t.Errorf("expected no error for unknown tool, got: %s", msg)
	}
}

func TestValidateRPC_ExportTokens(t *testing.T) {
	// no params — valid (defaults to json)
	if msg := ValidateRPC("export_tokens", nil, nil); msg != "" {
		t.Errorf("unexpected error for no params: %s", msg)
	}
	// valid formats
	for _, f := range []string{"json", "css"} {
		if msg := ValidateRPC("export_tokens", nil, map[string]interface{}{"format": f}); msg != "" {
			t.Errorf("unexpected error for format %s: %s", f, msg)
		}
	}
	// invalid format
	if msg := ValidateRPC("export_tokens", nil, map[string]interface{}{"format": "yaml"}); msg == "" {
		t.Error("expected error for invalid format")
	}
	if msg := ValidateRPC("export_tokens", nil, map[string]interface{}{"format": "style-dictionary"}); msg == "" {
		t.Error("expected error for unsupported format")
	}
}
