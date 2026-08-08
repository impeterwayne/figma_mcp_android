// Serializers — shared read/write helpers for converting Figma node data to JSON.

export const isMixed = (value: any) => typeof value === "symbol";

// Round floating-point pixel values to 2 decimal places.
// Figma sometimes returns values like 123.99999999999999 instead of 124.
const pixelRound = (v: number) => Math.round(v * 100) / 100;

export const toHex = (color: any) => {
  const clamp = (v: any) => Math.min(255, Math.max(0, Math.round(v * 255)));
  const [r, g, b] = [clamp(color.r), clamp(color.g), clamp(color.b)];
  return `#${[r, g, b].map((v) => v.toString(16).padStart(2, "0")).join("")}`;
};

// #rrggbb, plus an aa suffix when not fully opaque. Alpha defaults to the
// color's own `a`; paints pass their separate `opacity` instead.
export const toHexWithAlpha = (color: any, alpha?: number) => {
  const a = alpha != null ? alpha : color && color.a != null ? color.a : 1;
  const hex = toHex(color);
  if (a >= 1) return hex;
  return (
    hex +
    Math.round(a * 255)
      .toString(16)
      .padStart(2, "0")
  );
};

export const serializePaints = (paints: any) => {
  if (isMixed(paints)) return "mixed";

  if (!paints || !Array.isArray(paints)) return undefined;

  const result = paints.map((paint: any) => {
    if (!paint || typeof paint !== "object") return paint;

    if (paint.type === "SOLID" && "color" in paint) {
      return toHexWithAlpha(paint.color, paint.opacity != null ? paint.opacity : 1);
    }

    if (paint.type && typeof paint.type === "string" && paint.type.startsWith("GRADIENT_")) {
      const gradientObj: any = {
        type: paint.type,
      };
      if (Array.isArray(paint.gradientStops)) {
        gradientObj.gradientStops = paint.gradientStops.map((stop: any) => ({
          position: pixelRound(stop.position ?? 0),
          color: stop.color ? toHexWithAlpha(stop.color) : "#000000",
        }));
      }
      if (paint.gradientTransform) {
        gradientObj.gradientTransform = paint.gradientTransform;
      }
      if (paint.opacity != null && paint.opacity < 1) {
        gradientObj.opacity = paint.opacity;
      }
      if (paint.blendMode && paint.blendMode !== "NORMAL") {
        gradientObj.blendMode = paint.blendMode;
      }
      if (paint.visible === false) {
        gradientObj.visible = false;
      }
      return gradientObj;
    }

    if (paint.type === "IMAGE") {
      const imageObj: any = { type: "IMAGE" };
      if (paint.scaleMode) imageObj.scaleMode = paint.scaleMode;
      if (paint.imageHash) imageObj.imageHash = paint.imageHash;
      if (paint.opacity != null && paint.opacity < 1) imageObj.opacity = paint.opacity;
      if (paint.blendMode && paint.blendMode !== "NORMAL") imageObj.blendMode = paint.blendMode;
      if (paint.visible === false) imageObj.visible = false;
      return imageObj;
    }

    const genericPaint: any = { type: paint.type };
    if (paint.opacity != null && paint.opacity < 1) genericPaint.opacity = paint.opacity;
    if (paint.blendMode && paint.blendMode !== "NORMAL") genericPaint.blendMode = paint.blendMode;
    if (paint.visible === false) genericPaint.visible = false;
    return genericPaint;
  });

  return result.length > 0 ? result : undefined;
};

export const getBounds = (node: any) => {
  if ("x" in node && "y" in node && "width" in node && "height" in node) {
    const bounds: any = {
      x: pixelRound(node.x),
      y: pixelRound(node.y),
      width: pixelRound(node.width),
      height: pixelRound(node.height),
    };
    if ("rotation" in node && node.rotation !== 0) {
      bounds.rotation = pixelRound(node.rotation);
    }
    if ("absoluteBoundingBox" in node && node.absoluteBoundingBox) {
      bounds.absoluteBoundingBox = {
        x: pixelRound(node.absoluteBoundingBox.x),
        y: pixelRound(node.absoluteBoundingBox.y),
        width: pixelRound(node.absoluteBoundingBox.width),
        height: pixelRound(node.absoluteBoundingBox.height),
      };
    }
    return bounds;
  }

  return undefined;
};

