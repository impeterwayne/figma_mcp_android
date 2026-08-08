import { describe, it, expect, beforeEach } from "bun:test";
import {
  isMixed,
  toHex,
  serializePaints,
  getBounds,
  deduplicateStyles,
  serializeVariableValue,
  serializeLineHeight,
  serializeLetterSpacing,
  serializeStyles,
  serializeText,
  serializeNode,
  sanitizeSymbols,
  serializeReactions,
} from "./serializers";

// ── Figma global mock ─────────────────────────────────────────────────────────

let mockGetStyleByIdAsync: (id: string) => Promise<{ name: string } | null>;

beforeEach(() => {
  mockGetStyleByIdAsync = async (_id: string) => null;
  (globalThis as any).figma = {
    getStyleByIdAsync: (id: string) => mockGetStyleByIdAsync(id),
  };
});

// ── isMixed ──────────────────────────────────────────────────────────────────

describe("isMixed", () => {
  it("returns true for symbols", () => {
    expect(isMixed(Symbol())).toBe(true);
  });
  it("returns false for non-symbols", () => {
    expect(isMixed(14)).toBe(false);
    expect(isMixed("hello")).toBe(false);
    expect(isMixed(null)).toBe(false);
    expect(isMixed(undefined)).toBe(false);
  });
});

// ── toHex ────────────────────────────────────────────────────────────────────

describe("toHex", () => {
  it("converts full white", () => {
    expect(toHex({ r: 1, g: 1, b: 1 })).toBe("#ffffff");
  });
  it("converts full black", () => {
    expect(toHex({ r: 0, g: 0, b: 0 })).toBe("#000000");
  });
  it("converts a mid-range color", () => {
    expect(toHex({ r: 1, g: 0, b: 0 })).toBe("#ff0000");
  });
  it("clamps values above 1", () => {
    expect(toHex({ r: 2, g: 0, b: 0 })).toBe("#ff0000");
  });
  it("clamps values below 0", () => {
    expect(toHex({ r: -1, g: 0, b: 0 })).toBe("#000000");
  });
  it("rounds fractional values", () => {
    // 0.5 * 255 = 127.5 → rounds to 128 = 0x80
    expect(toHex({ r: 0.5, g: 0.5, b: 0.5 })).toBe("#808080");
  });
});

// ── serializePaints ───────────────────────────────────────────────────────────

describe("serializePaints", () => {
  it("returns 'mixed' for symbol input", () => {
    expect(serializePaints(Symbol())).toBe("mixed");
  });
  it("returns undefined for null/non-array", () => {
    expect(serializePaints(null)).toBeUndefined();
    expect(serializePaints("red")).toBeUndefined();
  });
  it("returns undefined for empty array", () => {
    expect(serializePaints([])).toBeUndefined();
  });
  it("serializes gradient paints with extracted gradient properties", () => {
    const paints = [
      {
        type: "GRADIENT_LINEAR",
        gradientStops: [
          { position: 0, color: { r: 1, g: 0, b: 0, a: 1 } },
          { position: 1, color: { r: 0, g: 0, b: 1, a: 0.5 } },
        ],
        gradientTransform: [
          [1, 0, 0],
          [0, 1, 0],
        ],
      },
    ];
    expect(serializePaints(paints)).toEqual([
      {
        type: "GRADIENT_LINEAR",
        gradientStops: [
          { position: 0, color: "#ff0000" },
          { position: 1, color: "#0000ff80" },
        ],
        gradientTransform: [
          [1, 0, 0],
          [0, 1, 0],
        ],
      },
    ]);
  });
  it("serializes image paints with image properties", () => {
    const paints = [{ type: "IMAGE", scaleMode: "FILL", imageHash: "abc123hash" }];
    expect(serializePaints(paints)).toEqual([
      { type: "IMAGE", scaleMode: "FILL", imageHash: "abc123hash" },
    ]);
  });
  it("serializes a solid paint with opacity 1 as plain hex", () => {
    const paints = [{ type: "SOLID", color: { r: 1, g: 0, b: 0 }, opacity: 1 }];
    expect(serializePaints(paints)).toEqual(["#ff0000"]);
  });
  it("appends alpha hex when opacity < 1", () => {
    // opacity 0.5 → Math.round(0.5 * 255) = 128 = 0x80
    const paints = [{ type: "SOLID", color: { r: 1, g: 0, b: 0 }, opacity: 0.5 }];
    const result = serializePaints(paints) as string[];
    expect(result[0]).toBe("#ff000080");
  });
  it("defaults opacity to 1 when not provided", () => {
    const paints = [{ type: "SOLID", color: { r: 0, g: 0, b: 1 } }];
    expect(serializePaints(paints)).toEqual(["#0000ff"]);
  });
  it("serializes multiple solid paints", () => {
    const paints = [
      { type: "SOLID", color: { r: 1, g: 0, b: 0 } },
      { type: "SOLID", color: { r: 0, g: 1, b: 0 } },
    ];
    expect(serializePaints(paints)).toEqual(["#ff0000", "#00ff00"]);
  });
});

