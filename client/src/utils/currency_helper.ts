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
    }
};

export default currencyHelper;