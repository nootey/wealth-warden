import type {GroupedItem} from "../models/shared.ts";

interface ValidationObject {
    $error: boolean;
}

const vueHelper = {
    pivotedRecords: (records: any[], getCategoryType?: (item: any) => string | null) => {
        if (!records || records.length === 0) {
            return [];
        }

        const pivot: Record<string, any> = {};

        records.forEach(item => {
            const { category_id, category_name, month, total_amount } = item;
            const category_type = getCategoryType ? getCategoryType(item) : (item.category_type || "Unknown");

            if (category_type.toLowerCase() === "total") {
                // For total rows, use a fixed key so that they are merged into one row
                const uniqueKey = "total";
                if (!pivot[uniqueKey]) {
                    pivot[uniqueKey] = { category_type: "total" };
                }
                // Sum amounts if there are multiple total records for the same month
                pivot[uniqueKey][month] = (pivot[uniqueKey][month] || 0) + (total_amount || 0);
            } else {
                // For other rows, group by category_type and category_id
                const uniqueKey = `${category_type}_${category_id}`;
                if (!pivot[uniqueKey]) {
                    pivot[uniqueKey] = {
                        category_id,
                        category_name,
                        category_type,
                    };
                }
                // Simply assign (or overwrite) the amount for this month
                pivot[uniqueKey][month] = total_amount || 0;
            }
        });

        const pivotArray = Object.values(pivot);

        // Separate total rows and non-total rows.
        // Total row(s) always come first, while non-total rows remain unsorted.
        const totalRows = pivotArray.filter(item => item.category_type.toLowerCase() === "total");
        const otherRows = pivotArray.filter(item => item.category_type.toLowerCase() !== "total");

        return [...totalRows, ...otherRows];
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
        targetRef: { value: { category: string; total: number; average: number; spending_limit: number | null, category_type: string | null }[] },
        getCategoryId: (item: T) => number,
        getCategoryName: (item: T) => string,
        getTotalAmount: (item: T) => number,
        getMonth: (item: T) => number,
        getSpendingLimit?: (item: T) => number | null,
        getCategoryType?: (item: T) => string | null,
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
        const category_type = getCategoryType ? getCategoryType(curr) : null;

        if (!acc[category_id]) {
            acc[category_id] = {
                categoryName: category_name,
                total: 0,
                months: new Set<number>(),
                spendingLimit: spending_limit,
                categoryType: category_type,
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
            category_type: category.categoryType ?? null
        };
    });
    }

};

export default vueHelper;