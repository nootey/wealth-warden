import apiClient from './axios.ts';
import { useAuthStore } from '../stores/authStore.ts';

// Request Interceptor (Optional if you want to add headers globally)
apiClient.interceptors.request.use((config) => {
    config.headers["wealth-warden-client"] = "true";
    return config;
});

// Response Interceptor for handling 401 errors
apiClient.interceptors.response.use(
    (response) => response,
    (error) => {
        if (error?.response?.status === 401) {
            const authStore = useAuthStore();
            if (authStore.isAuthenticated) {
                authStore.logoutUser().then();
            }
        }
        return Promise.reject(error);
    }
);

export default apiClient;