import axios from 'axios';

const isProduction = import.meta.env.VITE_APP_PRODUCTION_MODE === "true";
const httpProtocol = isProduction ? "https" : "http";

const apiClient = axios.create({
    baseURL: `${httpProtocol}://${import.meta.env.VITE_API_BASE_URL}`,
    withCredentials: true,
});

export default apiClient;