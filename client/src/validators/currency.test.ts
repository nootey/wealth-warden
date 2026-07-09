import { describe, it, expect } from "vitest";
import {
  decimalValid,
  decimalMin,
  decimalMax,
  decimalNonZero,
} from "./currency.ts";

type Rule = {
  $validator: (value: unknown, siblings: unknown, vm: unknown) => boolean;
};

const check = (rule: unknown, value: unknown): boolean =>
  (rule as Rule).$validator(value, {}, {});

const EMPTY = [null, undefined, ""];

describe("currency validators", () => {
  describe("decimalValid", () => {
    it("rejects an empty value", () => {
      for (const value of EMPTY) expect(check(decimalValid, value)).toBe(false);
    });

    it("rejects a non-numeric string", () => {
      expect(check(decimalValid, "abc")).toBe(false);
    });

    it("accepts decimal strings and numbers", () => {
      for (const value of ["12.50", "-4.2", "1e3", 0, 12.5])
        expect(check(decimalValid, value)).toBe(true);
    });
  });

  describe("decimalMin", () => {
    it("defers an empty value to the required rule", () => {
      for (const value of EMPTY) expect(check(decimalMin(5), value)).toBe(true);
    });

    it("accepts a value at or above the bound", () => {
      expect(check(decimalMin(5), 5)).toBe(true);
      expect(check(decimalMin(5), "5.00")).toBe(true);
      expect(check(decimalMin(5), 5.01)).toBe(true);
    });

    it("rejects a value below the bound", () => {
      expect(check(decimalMin(5), 4.99)).toBe(false);
    });

    it("rejects an unparseable value", () => {
      expect(check(decimalMin(5), "abc")).toBe(false);
    });
  });

  describe("decimalMax", () => {
    it("defers an empty value to the required rule", () => {
      for (const value of EMPTY) expect(check(decimalMax(5), value)).toBe(true);
    });

    it("accepts a value at or below the bound", () => {
      expect(check(decimalMax(5), 5)).toBe(true);
      expect(check(decimalMax(5), 4.99)).toBe(true);
    });

    it("rejects a value above the bound", () => {
      expect(check(decimalMax(5), 5.01)).toBe(false);
    });
  });

  describe("decimalNonZero", () => {
    it("defers an empty value to the required rule", () => {
      for (const value of EMPTY)
        expect(check(decimalNonZero, value)).toBe(true);
    });

    it("rejects every spelling of zero", () => {
      for (const value of [0, "0", "0.00", "-0"])
        expect(check(decimalNonZero, value)).toBe(false);
    });

    it("accepts a non-zero value", () => {
      expect(check(decimalNonZero, 0.01)).toBe(true);
      expect(check(decimalNonZero, -0.01)).toBe(true);
    });
  });
});