// ── getBounds ─────────────────────────────────────────────────────────────────

describe("getBounds", () => {
  it("returns bounds for a node with x/y/width/height", () => {
    expect(getBounds({ x: 10, y: 20, width: 100, height: 50 })).toEqual({
      x: 10, y: 20, width: 100, height: 50,
    });
  });
  it("rounds floating point values to 2 decimal places", () => {
    const bounds = getBounds({ x: 10.999, y: 0, width: 99.999, height: 50 });
    expect(bounds?.x).toBe(11);
    expect(bounds?.width).toBe(100);
  });
  it("returns undefined when coordinates are missing", () => {
    expect(getBounds({ name: "page" })).toBeUndefined();
    expect(getBounds({ x: 0, y: 0 })).toBeUndefined();
  });
});

// ── serializeLineHeight ───────────────────────────────────────────────────────

describe("serializeLineHeight", () => {
  it("returns 'mixed' for symbol", () => {
    expect(serializeLineHeight(Symbol())).toBe("mixed");
  });
  it("returns undefined for AUTO unit", () => {
    expect(serializeLineHeight({ unit: "AUTO" })).toBeUndefined();
  });
  it("returns undefined for null/falsy", () => {
    expect(serializeLineHeight(null)).toBeUndefined();
    expect(serializeLineHeight(undefined)).toBeUndefined();
  });
  it("returns value and unit for PIXELS", () => {
    expect(serializeLineHeight({ value: 24, unit: "PIXELS" })).toEqual({ value: 24, unit: "PIXELS" });
  });
  it("returns value and unit for PERCENT", () => {
    expect(serializeLineHeight({ value: 150, unit: "PERCENT" })).toEqual({ value: 150, unit: "PERCENT" });
  });
});

// ── serializeLetterSpacing ────────────────────────────────────────────────────

describe("serializeLetterSpacing", () => {
  it("returns 'mixed' for symbol", () => {
    expect(serializeLetterSpacing(Symbol())).toBe("mixed");
  });
  it("returns undefined when value is 0", () => {
    expect(serializeLetterSpacing({ value: 0, unit: "PIXELS" })).toBeUndefined();
  });
  it("returns undefined for null/falsy", () => {
    expect(serializeLetterSpacing(null)).toBeUndefined();
  });
  it("returns value and unit for non-zero spacing", () => {
    expect(serializeLetterSpacing({ value: 1.5, unit: "PIXELS" })).toEqual({ value: 1.5, unit: "PIXELS" });
  });
});

// ── deduplicateStyles ─────────────────────────────────────────────────────────

describe("deduplicateStyles", () => {
  it("returns original tree and undefined globalVars when nothing is repeated", () => {
    const tree = {
      children: [
        { styles: { fills: ["#ff0000"] } },
        { styles: { fills: ["#00ff00"] } },
      ],
    };
    const { tree: result, globalVars } = deduplicateStyles(tree);
    expect(globalVars).toBeUndefined();
    expect(result).toBe(tree);
  });

  it("deduplicates fills that appear more than once", () => {
    const sharedFill = ["#ff0000"];
    const tree = {
      children: [
        { styles: { fills: sharedFill } },
        { styles: { fills: sharedFill } },
      ],
    };
    const { tree: result, globalVars } = deduplicateStyles(tree);
    expect(globalVars).toBeDefined();
    const refs = Object.keys(globalVars!.styles);
    expect(refs.length).toBe(1);
    // Both nodes should now reference the short key instead of the array
    const children = (result as any).children;
    expect(typeof children[0].styles.fills).toBe("string");
    expect(children[0].styles.fills).toBe(children[1].styles.fills);
  });

  it("deduplicates strokes that appear more than once", () => {
    const sharedStroke = ["#0000ff"];
    const tree = {
      children: [
        { styles: { strokes: sharedStroke } },
        { styles: { strokes: sharedStroke } },
      ],
    };
    const { globalVars } = deduplicateStyles(tree);
    expect(globalVars).toBeDefined();
  });

  it("preserves unique fills as-is", () => {
    const tree = {
      children: [
        { styles: { fills: ["#ff0000"] } },
        { styles: { fills: ["#00ff00"] } },
        { styles: { fills: ["#ff0000"] } },
        { styles: { fills: ["#00ff00"] } },
      ],
    };
    const { globalVars } = deduplicateStyles(tree);
    // Both colors appear twice so both should be deduped
    expect(Object.keys(globalVars!.styles).length).toBe(2);
  });

  it("handles empty tree without errors", () => {
    const { tree, globalVars } = deduplicateStyles({});
    expect(globalVars).toBeUndefined();
    expect(tree).toEqual({});
  });
});

