import { describe, it, expect } from "vitest";
import { ref } from "vue";
import currencyHelper from "./currency_helper.ts";
import type { Ref } from "vue";

const moneyField = (initial: string | null = null, scale = 2) => {
  const model: Ref<string | null> = ref(initial);
  const { number } = currencyHelper.useMoneyField(model, scale);
  return { model, number };
};

describe("currencyHelper", () => {
  describe("useMoneyField reading into the input", () => {
    it("reads an empty model as no value", () => {
      expect(moneyField(null).number.value).toBeNull();
      expect(moneyField("").number.value).toBeNull();
    });

    it("reads a decimal string as a number", () => {
      expect(moneyField("12.50").number.value).toBe(12.5);
    });

    it("reads exponent notation", () => {
      expect(moneyField("1e3").number.value).toBe(1000);
    });

    it("reads an unparseable model as no value rather than NaN", () => {
      expect(moneyField("abc").number.value).toBeNull();
    });
  });

  describe("useMoneyField writing back from the input", () => {
    it("normalises to the fixed scale", () => {
      const { model, number } = moneyField();
      number.value = 12.5;
      expect(model.value).toBe("12.50");
    });

    it("clears the model when the input is cleared", () => {
      const { model, number } = moneyField("12.50");
      number.value = null;
      expect(model.value).toBeNull();
    });

    it("rounds halves away from zero, unlike float toFixed", () => {
      const { model, number } = moneyField();

      number.value = 1.005;
      expect(model.value).toBe("1.01");
      expect((1.005).toFixed(2)).toBe("1.00");

      number.value = 2.675;
      expect(model.value).toBe("2.68");
      expect((2.675).toFixed(2)).toBe("2.67");
    });

    it("rounds negative halves away from zero", () => {
      const { model, number } = moneyField();
      number.value = -0.005;
      expect(model.value).toBe("-0.01");
    });

    it("honours a custom scale", () => {
      const whole = moneyField(null, 0);
      whole.number.value = 12.5;
      expect(whole.model.value).toBe("13");

      const precise = moneyField(null, 4);
      precise.number.value = 1.23456;
      expect(precise.model.value).toBe("1.2346");
    });

    it("round-trips a rounded value back through the getter", () => {
      const { number } = moneyField();
      number.value = 1.005;
      expect(number.value).toBe(1.01);
    });
  });

  describe("mightBeBalance", () => {
    it("matches amount and any balance-suffixed field", () => {
      expect(currencyHelper.mightBeBalance("amount")).toBe(true);
      expect(currencyHelper.mightBeBalance("Start_Balance")).toBe(true);
      expect(currencyHelper.mightBeBalance("note")).toBe(false);
      expect(currencyHelper.mightBeBalance(null)).toBe(false);
    });
  });
});