export const serializeStyles = async (node: any) => {
  const styles: any = {};

  if ("fills" in node) {
    // Prefer named style over raw fill values when a style is applied.
    if (node.fillStyleId && typeof node.fillStyleId === "string") {
      const style = await figma.getStyleByIdAsync(node.fillStyleId);
      if (style) styles.fillStyle = style.name;
    }
    const fills = serializePaints(node.fills);
    if (fills !== undefined) styles.fills = fills;
  }

  if ("strokes" in node) {
    if (node.strokeStyleId && typeof node.strokeStyleId === "string") {
      const style = await figma.getStyleByIdAsync(node.strokeStyleId);
      if (style) styles.strokeStyle = style.name;
    }
    const strokes = serializePaints(node.strokes);
    if (strokes !== undefined) styles.strokes = strokes;
    
    // strokeWeight is figma.mixed when sides differ (e.g. a bottom-only border).
    // Expand to per-side weights so a divider stays distinguishable from a full border.
    if ("strokeWeight" in node) {
      if (isMixed(node.strokeWeight)) {
        styles.strokeWeight = {
          top: node.strokeTopWeight,
          right: node.strokeRightWeight,
          bottom: node.strokeBottomWeight,
          left: node.strokeLeftWeight,
        };
      } else if (node.strokeWeight !== 0) {
        styles.strokeWeight = node.strokeWeight;
      }
    }
    if ("strokeAlign" in node && node.strokeAlign !== "INSIDE") styles.strokeAlign = node.strokeAlign;
    if ("strokeCap" in node) {
      if (isMixed(node.strokeCap)) styles.strokeCap = "mixed";
      else if (node.strokeCap !== "NONE") styles.strokeCap = node.strokeCap;
    }
    if ("strokeJoin" in node) {
      if (isMixed(node.strokeJoin)) styles.strokeJoin = "mixed";
      else if (node.strokeJoin !== "MITER") styles.strokeJoin = node.strokeJoin;
    }
    if ("dashPattern" in node && node.dashPattern.length > 0) styles.dashPattern = node.dashPattern;
  }

  if ("effects" in node) {
    const fx = serializeEffects(node.effects);
    if (fx) {
      if (node.effectStyleId && typeof node.effectStyleId === "string") {
        const style = await figma.getStyleByIdAsync(node.effectStyleId);
        if (style) styles.effectStyle = style.name;
      }
      styles.effects = fx;
    }
  }

  if ("opacity" in node && node.opacity < 1) {
    styles.opacity = node.opacity;
  }
  
  if ("blendMode" in node && node.blendMode !== "PASS_THROUGH" && node.blendMode !== "NORMAL") {
    styles.blendMode = node.blendMode;
  }

  if ("cornerRadius" in node) {
    const cr = isMixed(node.cornerRadius) ? "mixed" : node.cornerRadius;
    if (cr !== 0) {
      if (cr === "mixed") {
        styles.cornerRadius = {
          topLeft: node.topLeftRadius,
          topRight: node.topRightRadius,
          bottomRight: node.bottomRightRadius,
          bottomLeft: node.bottomLeftRadius,
        };
      } else {
        styles.cornerRadius = cr;
      }
    }
  }

  if ("paddingLeft" in node) {
    styles.padding = {
      top: node.paddingTop,
      right: node.paddingRight,
      bottom: node.paddingBottom,
      left: node.paddingLeft,
    };
  }

  return styles;
};

export const serializeLineHeight = (lineHeight: any) => {
  if (isMixed(lineHeight)) return "mixed";

  if (!lineHeight || lineHeight.unit === "AUTO") return undefined;

  return { value: lineHeight.value, unit: lineHeight.unit };
};