// ── serializeVariableValue ────────────────────────────────────────────────────

describe("serializeVariableValue", () => {
  it("passes through primitives unchanged", () => {
    expect(serializeVariableValue(42)).toBe(42);
    expect(serializeVariableValue("hello")).toBe("hello");
    expect(serializeVariableValue(true)).toBe(true);
    expect(serializeVariableValue(null)).toBe(null);
  });

  it("serializes VARIABLE_ALIAS objects", () => {
    const val = { type: "VARIABLE_ALIAS", id: "abc123", extra: "ignored" };
    expect(serializeVariableValue(val)).toEqual({ type: "VARIABLE_ALIAS", id: "abc123" });
  });

  it("serializes color objects to COLOR type", () => {
    const val = { r: 1, g: 0, b: 0, a: 1 };
    expect(serializeVariableValue(val)).toEqual({ type: "COLOR", r: 1, g: 0, b: 0, a: 1 });
  });

  it("defaults alpha to 1 when missing from color", () => {
    const val = { r: 0, g: 1, b: 0 };
    expect(serializeVariableValue(val)).toEqual({ type: "COLOR", r: 0, g: 1, b: 0, a: 1 });
  });

  it("passes through unknown objects unchanged", () => {
    const val = { foo: "bar" };
    expect(serializeVariableValue(val)).toEqual({ foo: "bar" });
  });
});

// ── serializeStyles ───────────────────────────────────────────────────────────

describe("serializeStyles", () => {
  it("returns empty object for node with no relevant properties", async () => {
    const result = await serializeStyles({ id: "1", name: "box" });
    expect(result).toEqual({});
  });

  it("includes fills when fills is a solid paint array", async () => {
    const node = { fills: [{ type: "SOLID", color: { r: 1, g: 0, b: 0 } }] };
    const result = await serializeStyles(node);
    expect(result.fills).toEqual(["#ff0000"]);
  });

  it("includes fillStyle name when fillStyleId resolves to a style", async () => {
    mockGetStyleByIdAsync = async (id) => (id === "style-1" ? { name: "Red" } : null);
    const node = {
      fills: [{ type: "SOLID", color: { r: 1, g: 0, b: 0 } }],
      fillStyleId: "style-1",
    };
    const result = await serializeStyles(node);
    expect(result.fillStyle).toBe("Red");
    expect(result.fills).toEqual(["#ff0000"]);
  });

  it("skips fillStyle when fillStyleId resolves to null", async () => {
    const node = {
      fills: [{ type: "SOLID", color: { r: 1, g: 0, b: 0 } }],
      fillStyleId: "missing",
    };
    const result = await serializeStyles(node);
    expect(result.fillStyle).toBeUndefined();
    expect(result.fills).toEqual(["#ff0000"]);
  });

  it("skips fillStyle when fillStyleId is not a string", async () => {
    const node = {
      fills: [{ type: "SOLID", color: { r: 0, g: 0, b: 1 } }],
      fillStyleId: Symbol(),
    };
    const result = await serializeStyles(node);
    expect(result.fillStyle).toBeUndefined();
  });

  it("includes strokes and strokeStyle", async () => {
    mockGetStyleByIdAsync = async (id) => (id === "s-1" ? { name: "Border" } : null);
    const node = {
      strokes: [{ type: "SOLID", color: { r: 0, g: 0, b: 0 } }],
      strokeStyleId: "s-1",
    };
    const result = await serializeStyles(node);
    expect(result.strokeStyle).toBe("Border");
    expect(result.strokes).toEqual(["#000000"]);
  });

  it("omits cornerRadius when value is 0", async () => {
    const result = await serializeStyles({ cornerRadius: 0 });
    expect(result.cornerRadius).toBeUndefined();
  });

  it("includes cornerRadius when non-zero", async () => {
    const result = await serializeStyles({ cornerRadius: 8 });
    expect(result.cornerRadius).toBe(8);
  });

  it("extracts individual corner radii when cornerRadius is mixed", async () => {
    const node = {
      cornerRadius: Symbol(),
      topLeftRadius: 10,
      topRightRadius: 20,
      bottomRightRadius: 30,
      bottomLeftRadius: 40,
    };
    const result = await serializeStyles(node);
    expect(result.cornerRadius).toEqual({
      topLeft: 10,
      topRight: 20,
      bottomRight: 30,
      bottomLeft: 40,
    });
  });

  it("includes padding when paddingLeft is present", async () => {
    const node = { paddingLeft: 10, paddingRight: 20, paddingTop: 5, paddingBottom: 15 };
    const result = await serializeStyles(node);
    expect(result.padding).toEqual({ top: 5, right: 20, bottom: 15, left: 10 });
  });
});

