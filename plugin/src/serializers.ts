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

export const serializePaints = (paints: any) => {
  if (isMixed(paints)) return "mixed";

  if (!paints || !Array.isArray(paints)) return undefined;

  const result = paints.map((paint: any) => {
    if (!paint || typeof paint !== "object") return paint;

    if (paint.type === "SOLID" && "color" in paint) {
      const hex = toHex(paint.color);
      const opacity = paint.opacity != null ? paint.opacity : 1;
      if (opacity === 1) return hex;
      return (
        hex +
        Math.round(opacity * 255)
          .toString(16)
          .padStart(2, "0")
      );
    }

    if (paint.type && typeof paint.type === "string" && paint.type.startsWith("GRADIENT_")) {
      const gradientObj: any = {
        type: paint.type,
      };
      if (Array.isArray(paint.gradientStops)) {
        gradientObj.gradientStops = paint.gradientStops.map((stop: any) => {
          const stopColor = stop.color;
          let hex = stopColor ? toHex(stopColor) : "#000000";
          const alpha = stopColor && stopColor.a != null ? stopColor.a : 1;
          if (alpha < 1) {
            hex += Math.round(alpha * 255)
              .toString(16)
              .padStart(2, "0");
          }
          return {
            position: pixelRound(stop.position ?? 0),
            color: hex,
          };
        });
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
    
    if ("strokeWeight" in node && node.strokeWeight !== 0) styles.strokeWeight = node.strokeWeight;
    if ("strokeAlign" in node && node.strokeAlign !== "INSIDE") styles.strokeAlign = node.strokeAlign;
    if ("strokeCap" in node && node.strokeCap !== "NONE") styles.strokeCap = node.strokeCap;
    if ("strokeJoin" in node && node.strokeJoin !== "MITER") styles.strokeJoin = node.strokeJoin;
    if ("dashPattern" in node && node.dashPattern.length > 0) styles.dashPattern = node.dashPattern;
  }

  if ("effects" in node && node.effects.length > 0) {
    if (node.effectStyleId && typeof node.effectStyleId === "string") {
      const style = await figma.getStyleByIdAsync(node.effectStyleId);
      if (style) styles.effectStyle = style.name;
    }
    styles.effects = node.effects;
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

  if ("layoutGrids" in node && node.layoutGrids.length > 0) {
    base.layoutGrids = node.layoutGrids;
  }
  
  if ("constraints" in node) {
    base.constraints = node.constraints;
  }

  if ("boundVariables" in node && node.boundVariables && Object.keys(node.boundVariables).length > 0) {
    base.boundVariables = node.boundVariables;
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

  if ("vectorPaths" in node && node.vectorPaths.length > 0) {
    base.vectorPaths = node.vectorPaths;
  }

  if ("reactions" in node && node.reactions.length > 0) {
    base.reactions = node.reactions;
  }

  if ("exportSettings" in node && node.exportSettings.length > 0) {
    base.exportSettings = node.exportSettings;
  }

  if ("visible" in node && node.visible === false) {
    base.visible = false;
  }

  if ("locked" in node && node.locked === true) {
    base.locked = true;
  }

  if (node.type === "INSTANCE") {
    base.mainComponentId = node.mainComponentId;
    base.componentProperties = node.componentProperties;
  } else if (node.type === "COMPONENT" || node.type === "COMPONENT_SET") {
    base.componentPropertyDefinitions = node.componentPropertyDefinitions;
  }

  if (node.type === "TEXT") return serializeText(node, base);
  if ("children" in node) {
    return Object.assign({}, base, {
      children: await Promise.all(node.children.map((child: any) => serializeNode(child))),
    });
  }
  return base;
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
