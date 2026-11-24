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

  describe("evaluateExpression", () => {
    const evaluate = (expression: string) =>
      currencyHelper.evaluateExpression(expression)?.toString() ?? null;

    it("applies the basic operators with normal precedence", () => {
      expect(evaluate("120-20")).toBe("100");
      expect(evaluate("2+3*4")).toBe("14");
      expect(evaluate("(2+3)*4")).toBe("20");
      expect(evaluate("10/4")).toBe("2.5");
      expect(evaluate(" 120 - 20 ")).toBe("100");
    });

    it("adds and subtracts percent of the running total", () => {
      expect(evaluate("120-5%")).toBe("114");
      expect(evaluate("120+5%")).toBe("126");
      expect(evaluate("100-5%-5%")).toBe("90.25");
      expect(evaluate("(80+20)-10%")).toBe("90");
    });

    it("multiplies and divides by percent as a fraction", () => {
      expect(evaluate("120*5%")).toBe("6");
      expect(evaluate("120/50%")).toBe("240");
      expect(evaluate("5%")).toBe("0.05");
    });

    it("handles unary signs", () => {
      expect(evaluate("-5+10")).toBe("5");
      expect(evaluate("120*-2")).toBe("-240");
      expect(evaluate("120--5")).toBe("125");
      expect(evaluate("120+-5%")).toBe("114");
    });

    it("reads either decimal mark and drops thousands separators", () => {
      expect(evaluate("120,50-0,5")).toBe("120");
      expect(evaluate("10.5+0,5")).toBe("11");
      expect(evaluate("1.234,56")).toBe("1234.56");
      expect(evaluate("1,234.56")).toBe("1234.56");
      expect(evaluate("1.000.000")).toBe("1000000");
      expect(evaluate("1,234,567")).toBe("1234567");
      expect(evaluate(".5+5.")).toBe("5.5");
    });

    it("accepts the typographic operator signs", () => {
      expect(evaluate("3x4")).toBe("12");
      expect(evaluate("3×4")).toBe("12");
      expect(evaluate("8÷2")).toBe("4");
      expect(evaluate("10−3")).toBe("7");
    });

    it("stays exact where float math drifts", () => {
      expect(evaluate("0.1+0.2")).toBe("0.3");
    });

    it("rejects incomplete or malformed input", () => {
      expect(evaluate("")).toBeNull();
      expect(evaluate("120-")).toBeNull();
      expect(evaluate("(1+2")).toBeNull();
      expect(evaluate("1+2)")).toBeNull();
      expect(evaluate("120%5")).toBeNull();
      expect(evaluate("5%%")).toBeNull();
      expect(evaluate("2(3)")).toBeNull();
      expect(evaluate("1..2")).toBeNull();
      expect(evaluate("1.2.3,4,5")).toBeNull();
      expect(evaluate("12.34.56")).toBeNull();
      expect(evaluate("abc")).toBeNull();
    });

    it("rejects division by zero", () => {
      expect(evaluate("1/0")).toBeNull();
      expect(evaluate("0/0")).toBeNull();
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
