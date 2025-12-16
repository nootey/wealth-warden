import apiClient from './axios.ts';
import { useAuthStore } from '../stores/auth_store.ts';

let isRetrying = false;

apiClient.interceptors.response.use(
    (response) => response,
    async (error) => {
        const { config, response } = error;
        const url: string = error?.config?.url ?? '';

        if (!response) return Promise.reject(error);

        if (response.status === 401 && !config._retry) {
            config._retry = true;

            try {
                // The server already set a new access cookie, retry
                return await apiClient(config);
            } catch (retryError) {
                // If retry fails, fall through to logout
            }
        }

        if (response.status === 401) {
            const auth = useAuthStore();
            const isAuthEndpoint = /\/auth\/(current|logout|login)/.test(url);

            if (!isAuthEndpoint && auth.isAuthenticated && !isRetrying) {
                isRetrying = true;
                try {
                    await auth.logoutUser();
                } finally {
                    isRetrying = false;
                }
            }
        }

        return Promise.reject(error);
    }
);

export default apiClient;