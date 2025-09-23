import styleHelper from "../../utils/style_helper.ts";

const KNOWN: Record<string, string> = {
    cash:            '#5249D1',
    investment:      '#4563FF',
    crypto:          '#78BEFF',
    property:        '#55EDD9',
    vehicle:         '#30BF70',
    other_asset:     '#71D17B',
    credit_card:     '#ef4444',
    loan:            '#f97316',
    other_liability: '#eab308',
};

const FALLBACK = [
    '#7C3AED',
    '#6366F1',
    '#06B6D4',
    '#14B8A6',
    '#84CC16',
    '#F59E0B',
    '#DC2626',
    '#EC4899',
    '#F43F5E',
];

function hashToIndex(key: string, mod: number) {
    let h = 2166136261;
    for (let i = 0; i < key.length; i++) {
        h ^= key.charCodeAt(i);
        h = (h * 16777619) >>> 0;
    }
    return h % mod;
}

function baseColorFor(type?: string) {
    const t = (type || 'other_asset').toLowerCase();
    if (KNOWN[t]) return KNOWN[t];
    return FALLBACK[hashToIndex(t, FALLBACK.length)];
}

export type AccountTypeColor = {
    bg: string
    fg: string;
    border: string;
};

export function colorForAccountType(type?: string): AccountTypeColor {
    const bg = baseColorFor(type);
    const fg = styleHelper.readableTextOn(bg);
    const border = styleHelper.shadeHsl(bg, -0.12);
    return { bg, fg, border };
}

// Optional: expose the raw mapping if you ever need the same color list
export const ACCOUNT_COLOR_KNOWN = KNOWN;