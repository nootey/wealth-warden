import { defineAsyncComponent } from 'vue';
import type { Component } from 'vue';
import type { FilterObj } from '../models/shared_models';
import Decimal from 'decimal.js';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
dayjs.extend(utc);

export type Column = {
    field: string;
    header: string;
    type?: 'text'|'number'|'date'|'enum';
    options?: any[];
    optionLabel?: string;
    optionValue?: string;
};

export type PanelDef<M=any> = {
    component: Component;
    makeModel: () => M;
    toFilters: (model: M, ctx: { field: string; source: string }) => FilterObj[];
    passProps?: Record<string, any>;
};

// Reusable panel factories
const DatePanel   = defineAsyncComponent(() => import('../_vue/components/filters/panels/DatePanel.vue'));
const MultiSelectPanel   = defineAsyncComponent(() => import('../_vue/components/filters/panels/MultiSelectPanel.vue'));
const RangePanel  = defineAsyncComponent(() => import('../_vue/components/filters/panels/RangePanel.vue'));
const TextPanel   = defineAsyncComponent(() => import('../_vue/components/filters/panels/TextPanel.vue')); // default

export const defs = {
    date(): PanelDef<{ date: Date|null; from: Date|null; to: Date|null }> {
        return {
            component: DatePanel,
            makeModel: () => ({ date: null, from: null, to: null }),
            toFilters: ({ date, from, to }, { field, source }) => {
                const out: FilterObj[] = [];
                const ymd = (d: Date) => dayjs(d).format('YYYY-MM-DD');
                const startISO = (d: Date) => dayjs(d).startOf('day').utc().toISOString();
                const endISO   = (d: Date) => dayjs(d).endOf('day').utc().toISOString();

                if (date) return [{ source, field, operator: '=', value: ymd(date) }];

                if (from) out.push({ source, field, operator: '>=', value: startISO(from) });
                if (to)   out.push({ source, field, operator: '<=', value: endISO(to) });
                return out;
            }
        };
    },

    numberRange(): PanelDef<{min:number|null; max:number|null; single?:number|null; singleOp?: '='|'>='|'<='}> {
        return {
            component: RangePanel,
            makeModel: () => ({ min: null, max: null, single: null, singleOp: '=' }),
            toFilters: ({ single, singleOp, min, max }, { field, source }) => {
                const out: FilterObj[] = [];

                if (single !== null && single !== undefined) {
                    out.push({
                        source,
                        field,
                        operator: singleOp ?? '=',
                        value: toMoneyStr(single)
                    });
                    return out;
                }

                if (min !== null && min !== undefined) {
                    out.push({ source, field, operator: '>=', value: toMoneyStr(min) });
                }
                if (max !== null && max !== undefined) {
                    out.push({ source, field, operator: '<=', value: toMoneyStr(max) });
                }
                return out;
            }
        };
    },

    enumMulti(col?: Column): PanelDef<string[]|number[]|null> {
        const options = col?.options ?? [];
        const optionLabel = col?.optionLabel;
        const inferredValueKey = inferOptionValueKey(options, optionLabel, col?.optionValue);

        // map value -> label for display
        const valueKey = inferredValueKey ?? optionLabel ?? 'id';
        const labelKey = optionLabel ?? 'label';
        const valueToLabel = new Map(
            (options as any[]).map(o => [o?.[valueKey], String(o?.[labelKey])])
        );

        return {
            component: MultiSelectPanel,
            makeModel: () => null,
            toFilters: (v, { field, source }) =>
                Array.isArray(v) && v.length
                    ? v.map(val => ({ source, field, operator: '=', value: val, display: valueToLabel.get(val) }))
                    : [],
            passProps: {
                options,
                optionLabel,
                optionValue: valueKey
            }
        };
    },

    textLike(): PanelDef<string|null> {
        return {
            component: TextPanel,
            makeModel: () => null,
            toFilters: (v, { field, source }) =>
                !v ? [] : [{ source, field, operator: 'like', value: String(v).trim().replace(/\s+/g,' ') }]
        };
    }
};

// Rules
type Rule = {
    test: (col: Column) => boolean;
    use: (col: Column) => PanelDef<any>;
    icon: string;
};
const byField = (re: RegExp) => (c: Column) => re.test(c.field);
const byType  = (t: Column['type']) => (c: Column) => c.type === t;
const always  = () => true;

export const rules: Rule[] = [
    { test: byType('date'),               use: () => defs.date(),             icon: 'pi pi-calendar' },
    { test: byField(/^amount$|^balance$/),use: () => defs.numberRange(),      icon: 'pi pi-wallet'   },
    { test: byType('number'),             use: () => defs.numberRange(),      icon: 'pi pi-hashtag'  },
    { test: byType('enum'),               use: (c) => defs.enumMulti(c),      icon: 'pi pi-list'     },
    { test: always,                       use: () => defs.textLike(),         icon: 'pi pi-search'   },
];

export function resolveFor(col: Column) {
    const rule = rules.find(r => r.test(col))!;
    return { def: rule.use(col), icon: rule.icon };
}

function inferOptionValueKey(
    options: any[] | undefined,
    optionLabel?: string,
    explicit?: string
): string | null {
    if (!options || options.length === 0) return null;

    if (explicit && options.every(o => explicit in o)) return explicit;

    if (options.every(o => typeof o === 'object' && o && 'id' in o)) return 'id';

    if (optionLabel && options.every(o => typeof o === 'object' && o && optionLabel in o)) {
        return optionLabel;
    }

    const candidateKeys = Object.keys(options[0] ?? {});
    for (const k of candidateKeys) {
        if (options.every(o => o && k in o && (['string','number','boolean'].includes(typeof o[k])))) {
            return k;
        }
    }

    return null;
}

function toMoneyStr(v: unknown, scale = 4): string {
    if (v === null || v === undefined || v === '') return '';
    return new Decimal(v as any).toFixed(scale);
}