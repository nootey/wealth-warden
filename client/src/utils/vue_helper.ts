import Decimal from "decimal.js";

interface ValidationObject {
    $error: boolean;
}

type ChangeSet = { new?: Record<string, any>; old?: Record<string, any> };
type Change = { prop: string; oldVal: any; newVal: any };

type Causer = {
    id: number;
    name: string;
};

const vueHelper = {
    capitalize(value: unknown): string {
        if (value == null) return '';
        const str = String(value);
        return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase();
    },
    normalize(value: string): string {
        if (!value) return '';
        return value.replace(/\s+/g, "_");
    },
    denormalize(value: string): string {
        if (!value) return '';
        return value.replace("_", " ");
    },
    formatString: (value: string) => {
        if (!value) return "";

        let formatted = value.replace(/_/g, " ");
        formatted = formatted.replace(/\bliablity\b/i, "liability");
        formatted = formatted.charAt(0).toUpperCase() + formatted.slice(1).toLowerCase();

        return formatted;
    },
    getValidationClass: (state: ValidationObject | null | undefined, errorClass: string) => {
        return {
            [errorClass]: !!state?.$error,
        }
    },
    displayAsCurrency: (amount: Decimal | number | string | null) => {
        // Hardcode for EU region for now
        if (amount === null || amount === undefined) return null;
        const num = Number(amount);
        if (isNaN(num)) return "Invalid Amount";

        return num.toLocaleString("de-DE", {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2
        }) + "€";
    },
    displayAsPercentage: (value: number | string | null, decimals = 1) => {
        if (value === null || value === undefined) return null;
        const num = Number(value);
        if (isNaN(num)) return "Invalid Percentage";

        const pct = num * 100; 
        return pct.toFixed(decimals) + " %";
    },
    formatChanges(payload: any): Change[] | null {
        if (!payload) return null;

        const obj: ChangeSet = typeof payload === 'string' ? JSON.parse(payload) : payload;

        const newVals = obj.new ?? {};
        const oldVals = obj.old ?? {};
        const keys = new Set([...Object.keys(newVals), ...Object.keys(oldVals)]);

        const out: Change[] = [];
        for (const k of keys) {
            const oldVal = oldVals?.[k] ?? null;
            const newVal = newVals?.[k] ?? null;
            if (oldVal !== newVal) out.push({ prop: k, oldVal, newVal });
        }
        return out.length ? out : null;
    },
    formatValue(item: { newVal: any; oldVal: any }) {
        const hasNew = item.newVal !== undefined && item.newVal !== null && item.newVal !== '';
        const hasOld = item.oldVal !== undefined && item.oldVal !== null && item.oldVal !== '';
        if (hasNew && hasOld && item.newVal !== item.oldVal) {
            return `${item.oldVal} => ${item.newVal}`;
        }
        const v = hasNew ? item.newVal : (hasOld ? item.oldVal : '');
        return String(v ?? '');
    },
    displayCauserFromId(causerId: number | null, availableCausers: Causer[]) {
        if (!causerId || !availableCausers) return '';
        const causer = availableCausers.find(c => c.id === causerId);
        return causer ? causer.name : "Deleted user";
    },
    deletedRowClass<T extends { deleted_at?: string | null }>(data: T) {
        return data?.deleted_at ? 'row-deleted' : '';
    }
};

export default vueHelper;