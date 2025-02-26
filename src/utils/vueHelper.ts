import type {GroupedItem} from "../models/shared.ts";

interface ValidationObject {
    $error: boolean;
}

const vueHelper = {
    pivotedRecords: (records: any[]) => {
        if (!records || records.length === 0) {
            return;
        }

        const pivot: Record<number, any> = {}

        records.forEach(item => {
            const { category_id, category_name, month, total_amount } = item

            // Create a new pivot row for the category if it doesn't exist yet
            if (!pivot[category_id]) {
                pivot[category_id] = {
                    category_id,
                    category_name
                }
            }
            // Use the month number as key
            pivot[category_id][month] = total_amount
        })

        // Return an array of pivoted rows
        return Object.values(pivot)
    },
    getValidationClass: (state: ValidationObject | null | undefined, errorClass: string) => {
        return {
            [errorClass]: !!state?.$error,
        }
    },
    displayAsCurrency: (amount: number|string|null) => {
        if (amount === null || amount === undefined) return null;
        let num = Number(amount);  // Ensure it's a number
        if (isNaN(num)) return "Invalid Amount"; // Handle invalid cases
        return num.toFixed(2) + " €";
    },
    initSort() {
        return {
            order: -1,
            field: 'created_at'
        };
    },
    calculateGroupedStatistics<T>(
        groupedItems: T[],
        targetRef: { value: { category: string; total: number; average: number; spending_limit: number | null }[] },
        getCategoryId: (item: T) => number,
        getCategoryName: (item: T) => string,
        getTotalAmount: (item: T) => number,
        getMonth: (item: T) => number,
        getSpendingLimit?: (item: T) => number
    ): void {
        if (!groupedItems || groupedItems.length === 0) {
        return;
    }

    const groupedData = groupedItems.reduce<Record<number, GroupedItem>>((acc, curr) => {
        const category_id = getCategoryId(curr);
        const category_name = getCategoryName(curr);
        const total_amount = getTotalAmount(curr);
        const month = getMonth(curr);
        const spending_limit = getSpendingLimit ? getSpendingLimit(curr) : null;

        if (!acc[category_id]) {
            acc[category_id] = {
                categoryName: category_name,
                total: 0,
                months: new Set<number>(),
                spendingLimit: spending_limit,
            };
        }

        acc[category_id].total += total_amount;
        acc[category_id].months.add(month);

        return acc;
    }, {});

    targetRef.value = Object.values(groupedData).map((category: GroupedItem) => {
        const monthCount = category.months.size;
        return {
            category: category.categoryName,
            total: category.total,
            average: category.total / monthCount,
            spending_limit: category.spendingLimit ?? null,
        };
    });
    }

};

export default vueHelper;