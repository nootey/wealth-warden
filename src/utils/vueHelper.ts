interface ValidationObject {
    $error: boolean;
}

const vueHelper = {
    formatString: (value: string) => {
        if (!value) return "";

        let formatted = value.replace(/_/g, " ");
        formatted = formatted.replace(/\bliablity\b/i, "liability");
        formatted = formatted.charAt(0).toUpperCase() + formatted.slice(1).toLowerCase();

        return formatted;
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
    displayAsPercentage: (value: number | string | null) => {
        if (value === null || value === undefined) return null;

        const num = Number(value);
        if (isNaN(num)) return "Invalid Percentage";

        return num.toFixed(1) + " %";
    },
    initSort() {
        return {
            order: -1,
            field: 'created_at'
        };
    },
    toggleSort(sortValue: number) {
        switch (sortValue) {
            case 1:
                return -1;
            case -1:
                return 1;
        }
    },
    sortIcon(sort: any, field: string) {
        if (sort.order === -1 && sort.field === field) {
            return 'pi-sort-down';
        }
        if (sort.order === 1 && sort.field === field) {
            return 'pi-sort-up';
        }
        return 'pi-sort';
    },
    formatSuccessToast(title: string, msg: string) {
        let message = {
            'data': {
                'messages': {
                    'success': [] as string[]
                },
                'title': {}
            }
        }
        message['data']['title'] = title;
        message['data']['messages']['success'].push(msg);
        return message;
    },
    formatInfoToast(title: string, msg: string) {
        let message = {
            'data': {
                'messages': {
                    'info': [] as string[]
                },
                'title': {}
            }
        }
        message['data']['title'] = title;
        message['data']['messages']['info'].push(msg);
        return message;
    },
    formatErrorToast(title: string, msg: string){
        let message = {
            'response': {
                'data': {
                    'messages': {
                        'error': [] as string[]
                    },
                    'title': {}
                }
            }
        }
        message['response']['data']['title'] = title;
        message['response']['data']['messages']['error'].push(msg);
        return message;
    },
    formatChanges(payload: any) {
        if (payload === "[]") return null;
        if (payload) {
            let newValues = JSON.parse(payload).new;
            let oldValues = JSON.parse(payload).old ?? null;
            let finalOutput = [];

            let properties = new Set([...Object.keys(newValues), ...Object.keys(oldValues)]);

            properties.forEach(property => {
                let change = {
                    prop: property,
                    oldVal: oldValues ? oldValues[property] : null,
                    newVal: newValues[property]
                };
                if (change.oldVal !== change.newVal) {
                    finalOutput.push(change);
                }
            });

            return finalOutput;
        }
        return null;
    },
    isEmpty(value){
        return (value == null || value.length === 0 || value === ' ');
    },
    formatValue(item){
        if(this.isEmpty(item.oldVal) && this.isEmpty(item.newVal)) return "NULL";
        return (this.isEmpty(item.oldVal) ? "NEW" : item.oldVal)
            + " => " +
            (this.isEmpty(item.newVal) ? "DELETED" : item.newVal);
    },
    displayCauserFromId(causerId, availableCausers) {
        if (!causerId || !availableCausers) return '';
        const causer = availableCausers.find(c => c.id === causerId);
        return causer ? causer.username : "Deleted user";
    }
};

export default vueHelper;