// ── serializeText ─────────────────────────────────────────────────────────────

describe("serializeText", () => {
  const makeBase = () => ({ id: "t1", name: "Text", type: "TEXT", bounds: undefined, styles: {} });

  it("handles mixed font name", async () => {
    const node = {
      fontName: Symbol(),
      fontSize: 16,
      fontWeight: 400,
      textDecoration: "NONE",
      lineHeight: { unit: "AUTO" },
      letterSpacing: { value: 0, unit: "PIXELS" },
      textAlignHorizontal: "LEFT",
      characters: "hello",
    };
    const result = await serializeText(node, makeBase());
    expect(result.styles.fontFamily).toBe("mixed");
    expect(result.styles.fontStyle).toBe("mixed");
  });

  it("handles regular font name", async () => {
    const node = {
      fontName: { family: "Inter", style: "Regular" },
      fontSize: 14,
      fontWeight: 400,
      textDecoration: "NONE",
      lineHeight: { unit: "AUTO" },
      letterSpacing: { value: 0, unit: "PIXELS" },
      textAlignHorizontal: "LEFT",
      characters: "hello",
    };
    const result = await serializeText(node, makeBase());
    expect(result.styles.fontFamily).toBe("Inter");
    expect(result.styles.fontStyle).toBe("Regular");
    expect(result.characters).toBe("hello");
  });

  it("includes textStyle when textStyleId resolves", async () => {
    mockGetStyleByIdAsync = async (id) => (id === "ts-1" ? { name: "Heading 1" } : null);
    const node = {
      fontName: { family: "Inter", style: "Bold" },
      fontSize: 32,
      fontWeight: 700,
      textDecoration: "NONE",
      lineHeight: { unit: "AUTO" },
      letterSpacing: { value: 0, unit: "PIXELS" },
      textAlignHorizontal: "LEFT",
      textStyleId: "ts-1",
      characters: "Title",
    };
    const result = await serializeText(node, makeBase());
    expect(result.styles.textStyle).toBe("Heading 1");
  });

  it("omits textStyle when textStyleId is not a string", async () => {
    const node = {
      fontName: { family: "Inter", style: "Regular" },
      fontSize: 14,
      fontWeight: 400,
      textDecoration: "NONE",
      lineHeight: { unit: "AUTO" },
      letterSpacing: { value: 0, unit: "PIXELS" },
      textAlignHorizontal: "LEFT",
      textStyleId: Symbol(),
      characters: "hi",
    };
    const result = await serializeText(node, makeBase());
    expect(result.styles.textStyle).toBeUndefined();
  });

  it("serializes mixed text properties", async () => {
    const node = {
      fontName: { family: "Inter", style: "Regular" },
      fontSize: Symbol(),
      fontWeight: Symbol(),
      textDecoration: Symbol(),
      lineHeight: { unit: "AUTO" },
      letterSpacing: { value: 0, unit: "PIXELS" },
      textAlignHorizontal: Symbol(),
      characters: "mixed",
    };
    const result = await serializeText(node, makeBase());
    expect(result.styles.fontSize).toBe("mixed");
    expect(result.styles.fontWeight).toBe("mixed");
    expect(result.styles.textDecoration).toBe("mixed");
    expect(result.styles.textAlignHorizontal).toBe("mixed");
  });

  it("omits textDecoration when value is NONE", async () => {
    const node = {
      fontName: { family: "Inter", style: "Regular" },
      fontSize: 14,
      fontWeight: 400,
      textDecoration: "NONE",
      lineHeight: { unit: "AUTO" },
      letterSpacing: { value: 0, unit: "PIXELS" },
      textAlignHorizontal: "LEFT",
      characters: "plain",
    };
    const result = await serializeText(node, makeBase());
    expect(result.styles.textDecoration).toBeUndefined();
  });

  it("includes textDecoration when not NONE", async () => {
    const node = {
      fontName: { family: "Inter", style: "Regular" },
      fontSize: 14,
      fontWeight: 400,
      textDecoration: "UNDERLINE",
      lineHeight: { unit: "AUTO" },
      letterSpacing: { value: 0, unit: "PIXELS" },
      textAlignHorizontal: "LEFT",
      characters: "underlined",
    };
    const result = await serializeText(node, makeBase());
    expect(result.styles.textDecoration).toBe("UNDERLINE");
  });
});

