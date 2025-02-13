import { defineStore } from 'pinia';
import { useToast } from 'primevue/usetoast';

export const useToastStore = defineStore('toast', () => {
    const toast = useToast();

    const errorResponseToast = (error: any) => {
        console.error(error);

        let errorTitle = "Error occurred";
        let errorMsg = error ?? "An error has occurred without additional information.";

        if (error?.response?.data?.errors) {
            let errors = error.response.data.errors;
            for (let key in errors) {
                if (errors.hasOwnProperty(key)) {
                    toast.add({
                        severity: 'error',
                        summary: key ?? 'Error occurred',
                        detail: errors[key] ?? "...",
                        life: 5000
                    });
                }
            }
            return;
        }

        if (error.code === 'ERR_NETWORK' || error.message === 'Network Error') {
            errorTitle = "Server unreachable";
            errorMsg = "The server is currently not reachable.";
        }

        if (error?.response?.data) {
            errorTitle = error.response.data.title;
            errorMsg = error?.response?.data?.message || error.response.data;
        }

        toast.add({
            severity: 'error',
            summary: errorTitle,
            detail: errorMsg,
            life: 5000
        });
    };

    const successResponseToast = (response: any) => {
        if (response.data) {
            toast.add({
                severity: 'success',
                summary: response.data.title,
                detail: response.data.message,
                life: 3000
            });
        }
    };

    const infoResponseToast = (response: any) => {
        if (response.data) {
            toast.add({
                severity: 'info',
                summary: response.data.title,
                detail: response.data.message,
                life: 3000
            });
        }
    };

    return { errorResponseToast, successResponseToast, infoResponseToast };
});