export const serializeLetterSpacing = (letterSpacing: any) => {
  if (isMixed(letterSpacing)) return "mixed";

  if (!letterSpacing || letterSpacing.value === 0) return undefined;

  return { value: letterSpacing.value, unit: letterSpacing.unit };
};

export const serializeText = async (node: any, base: any) => {
  let fontFamily: any;
  let fontStyle: any;

  if (typeof node.fontName === "symbol") {
    fontFamily = "mixed";
    fontStyle = "mixed";
  } else if (node.fontName) {
    fontFamily = node.fontName.family;
    fontStyle = node.fontName.style;
  }

  const textStyleName =
    node.textStyleId && typeof node.textStyleId === "string"
      ? ((await figma.getStyleByIdAsync(node.textStyleId))?.name ?? undefined)
      : undefined;

  return Object.assign({}, base, {
    characters: node.characters,
    styles: Object.assign({}, base.styles, {
      ...(textStyleName ? { textStyle: textStyleName } : {}),
      fontSize: isMixed(node.fontSize) ? "mixed" : node.fontSize,
      fontFamily,
      fontStyle,
      fontWeight: isMixed(node.fontWeight) ? "mixed" : node.fontWeight,
      textDecoration: isMixed(node.textDecoration)
        ? "mixed"
        : node.textDecoration !== "NONE"
          ? node.textDecoration
          : undefined,
      lineHeight: serializeLineHeight(node.lineHeight),
      letterSpacing: serializeLetterSpacing(node.letterSpacing),
      textAlignHorizontal: isMixed(node.textAlignHorizontal)
        ? "mixed"
        : node.textAlignHorizontal,
      textAlignVertical: isMixed(node.textAlignVertical)
        ? "mixed"
        : node.textAlignVertical,
      textAutoResize: node.textAutoResize !== "NONE" ? node.textAutoResize : undefined,
      paragraphSpacing: node.paragraphSpacing !== 0 ? node.paragraphSpacing : undefined,
      paragraphIndent: node.paragraphIndent !== 0 ? node.paragraphIndent : undefined,
    }),
  });
};

export const serializeEffects = (effects: any) => {
  if (!effects || !Array.isArray(effects) || isMixed(effects)) return undefined;
  const result = effects.map((e: any) => {
    if (!e || typeof e !== "object") return e;
    try {
      const eff: any = { type: e.type };
      if (e.visible === false) eff.visible = false;
      if (e.color) eff.color = toHexWithAlpha(e.color);
      if (e.offset) eff.offset = { x: pixelRound(e.offset.x), y: pixelRound(e.offset.y) };
      if (e.radius != null) eff.radius = pixelRound(e.radius);
      if (e.spread != null) eff.spread = pixelRound(e.spread);
      if (e.blendMode && e.blendMode !== "NORMAL") eff.blendMode = e.blendMode;
      return eff;
    } catch {
      return null;
    }
  }).filter((e) => e !== null);
  return result.length > 0 ? result : undefined;
};

export const serializeLayoutGrids = (grids: any) => {
  if (!grids || !Array.isArray(grids) || isMixed(grids)) return undefined;
  const result = grids.map((g: any) => {
    if (!g || typeof g !== "object") return g;
    try {
      const grid: any = { pattern: g.pattern };
      if (g.sectionSize != null) grid.sectionSize = g.sectionSize;
      if (g.visible === false) grid.visible = false;
      if (g.color) grid.color = toHexWithAlpha(g.color);
      if (g.alignment) grid.alignment = g.alignment;
      if (g.gutterSize != null) grid.gutterSize = g.gutterSize;
      if (g.offset != null) grid.offset = g.offset;
      if (g.count != null) grid.count = g.count;
      return grid;
    } catch {
      return null;
    }
  }).filter((g) => g !== null);
  return result.length > 0 ? result : undefined;
};

export const serializeConstraints = (c: any) => {
  if (!c || typeof c !== "object" || isMixed(c)) return undefined;
  return { horizontal: c.horizontal, vertical: c.vertical };
};