// ── serializeNode ─────────────────────────────────────────────────────────────

describe("serializeNode", () => {
  it("serializes a plain node with bounds", async () => {
    const node = { id: "1:1", name: "Box", type: "RECTANGLE", x: 0, y: 0, width: 100, height: 50 };
    const result = await serializeNode(node);
    expect(result.id).toBe("1:1");
    expect(result.type).toBe("RECTANGLE");
    expect(result.bounds).toEqual({ x: 0, y: 0, width: 100, height: 50 });
  });

  it("serializes a TEXT node", async () => {
    const node = {
      id: "1:2",
      name: "Label",
      type: "TEXT",
      x: 0, y: 0, width: 50, height: 20,
      fontName: { family: "Inter", style: "Regular" },
      fontSize: 14,
      fontWeight: 400,
      textDecoration: "NONE",
      lineHeight: { unit: "AUTO" },
      letterSpacing: { value: 0, unit: "PIXELS" },
      textAlignHorizontal: "LEFT",
      characters: "Hello",
    };
    const result = await serializeNode(node);
    expect(result.type).toBe("TEXT");
    expect(result.characters).toBe("Hello");
  });

  it("recursively serializes children", async () => {
    const node = {
      id: "1:3",
      name: "Frame",
      type: "FRAME",
      x: 0, y: 0, width: 200, height: 200,
      children: [
        { id: "1:4", name: "Child", type: "RECTANGLE", x: 10, y: 10, width: 50, height: 50 },
      ],
    };
    const result = await serializeNode(node);
    expect(result.children).toHaveLength(1);
    expect(result.children[0].id).toBe("1:4");
  });
});

describe("sanitizeSymbols", () => {
  it("converts top-level symbol to 'mixed'", () => {
    expect(sanitizeSymbols(Symbol("mixed"))).toBe("mixed");
  });

  it("converts nested symbols in objects and arrays to 'mixed'", () => {
    const input = {
      fontName: Symbol("mixed"),
      styles: {
        fontSize: 16,
        fontWeight: Symbol("mixed"),
      },
      list: [Symbol("mixed"), 123, "hello"],
    };
    expect(sanitizeSymbols(input)).toEqual({
      fontName: "mixed",
      styles: {
        fontSize: 16,
        fontWeight: "mixed",
      },
      list: ["mixed", 123, "hello"],
    });
  });

  it("passes primitive values unchanged", () => {
    expect(sanitizeSymbols("hello")).toBe("hello");
    expect(sanitizeSymbols(123)).toBe(123);
    expect(sanitizeSymbols(null)).toBeNull();
  });

  it("returns a symbol-free payload by reference instead of cloning it", () => {
    const input = { data: { nodes: [{ id: "1:1", styles: { fills: ["#ffffff"] } }] } };
    expect(sanitizeSymbols(input)).toBe(input);
  });

  it("clones only the branches that contain a symbol", () => {
    const clean = { id: "1:2", styles: { fills: ["#000000"] } };
    const input = { data: { clean, dirty: { id: "1:3", fontName: Symbol("mixed") } } };

    const result = sanitizeSymbols(input);
    expect(result).not.toBe(input);
    expect(result.data.clean).toBe(clean);
    expect(result.data.dirty).toEqual({ id: "1:3", fontName: "mixed" });
  });

  it("reports a throwing getter as 'mixed'", () => {
    const input = {
      id: "1:1",
      get fills(): any {
        throw new Error("node removed");
      },
    };
    expect(sanitizeSymbols(input)).toEqual({ id: "1:1", fills: "mixed" });
  });
});

