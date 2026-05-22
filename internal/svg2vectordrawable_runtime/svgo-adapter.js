const { optimize } = require('svgo');

class JSAPI {
    constructor(data, parentNode) {
        this.parentNode = parentNode || null;
        this.type = data.type || 'element';
        this.name = data.name || '';
        this.children = [];
        this.attrs = {}; // Legacy API expects 'attrs' object

        // Map XAST attributes (simple key-value) to Legacy JSAPI attributes ({name, value, local, prefix})
        if (data.attributes) {
            for (const [key, value] of Object.entries(data.attributes)) {
                this._addAttrInternal(key, value);
            }
        } else if (data.attrs) {
            // Already in legacy format or similar
             this.attrs = data.attrs;
        }

        // Handle children
        if (data.children && Array.isArray(data.children)) {
            this.children = data.children.map(c => new JSAPI(c, this));
        }
    }

    _addAttrInternal(name, value) {
        const parts = name.split(':');
        let local = parts[0];
        let prefix = '';
        if (parts.length > 1) {
            prefix = parts[0];
            local = parts[1];
        }
        this.attrs[name] = {
            name: name,
            value: value,
            local: local,
            prefix: prefix
        };
    }

    // Legacy JSAPI text node handling? 
    // SVGO v2 had text nodes? XAST has { type: 'text', value: '...' }
    // Android VectorDrawable doesn't support text, so maybe it's ignored or handled?
    // The converter doesn't seem to handle text nodes explicitly, it iterates children.

    hasAttr(name, value) {
        const attr = this.attrs[name];
        if (!attr) return false;
        if (value !== undefined) return attr.value === value;
        return true;
    }

    attr(name) {
        // Return whole attribute object
        return this.attrs[name];
    }

    addAttr(attrObj) {
        // attrObj: { name, value, prefix, local }
        // Ensure it's in our internal format
        this.attrs[attrObj.name] = attrObj;
    }

    removeAttr(name) {
        delete this.attrs[name];
    }

    renameElem(newName) {
        this.name = newName;
    }

    eachAttr(callback, context) {
        for (const key in this.attrs) {
            callback.call(context || this, this.attrs[key]);
        }
    }

    isEmpty() {
        return !this.children || this.children.length === 0;
    }
    
    // Helper to find specific children (used in converter?)
    // converter uses querySelectorAll on `data` (which is Root node)
    // We need to implement querySelectorAll if it was part of JSAPI or SVGO API.
    // Looking at converter: 
    // `data.querySelectorAll('use')`
    // `root.querySelector('svg')`
    // Wait, JSAPI v2 had querySelector/All?
    // If so, I MUST implement them.
    
    querySelector(selector) {
        // Simple selector support: tag name only, or id (#id), or attr ([fill="..."])
        // The converter uses complex selectors: 'path[fill="url(#...)"]'
        const results = this.querySelectorAll(selector);
        return results.length > 0 ? results[0] : null;
    }

    querySelectorAll(selector) {
        const results = [];
        this._traverse(node => {
            if (this._matches(node, selector)) {
                results.push(node);
            }
        });
        return results;
    }
    
    _traverse(callback) {
        callback(this);
        if (this.children) {
            this.children.forEach(c => c._traverse(callback));
        }
    }
    
    _matches(node, selector) {
        if (node.type !== 'element') return false;
        
        // Split selector by comma
        if (selector.includes(',')) {
            const parts = selector.split(',').map(s => s.trim());
            return parts.some(p => this._matches(node, p));
        }

        // Tag name
        if (/^[a-zA-Z0-9\-_:]+$/.test(selector)) {
            return node.name === selector;
        }
        
        // Attribute exact match: path[fill="..."]
        const attrMatch = selector.match(/^([a-zA-Z0-9\-_:]+)?\[([a-zA-Z0-9\-_:]+)="([^"]+)"\]$/);
        if (attrMatch) {
            const tagName = attrMatch[1];
            const attrName = attrMatch[2];
            const attrVal = attrMatch[3];
            if (tagName && node.name !== tagName) return false;
            return node.hasAttr(attrName, attrVal);
        }
        
        // Attribute existence: path[fill] (not used much)
        
        return false; 
    }

    spliceContent(index, count, newItems) {
        const items = Array.isArray(newItems) ? newItems : [newItems];
        // Filter empty
        const validItems = items.filter(i => i && (i instanceof JSAPI || Array.isArray(i) && i.length === 0 ? false : true));
        
        // Flatten if newItems is [ [Node] ] or empty []
        // Code sometimes calls spliceContent(..., []) to remove.
        const flatItems = items.flat();

        flatItems.forEach(item => {
             if (item instanceof JSAPI) {
                 item.parentNode = this;
             }
        });
        
        this.children.splice(index, count, ...flatItems);
    }
}

function parseSvg(svgString) {
    let xastRoot = null;
    // We disable all plugins to get "raw-ish" AST, 
    // but strict parsing is handled by svgo's parser.
    optimize(svgString, {
        plugins: [
            {
                name: 'fetch-ast',
                fn: (root) => {
                    xastRoot = root;
                    return {}; 
                }
            }
        ]
    });

    if (!xastRoot) {
        // Fallback for extremely simple or empty SVG?
        // Or if SVGO failed silently.
        // Try creating a dummy root?
        throw new Error('SVGO failed to parse SVG');
    }
    
    return new JSAPI(xastRoot);
}

module.exports = {
    JSAPI,
    parseSvg
};