export const serializeBoundVariables = (bv: any) => {
  if (!bv || typeof bv !== "object" || isMixed(bv)) return undefined;
  const result: Record<string, any> = {};
  for (const key of Object.keys(bv)) {
    try {
      const val = bv[key];
      if (Array.isArray(val)) {
        result[key] = val.map((item) =>
          item && typeof item === "object" && "id" in item ? { id: item.id, type: item.type } : item
        );
      } else if (val && typeof val === "object" && "id" in val) {
        result[key] = { id: val.id, type: val.type };
      } else if (!isMixed(val)) {
        result[key] = val;
      }
    } catch {
      // ignore
    }
  }
  return Object.keys(result).length > 0 ? result : undefined;
};

export const serializeComponentProperties = (cp: any) => {
  if (!cp || typeof cp !== "object" || isMixed(cp)) return undefined;
  const result: Record<string, any> = {};
  for (const key of Object.keys(cp)) {
    try {
      const prop = cp[key];
      if (prop && typeof prop === "object") {
        const p: any = {
          type: prop.type,
          value: isMixed(prop.value) ? "mixed" : prop.value,
        };
        if (prop.preferredValues && Array.isArray(prop.preferredValues)) {
          p.preferredValues = prop.preferredValues.map((pv: any) => ({
            type: pv.type,
            key: pv.key,
          }));
        }
        result[key] = p;
      }
    } catch {
      // ignore
    }
  }
  return Object.keys(result).length > 0 ? result : undefined;
};

export const serializeComponentPropertyDefinitions = (cpd: any) => {
  if (!cpd || typeof cpd !== "object" || isMixed(cpd)) return undefined;
  const result: Record<string, any> = {};
  for (const key of Object.keys(cpd)) {
    try {
      const def = cpd[key];
      if (def && typeof def === "object") {
        const d: any = {
          type: def.type,
          defaultValue: isMixed(def.defaultValue) ? "mixed" : def.defaultValue,
        };
        if (def.variantOptions && Array.isArray(def.variantOptions)) {
          d.variantOptions = def.variantOptions.slice();
        }
        if (def.preferredValues && Array.isArray(def.preferredValues)) {
          d.preferredValues = def.preferredValues.map((pv: any) => ({
            type: pv.type,
            key: pv.key,
          }));
        }
        result[key] = d;
      }
    } catch {
      // ignore
    }
  }
  return Object.keys(result).length > 0 ? result : undefined;
};

const serializeAction = (a: any) => {
  const action: any = { type: a.type };
  if (a.destinationId) action.destinationId = a.destinationId;
  if (a.navigation) action.navigation = a.navigation;
  if (a.transition) action.transition = a.transition;
  if (a.url) action.url = a.url;
  return action;
};

export const serializeReactions = (reactions: any) => {
  if (!reactions || !Array.isArray(reactions) || isMixed(reactions)) return undefined;
  const result = reactions.map((r: any) => {
    if (!r || typeof r !== "object") return null;
    try {
      const item: any = {};
      // `actions` is the current field; `action` is deprecated but still populated
      // on older reactions, so fall back to it rather than dropping the behavior.
      const actions = Array.isArray(r.actions) ? r.actions : r.action ? [r.action] : [];
      const serialized = actions.filter((a: any) => a && typeof a === "object").map(serializeAction);
      if (serialized.length > 0) item.actions = serialized;
      if (r.trigger) {
        item.trigger = { type: r.trigger.type };
        if (r.trigger.delay) item.trigger.delay = r.trigger.delay;
      }
      return item;
    } catch {
      return null;
    }
  }).filter((r) => r !== null);
  return result.length > 0 ? result : undefined;
};

export const serializeVectorPaths = (paths: any) => {
  if (!paths || !Array.isArray(paths) || isMixed(paths)) return undefined;
  const result = paths.map((p: any) => {
    try {
      return {
        data: p.data,
        windingRule: p.windingRule,
      };
    } catch {
      return null;
    }
  }).filter((p) => p !== null);
  return result.length > 0 ? result : undefined;
};

