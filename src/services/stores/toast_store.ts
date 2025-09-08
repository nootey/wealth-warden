import { defineStore } from 'pinia';
import { useToast } from 'primevue/usetoast';

export const useToastStore = defineStore('toast', () => {
    const toast = useToast();

    const errorResponseToast = (error: any) => {

        console.error("triggered error", error);

        const data = error?.response?.data;
        let summary = 'Unexpected Error';
        let detail = error?.message ?? 'An unknown error occurred.';

        if (data?.title || data?.message) {
            summary = data.title ?? 'Error';
            detail = data.message ?? 'Something went wrong.';
        }

        if (error?.code === 'ERR_NETWORK' || error?.message === 'Network Error') {
            summary = 'Server unreachable';
            detail = 'The server is currently not reachable.';
        }

        toast.add({
            severity: 'error',
            summary,
            detail,
            life: 5000,
        });
    };

    const successResponseToast = (response: any) => {
        const data = response?.data || response;
        if (data?.title || data?.message) {
            toast.add({
                severity: 'success',
                summary: data.title ?? 'Success',
                detail: data.message ?? '',
                life: 3000,
            });
        }
    };
    const infoResponseToast = (response: any) => {
        const data = response?.data || response;
        if (data?.title || data?.message) {
            toast.add({
                severity: 'info',
                summary: data.title ?? 'Info',
                detail: data.message ?? '',
                life: 3000,
            });
        }
    };

    return {
        errorResponseToast,
        successResponseToast,
        infoResponseToast
    };
    
});
