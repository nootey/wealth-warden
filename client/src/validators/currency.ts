import Decimal from "decimal.js";
import { helpers } from "@vuelidate/validators";

const isEmpty = (v: unknown) => v === null || v === undefined || v === "";

export const decimalValid = helpers.withMessage(
    "Enter a valid amount",
    (v: unknown) => {
        if (isEmpty(v)) return false;
        try { new Decimal(v as string); return true; } catch { return false; }
    }
);

export const decimalMin = (min: string | number) =>
    helpers.withMessage(`Must be ≥ ${min}`, (v: unknown) => {
        if (isEmpty(v)) return true;
        try { return new Decimal(v as string).gte(min); } catch { return false; }
    });

export const decimalMax = (max: string | number) =>
    helpers.withMessage(`Must be ≤ ${max}`, (v: unknown) => {
        if (isEmpty(v)) return true;
        try { return new Decimal(v as string).lte(max); } catch { return false; }
    });