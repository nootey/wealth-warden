import Decimal from "decimal.js";
import { useSettingsStore } from "../services/stores/settings_store.ts";

interface ValidationObject {
  $error: boolean;
}

type ChangeSet = {
  new?: Record<string, unknown>;
  old?: Record<string, unknown>;
};
type Change = { prop: string; oldVal: unknown; newVal: unknown };

type Causer = {
  id: number;
  name: string;
};

const CURRENCY_LOCALE: Record<string, string> = {
  EUR: "de-DE",
  CHF: "de-CH",
  NOK: "nb-NO",
  SEK: "sv-SE",
  DKK: "da-DK",
  PLN: "pl-PL",
  CZK: "cs-CZ",
  HUF: "hu-HU",
};

const vueHelper = {
  getCurrencyLocale: (currency?: string): string | undefined => {
    if (!currency) return undefined;
    return CURRENCY_LOCALE[currency.toUpperCase()];
  },
  capitalize(value: unknown): string {
    if (value == null) return "";
    const str = String(value);
    return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase();
  },
  normalize(value: string): string {
    if (!value) return "";
    return value.replace(/\s+/g, "_");
  },
  denormalize(value: string): string {
    if (!value) return "";
    return value.replace("_", " ");
  },
  formatString: (value: string) => {
    if (!value) return "";

    let formatted = value.replace(/_/g, " ");
    formatted = formatted.replace(/\bliablity\b/i, "liability");
    formatted =
      formatted.charAt(0).toUpperCase() + formatted.slice(1).toLowerCase();

    return formatted;
  },
  getValidationClass: (
    state: ValidationObject | null | undefined,
    errorClass: string,
  ) => {
    return {
      [errorClass]: !!state?.$error,
    };
  },
  displayAsCurrency: (
    amount: Decimal | number | string | null,
    currency?: string,
  ) => {
    if (amount === null || amount === undefined) return null;
    const num = Number(amount);
    if (isNaN(num)) return "Invalid Amount";

    const cur = (
      currency ||
      useSettingsStore().defaultCurrency ||
      "EUR"
    ).toUpperCase();

    try {
      return new Intl.NumberFormat(CURRENCY_LOCALE[cur], {
        style: "currency",
        currency: cur,
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      }).format(num);
    } catch {
      return num.toFixed(2) + " " + cur;
    }
  },
  displayAsPercentage: (value: number | string | null, decimals = 1) => {
    if (value === null || value === undefined) return null;
    const num = Number(value);
    if (isNaN(num)) return "Invalid Percentage";

    const pct = num * 100;
    return pct.toFixed(decimals) + " %";
  },
  displayAssetPrice: (
    amount: Decimal | number | string | null,
    investmentType?: string,
    currency?: string,
    forceDoubleDigitDecimal?: boolean,
  ) => {
    if (amount === null || amount === undefined) return null;
    const num = Number(amount);
    if (isNaN(num)) return "Invalid Amount";

    const decimals = forceDoubleDigitDecimal
      ? 2
      : investmentType === "crypto"
        ? 4
        : 2;
    const cur = (
      currency ||
      useSettingsStore().defaultCurrency ||
      "EUR"
    ).toUpperCase();

    try {
      return new Intl.NumberFormat(CURRENCY_LOCALE[cur], {
        style: "currency",
        currency: cur,
        minimumFractionDigits: decimals,
        maximumFractionDigits: decimals,
      }).format(num);
    } catch {
      return num.toFixed(decimals) + " " + cur;
    }
  },
  formatChanges(payload: unknown): Change[] | null {
    if (!payload) return null;

    const obj: ChangeSet =
      typeof payload === "string" ? JSON.parse(payload) : payload;

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
  formatValue(item: { newVal: unknown; oldVal: unknown }) {
    const hasNew =
      item.newVal !== undefined && item.newVal !== null && item.newVal !== "";
    const hasOld =
      item.oldVal !== undefined && item.oldVal !== null && item.oldVal !== "";
    if (hasNew && hasOld && item.newVal !== item.oldVal) {
      return `${item.oldVal} => ${item.newVal}`;
    }
    const v = hasNew ? item.newVal : hasOld ? item.oldVal : "";
    return String(v ?? "");
  },
  displayCauserFromId(causerId: number | null, availableCausers: Causer[]) {
    if (!causerId || !availableCausers) return "";
    const causer = availableCausers.find((c) => c.id === causerId);
    return causer ? causer.name : "Deleted user";
  },
  deletedRowClass<T extends { deleted_at?: string | null }>(data: T) {
    return data?.deleted_at ? "row-deleted" : "";
  },
  isActiveRowClass<T extends { is_active?: string | null }>(data: T) {
    return !data?.is_active ? "row-deleted" : "";
  },
};

export default vueHelper;
