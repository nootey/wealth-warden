import Decimal from "decimal.js";
import { withMessage } from "@regle/rules";

const isEmpty = (v: unknown) => v === null || v === undefined || v === "";

export const decimalValid = withMessage((v: unknown) => {
  if (isEmpty(v)) return false;
  try {
    new Decimal(v as string);
    return true;
  } catch {
    return false;
  }
}, "Enter a valid amount");

export const decimalMin = (min: string | number) =>
  withMessage((v: unknown) => {
    if (isEmpty(v)) return true;
    try {
      return new Decimal(v as string).gte(min);
    } catch {
      return false;
    }
  }, `Must be ≥ ${min}`);

export const decimalNonZero = withMessage((v: unknown) => {
  if (isEmpty(v)) return true;
  try {
    return !new Decimal(v as string).isZero();
  } catch {
    return false;
  }
}, "Must not be zero");

export const decimalMax = (max: string | number) =>
  withMessage((v: unknown) => {
    if (isEmpty(v)) return true;
    try {
      return new Decimal(v as string).lte(max);
    } catch {
      return false;
    }
  }, `Must be ≤ ${max}`);