// ── mixed-property handling ──────────────────────────────────────────────────

describe("mixed stroke properties", () => {
  const strokedNode = (overrides: any) => ({
    id: "1:1",
    name: "Card",
    type: "FRAME",
    x: 0,
    y: 0,
    width: 100,
    height: 50,
    rotation: 0,
    strokes: [{ type: "SOLID", color: { r: 0, g: 0, b: 0 }, opacity: 1 }],
    dashPattern: [],
    ...overrides,
  });

  it("expands mixed strokeWeight into per-side weights", async () => {
    const styles = await serializeStyles(
      strokedNode({
        strokeWeight: Symbol("mixed"),
        strokeTopWeight: 0,
        strokeRightWeight: 0,
        strokeBottomWeight: 1,
        strokeLeftWeight: 0,
      }),
    );
    expect(styles.strokeWeight).toEqual({ top: 0, right: 0, bottom: 1, left: 0 });
  });

  it("keeps a uniform strokeWeight as a number", async () => {
    const styles = await serializeStyles(strokedNode({ strokeWeight: 2 }));
    expect(styles.strokeWeight).toBe(2);
  });

  it("omits a zero strokeWeight", async () => {
    const styles = await serializeStyles(strokedNode({ strokeWeight: 0 }));
    expect(styles.strokeWeight).toBeUndefined();
  });

  it("reports mixed strokeCap and strokeJoin as 'mixed'", async () => {
    const styles = await serializeStyles(
      strokedNode({
        strokeWeight: 1.5,
        strokeCap: Symbol("mixed"),
        strokeJoin: Symbol("mixed"),
      }),
    );
    expect(styles.strokeCap).toBe("mixed");
    expect(styles.strokeJoin).toBe("mixed");
  });
});

// Invariant: no Symbol may survive serialization. figma.mixed is a Symbol and
// figma.ui.postMessage throws "Cannot unwrap symbol" on it, so a single leaked
// property fails the whole request. This guards every mixable property at once,
// including ones added later.
describe("serializeNode symbol invariant", () => {
  const findSymbols = (value: any, path = "$", out: string[] = []): string[] => {
    if (typeof value === "symbol") out.push(path);
    else if (value && typeof value === "object")
      for (const key of Object.keys(value)) findSymbols(value[key], `${path}.${key}`, out);
    return out;
  };

  const MIXED = () => Symbol("figma.mixed");

  it("emits no symbols for a node with every mixable property mixed", async () => {
    const node: any = {
      id: "1:1",
      name: "All Mixed",
      type: "TEXT",
      x: 0,
      y: 0,
      width: 100,
      height: 20,
      rotation: 0,
      fills: MIXED(),
      strokes: MIXED(),
      strokeWeight: MIXED(),
      strokeTopWeight: 0,
      strokeRightWeight: 0,
      strokeBottomWeight: 1,
      strokeLeftWeight: 0,
      strokeCap: MIXED(),
      strokeJoin: MIXED(),
      dashPattern: [],
      effects: MIXED(),
      cornerRadius: MIXED(),
      topLeftRadius: 4,
      topRightRadius: 4,
      bottomRightRadius: 0,
      bottomLeftRadius: 0,
      opacity: 1,
      characters: "hello",
      fontSize: MIXED(),
      fontName: MIXED(),
      fontWeight: MIXED(),
      textDecoration: MIXED(),
      lineHeight: MIXED(),
      letterSpacing: MIXED(),
      textAlignHorizontal: MIXED(),
      textAlignVertical: MIXED(),
      textAutoResize: "NONE",
      paragraphSpacing: 0,
      paragraphIndent: 0,
    };

    const result = await serializeNode(node);
    expect(findSymbols(result)).toEqual([]);
  });

  it("emits no symbols for mixed properties nested in children", async () => {
    const child = {
      id: "1:2",
      name: "Icon",
      type: "VECTOR",
      x: 0,
      y: 0,
      width: 24,
      height: 24,
      rotation: 0,
      strokes: [{ type: "SOLID", color: { r: 0, g: 0, b: 0 }, opacity: 1 }],
      strokeWeight: 1.5,
      strokeCap: MIXED(),
      strokeJoin: MIXED(),
      dashPattern: [],
      vectorPaths: [{ windingRule: "NONZERO", data: "M0 0L24 24" }],
    };
    const parent: any = {
      id: "1:1",
      name: "Row",
      type: "FRAME",
      x: 0,
      y: 0,
      width: 100,
      height: 50,
      rotation: 0,
      fills: MIXED(),
      children: [child],
    };

    const result = await serializeNode(parent);
    expect(findSymbols(result)).toEqual([]);
    expect(result.children[0].styles.strokeCap).toBe("mixed");
  });
});