export const serializeExportSettings = (settings: any) => {
  if (!settings || !Array.isArray(settings) || isMixed(settings)) return undefined;
  const result = settings.map((s: any) => {
    try {
      return {
        format: s.format,
        suffix: s.suffix,
        constraint: s.constraint ? { type: s.constraint.type, value: s.constraint.value } : undefined,
      };
    } catch {
      return null;
    }
  }).filter((s) => s !== null);
  return result.length > 0 ? result : undefined;
};

export const serializeNode = async (node: any): Promise<any> => {
  const styles = await serializeStyles(node);
  const base: any = {
    id: node.id,
    name: node.name,
    type: node.type,
    bounds: getBounds(node),
    styles,
  };

  if ("layoutMode" in node && node.layoutMode !== "NONE") {
    base.layoutMode = node.layoutMode;
    base.itemSpacing = node.itemSpacing;
    base.primaryAxisAlignItems = node.primaryAxisAlignItems;
    base.counterAxisAlignItems = node.counterAxisAlignItems;
    if (node.layoutWrap === "WRAP") base.layoutWrap = node.layoutWrap;
  }

  if ("layoutPositioning" in node && node.layoutPositioning !== "AUTO") {
    base.layoutPositioning = node.layoutPositioning;
  }

  if ("layoutScrollBehavior" in node && node.layoutScrollBehavior !== "NONE") {
    base.layoutScrollBehavior = node.layoutScrollBehavior;
  }

  if ("layoutGrids" in node) {
    const lg = serializeLayoutGrids(node.layoutGrids);
    if (lg) base.layoutGrids = lg;
  }

  if ("constraints" in node) {
    const c = serializeConstraints(node.constraints);
    if (c) base.constraints = c;
  }

  if ("boundVariables" in node) {
    const bv = serializeBoundVariables(node.boundVariables);
    if (bv) base.boundVariables = bv;
  }

  if ("isMask" in node && node.isMask) {
    base.isMask = true;
  }

  if ("clipsContent" in node) {
    base.clipsContent = node.clipsContent;
  }

  if ("booleanOperation" in node) {
    base.booleanOperation = node.booleanOperation;
  }

  if ("vectorPaths" in node) {
    const vp = serializeVectorPaths(node.vectorPaths);
    if (vp) base.vectorPaths = vp;
  }

  if ("reactions" in node) {
    const rx = serializeReactions(node.reactions);
    if (rx) base.reactions = rx;
  }

  if ("exportSettings" in node) {
    const es = serializeExportSettings(node.exportSettings);
    if (es) base.exportSettings = es;
  }

  if ("visible" in node && node.visible === false) {
    base.visible = false;
  }

  if ("locked" in node && node.locked === true) {
    base.locked = true;
  }

  if (node.type === "INSTANCE") {
    base.mainComponentId = node.mainComponentId;
    const cp = serializeComponentProperties(node.componentProperties);
    if (cp) base.componentProperties = cp;
  } else if (node.type === "COMPONENT" || node.type === "COMPONENT_SET") {
    // Reading componentPropertyDefinitions throws on a COMPONENT that is a
    // variant inside a COMPONENT_SET — the definitions live on the set.
    try {
      const cpd = serializeComponentPropertyDefinitions(node.componentPropertyDefinitions);
      if (cpd) base.componentPropertyDefinitions = cpd;
    } catch {
      // variant: definitions belong to the parent set
    }
  }

  if (node.type === "TEXT") return serializeText(node, base);
  if ("children" in node) {
    // A child that fails to serialize degrades to an identity stub rather than
    // taking down the whole tree.
    const children = await Promise.all(
      node.children.map(async (child: any) => {
        try {
          return await serializeNode(child);
        } catch {
          return { id: child.id, name: child.name, type: child.type };
        }
      })
    );
    return Object.assign({}, base, { children });
  }
  return base;
};

