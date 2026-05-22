module.exports = function(floatPrecision = 2) {
    return {
        plugins: [
            {
                name: 'preset-default',
                params: {
                    overrides: {
                        // Disable things that were active: false
                        cleanupIds: false,
                        mergePaths: false, // active: false in legacy
                        
                        // Parameter overrides
                        convertPathData: {
                            floatPrecision: floatPrecision,
                            transformPrecision: floatPrecision,
                            leadingZero: false,
                            makeArcs: false,
                            noSpaceAfterFlags: false,
                            collapseRepeated: false
                        },
                        cleanupNumericValues: {
                            floatPrecision: floatPrecision,
                            leadingZero: false
                        },
                        convertShapeToPath: {
                            convertArcs: true,
                            floatPrecision: floatPrecision
                        },
                        // convertColors: { shorthex: false, shortname: false } // Legacy params
                    }
                }
            },
            // Additional plugins explicitly enabled in legacy
            {
                name: 'removeRasterImages' 
            },
            {
                name: 'convertColors',
                params: {
                    shorthex: false, 
                    shortname: false
                }
            }
        ]
    };
};