// ── serializeReactions ───────────────────────────────────────────────────────

describe("serializeReactions", () => {
  it("keeps every entry of the actions array", () => {
    const result = serializeReactions([
      {
        trigger: { type: "ON_CLICK" },
        actions: [
          { type: "NODE", destinationId: "2:5", navigation: "NAVIGATE", transition: { type: "SMART_ANIMATE" } },
          { type: "URL", url: "https://example.com" },
        ],
      },
    ]);
    expect(result).toEqual([
      {
        actions: [
          { type: "NODE", destinationId: "2:5", navigation: "NAVIGATE", transition: { type: "SMART_ANIMATE" } },
          { type: "URL", url: "https://example.com" },
        ],
        trigger: { type: "ON_CLICK" },
      },
    ]);
  });

  it("falls back to the deprecated single action field", () => {
    const result = serializeReactions([
      { trigger: { type: "ON_HOVER" }, action: { type: "BACK" } },
    ]);
    expect(result).toEqual([{ actions: [{ type: "BACK" }], trigger: { type: "ON_HOVER" } }]);
  });

  it("keeps the trigger delay and omits actions when there are none", () => {
    const result = serializeReactions([
      { trigger: { type: "AFTER_TIMEOUT", delay: 2 }, actions: [] },
    ]);
    expect(result).toEqual([{ trigger: { type: "AFTER_TIMEOUT", delay: 2 } }]);
  });

  it("returns undefined for mixed, empty, or missing reactions", () => {
    expect(serializeReactions(Symbol("mixed"))).toBeUndefined();
    expect(serializeReactions([])).toBeUndefined();
    expect(serializeReactions(undefined)).toBeUndefined();
  });

  it("surfaces reactions on a serialized node", async () => {
    const result = await serializeNode({
      id: "1:1",
      name: "Button",
      type: "FRAME",
      x: 0,
      y: 0,
      width: 100,
      height: 40,
      rotation: 0,
      reactions: [{ trigger: { type: "ON_CLICK" }, actions: [{ type: "NODE", destinationId: "2:5" }] }],
    });
    expect(result.reactions).toEqual([
      { actions: [{ type: "NODE", destinationId: "2:5" }], trigger: { type: "ON_CLICK" } },
    ]);
  });
});

// ── failure containment ──────────────────────────────────────────────────────

describe("serializeNode failure containment", () => {
  const baseProps = { x: 0, y: 0, width: 10, height: 10, rotation: 0 };

  it("degrades a failing child to an identity stub and keeps its siblings", async () => {
    const parent: any = {
      id: "1:1",
      name: "Row",
      type: "FRAME",
      ...baseProps,
      children: [
        {
          id: "1:2",
          name: "Broken",
          type: "RECTANGLE",
          ...baseProps,
          get fills(): any {
            throw new Error("node removed");
          },
        },
        { id: "1:3", name: "Ok", type: "RECTANGLE", ...baseProps },
      ],
    };

    const result = await serializeNode(parent);
    expect(result.children).toHaveLength(2);
    expect(result.children[0]).toEqual({ id: "1:2", name: "Broken", type: "RECTANGLE" });
    expect(result.children[1].name).toBe("Ok");
  });

  it("serializes a variant COMPONENT whose componentPropertyDefinitions throws", async () => {
    const variant: any = {
      id: "1:1",
      name: "Size=Large",
      type: "COMPONENT",
      ...baseProps,
      get componentPropertyDefinitions(): any {
        throw new Error("Cannot get componentPropertyDefinitions of a variant");
      },
    };

    const result = await serializeNode(variant);
    expect(result.id).toBe("1:1");
    expect(result.componentPropertyDefinitions).toBeUndefined();
  });
});