// Replaces every Symbol (figma.mixed) with the string "mixed" so the payload can
// cross the plugin/UI boundary. Operates on already-serialized plain objects.
// The serializers handle mixed values themselves, so this is a last-resort net:
// subtrees that need no rewriting are returned as-is rather than deep-cloned,
// which keeps a clean document tree from being copied twice on every request.
export const sanitizeSymbols = (obj: any): any => {
  if (typeof obj === "symbol") return "mixed";
  if (obj === null || typeof obj !== "object") return obj;

  if (Array.isArray(obj)) {
    let changed = false;
    const result = obj.map((item) => {
      const sanitized = sanitizeSymbols(item);
      if (sanitized !== item) changed = true;
      return sanitized;
    });
    return changed ? result : obj;
  }

  let changed = false;
  const result: any = {};
  for (const key of Object.keys(obj)) {
    let sanitized: any;
    try {
      const raw = obj[key];
      sanitized = sanitizeSymbols(raw);
      if (sanitized !== raw) changed = true;
    } catch {
      // A throwing getter is as unusable as a symbol — report it the same way.
      sanitized = "mixed";
      changed = true;
    }
    result[key] = sanitized;
  }
  return changed ? result : obj;
};

// deduplicateStyles does a two-pass walk over a serialized node tree.
// First pass: count how many times each fills/strokes array value appears.
// Second pass: replace values that appear more than once with a short ref key.
// Returns the rewritten tree and a globalVars.styles map (or undefined if nothing was deduped).
export const deduplicateStyles = (tree: any): { tree: any; globalVars: Record<string, any> | undefined } => {
  // Pass 1: count occurrences of each serialized fill/stroke value
  const counts = new Map<string, number>();
  const countWalk = (node: any) => {
    if (!node || typeof node !== "object") return;
    const s = node.styles;
    if (s) {
      if (Array.isArray(s.fills)) counts.set(JSON.stringify(s.fills), (counts.get(JSON.stringify(s.fills)) ?? 0) + 1);
      if (Array.isArray(s.strokes)) counts.set(JSON.stringify(s.strokes), (counts.get(JSON.stringify(s.strokes)) ?? 0) + 1);
    }
    if (Array.isArray(node.children)) node.children.forEach(countWalk);
  };
  countWalk(tree);

  // Build ref map for values that appear more than once
  let counter = 0;
  const keyToRef = new Map<string, string>();
  const refs: Record<string, any> = {};
  for (const [key, count] of counts) {
    if (count > 1) {
      const ref = `s${++counter}`;
      keyToRef.set(key, ref);
      refs[ref] = JSON.parse(key);
    }
  }
  if (keyToRef.size === 0) return { tree, globalVars: undefined };

  // Pass 2: replace repeated values with ref keys
  const replaceWalk = (node: any): any => {
    if (!node || typeof node !== "object") return node;
    let result = node;
    const s = node.styles;
    if (s) {
      let newStyles = s;
      if (Array.isArray(s.fills)) {
        const ref = keyToRef.get(JSON.stringify(s.fills));
        if (ref) newStyles = { ...newStyles, fills: ref };
      }
      if (Array.isArray(s.strokes)) {
        const ref = keyToRef.get(JSON.stringify(s.strokes));
        if (ref) newStyles = { ...newStyles, strokes: ref };
      }
      if (newStyles !== s) result = { ...node, styles: newStyles };
    }
    if (Array.isArray(node.children)) {
      const newChildren = node.children.map(replaceWalk);
      result = { ...result, children: newChildren };
    }
    return result;
  };

  return { tree: replaceWalk(tree), globalVars: { styles: refs } };
};

export const serializeVariableValue = (value: any) => {
  if (typeof value !== "object" || value === null) return value;

  if ("type" in value && value.type === "VARIABLE_ALIAS") {
    return { type: "VARIABLE_ALIAS", id: value.id };
  }

  if ("r" in value && "g" in value && "b" in value) {
    return {
      type: "COLOR",
      r: value.r,
      g: value.g,
      b: value.b,
      a: "a" in value ? value.a : 1,
    };
  }

  return value;
};
