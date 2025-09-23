const styleHelper = {
    hexToRgb(hex: string) {
        const h = hex.replace('#', '');
        const bigint = parseInt(h.length === 3 ? h.split('').map(c => c + c).join('') : h, 16);
        return { r: (bigint >> 16) & 255, g: (bigint >> 8) & 255, b: bigint & 255 };
    },
    rgbToHex(r: number, g: number, b: number) {
        return '#' + [r, g, b].map(v => v.toString(16).padStart(2, '0')).join('');
    },
    rgbToHsl(r: number, g: number, b: number) {
        r /= 255; g /= 255; b /= 255;
        const max = Math.max(r,g,b), min = Math.min(r,g,b);
        let h = 0, s = 0, l = (max + min) / 2;
        const d = max - min;
        if (d !== 0) {
            s = l > .5 ? d / (2 - max - min) : d / (max + min);
            switch (max) {
                case r: h = (g - b) / d + (g < b ? 6 : 0); break;
                case g: h = (b - r) / d + 2; break;
                case b: h = (r - g) / d + 4; break;
            }
            h /= 6;
        }
        return { h, s, l };
    },
    hslToRgb(h: number, s: number, l: number) {
        const hue2rgb = (p: number, q: number, t: number) => {
            if (t < 0) t += 1; if (t > 1) t -= 1;
            if (t < 1/6) return p + (q - p) * 6 * t;
            if (t < 1/2) return q;
            if (t < 2/3) return p + (q - p) * (2/3 - t) * 6;
            return p;
        };
        let r: number, g: number, b: number;
        if (s === 0) { r = g = b = l; }
        else {
            const q = l < .5 ? l * (1 + s) : l + s - l * s;
            const p = 2 * l - q;
            r = hue2rgb(p, q, h + 1/3);
            g = hue2rgb(p, q, h);
            b = hue2rgb(p, q, h - 1/3);
        }
        return { r: Math.round(r * 255), g: Math.round(g * 255), b: Math.round(b * 255) };
    },
    shadeHsl(hex: string, amount: number) {
        const { r, g, b } = this.hexToRgb(hex);
        const { h, s, l } = this.rgbToHsl(r, g, b);
        const nl = this.clamp(l + amount, 0, 1);
        const { r: rr, g: gg, b: bb } = this.hslToRgb(h, s, nl);
        return this.rgbToHex(rr, gg, bb);
    },
    clamp(n: number, min: number, max: number) { return Math.min(max, Math.max(min, n)); },
    luminance(hex: string) {
        const { r, g, b } = this.hexToRgb(hex);
        const srgb = [r, g, b].map(v => {
            const c = v / 255;
            return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
        });
        return 0.2126 * srgb[0] + 0.7152 * srgb[1] + 0.0722 * srgb[2];
    },
    readableTextOn(bg: string, light = '#ffffff', dark = '#111111') {
        return this.luminance(bg) > 0.5 ? dark : light;
    },
    shade(hex: string, amount: number) {
        const { r, g, b } = this.hexToRgb(hex);
        const mix = (c: number) => {
            const t = amount < 0 ? 0 : 255;
            return Math.round(c + (t - c) * Math.abs(amount));
        };
        const rr = mix(r), gg = mix(g), bb = mix(b);
        return '#' + [rr, gg, bb].map(v => v.toString(16).padStart(2, '0')).join('');
    }
};

export default styleHelper;
