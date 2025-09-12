import apiClient from './axios.ts';
import { useAuthStore } from '../stores/auth_store.ts';

let isRetrying = false;

apiClient.interceptors.response.use(
    (response) => response,
    async (error) => {
        const { config, response } = error;
        if (!response) return Promise.reject(error);

        if (response.status === 401 && !config.__retried) {
            config.__retried = true;
            try {
                // if the server minted a new access cookie, retry
                return await apiClient(config);
            } catch (e) {
                // fall through to logout
            }
        }

        if (response.status === 401) {
            const auth = useAuthStore();
            if (auth.isAuthenticated && !isRetrying) {
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