
interface ValidationObject {
    $error: boolean;
}

const vueHelper = {
    pivotedRecords: (records: any[]) => {
        const pivot: Record<number, any> = {}

        records.forEach(item => {
            const { inflow_category_id, inflow_category_name, month, total_amount } = item

            // Create a new pivot row for the category if it doesn't exist yet
            if (!pivot[inflow_category_id]) {
                pivot[inflow_category_id] = {
                    inflow_category_id,
                    inflow_category_name
                }
            }
            // Use the month number as key
            pivot[inflow_category_id][month] = total_amount
        })

        // Return an array of pivoted rows
        return Object.values(pivot)
    },
    getValidationClass: (state: ValidationObject | null | undefined, errorClass: string) => {
        return {
            [errorClass]: !!state?.$error,
        }
    },
    displayAsCurrency: (amount: number) => {
        if(!amount)
            return null;
        let currency = " €"
        return amount.toFixed(2) + currency;
    },
};

export default vueHelper;