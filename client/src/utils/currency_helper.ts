import { computed, type Ref } from "vue";
import Decimal from "decimal.js";

const currencyHelper = {
  mightBeBalance(field: string | null): boolean {
    if (!field) return false;
    const balanceFields = ["amount"];

    return (
      balanceFields.includes(field.toLowerCase()) ||
      field.toLowerCase().includes("balance")
    );
  },
  useMoneyField(model: Ref<string | null>, scale = 2) {
    const number = computed<number | null>({
      get() {
        const v = model.value;
        if (v === null || v === "") return null;
        try {
          return new Decimal(v).toNumber(); // for InputNumber
        } catch {
          return null;
        }
      },
      set(val) {
        if (val === null || val === undefined) {
          model.value = null;
          return;
        }
        // normalize to fixed scale (e.g., 2 decimals)
        model.value = new Decimal(val).toFixed(scale);
      },
    });

    return { number };
  },
  // Percent follows calculator rules: "a - b%" subtracts b% of a, "a * b%" is b% of a.
  evaluateExpression(expression: string): Decimal | null {
    const tokens =
      expression
        .replace(/[x×]/gi, "*")
        .replace(/÷/g, "/")
        .replace(/−/g, "-")
        .match(/[\d.,]+|\S/g) ?? [];
    let pos = 0;

    type Operand = { value: Decimal; percent: boolean };
    const operations: Record<string, (a: Decimal, b: Decimal) => Decimal> = {
      "+": (a, b) => a.plus(b),
      "-": (a, b) => a.minus(b),
      "*": (a, b) => a.mul(b),
      "/": (a, b) => a.div(b),
    };

    // Either "." or "," can be the decimal mark: it is the last mark when that
    // mark appears once. Any other mark must group thousands.
    const parseNumber = (raw: string): Decimal | null => {
      const last = raw.replace(/\d/g, "").slice(-1);
      const split =
        last && raw.indexOf(last) === raw.lastIndexOf(last)
          ? raw.lastIndexOf(last)
          : raw.length;
      const whole = raw.slice(0, split);
      const fraction = raw.slice(split + 1);
      if (
        !/\d/.test(raw) ||
        !/^(\d*|\d{1,3}(\.\d{3})+|\d{1,3}(,\d{3})+)$/.test(whole)
      ) {
        return null;
      }
      return new Decimal(
        `${whole.replace(/[.,]/g, "") || "0"}.${fraction || "0"}`,
      );
    };

    const parseFactor = (): Operand | null => {
      const token = tokens[pos++];
      if (token === "-" || token === "+") {
        const inner = parseFactor();
        if (!inner || token === "+") return inner;
        return { value: inner.value.neg(), percent: inner.percent };
      }

      let value: Decimal | null = null;
      if (token === "(") {
        value = parseSum();
        if (tokens[pos++] !== ")") return null;
      } else if (token !== undefined) {
        value = parseNumber(token);
      }
      if (!value) return null;

      if (tokens[pos] !== "%") return { value, percent: false };
      pos++;
      return { value: value.div(100), percent: true };
    };

    const parseProduct = (): Operand | null => {
      let left = parseFactor();
      while (left && (tokens[pos] === "*" || tokens[pos] === "/")) {
        const operate = operations[tokens[pos++]];
        const right = parseFactor();
        if (!right) return null;
        left = { value: operate(left.value, right.value), percent: false };
      }
      return left;
    };

    const parseSum = (): Decimal | null => {
      const first = parseProduct();
      if (!first) return null;
      let total = first.value;
      while (tokens[pos] === "+" || tokens[pos] === "-") {
        const operate = operations[tokens[pos++]];
        const right = parseProduct();
        if (!right) return null;
        const operand = right.percent ? total.mul(right.value) : right.value;
        total = operate(total, operand);
      }
      return total;
    };

    const result = parseSum();
    if (!result || pos !== tokens.length || !result.isFinite()) return null;
    return result;
  },
};

export default currencyHelper;
