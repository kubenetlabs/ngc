import axios from "axios";
import type { AxiosError, InternalAxiosRequestConfig } from "axios";
import { useClusterStore, ALL_CLUSTERS } from "@/store/clusterStore";

const MAX_RETRIES = 2;
const RETRY_DELAY_MS = 1000;

interface RetryableConfig extends InternalAxiosRequestConfig {
  __retryCount?: number;
}

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL || "/api/v1",
  headers: { "Content-Type": "application/json" },
  timeout: 30_000,
});

// Cluster routing interceptor: transparently prefixes active cluster to URLs.
apiClient.interceptors.request.use((config) => {
  const cluster = useClusterStore.getState().activeCluster;
  // Skip prefixing for "All Clusters" mode, cluster management URLs, global endpoints, and health checks.
  if (
    cluster &&
    cluster !== ALL_CLUSTERS &&
    config.url &&
    !config.url.startsWith("/clusters") &&
    !config.url.startsWith("/global")
  ) {
    config.url = `/clusters/${cluster}${config.url}`;
  }
  return config;
});

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const config = error.config as RetryableConfig | undefined;
    if (!config || (config.__retryCount ?? 0) >= MAX_RETRIES) {
      console.error("API Error:", error.response?.data || error.message);
      return Promise.reject(error);
    }

    // Only retry GET requests on server errors or timeouts.
    const isRetryable =
      config.method === "get" &&
      (!error.response || error.response.status >= 500 || error.code === "ECONNABORTED");

    if (!isRetryable) {
      console.error("API Error:", error.response?.data || error.message);
      return Promise.reject(error);
    }

    config.__retryCount = (config.__retryCount ?? 0) + 1;
    const delay = RETRY_DELAY_MS * Math.pow(2, config.__retryCount - 1);
    await new Promise((resolve) => setTimeout(resolve, delay));
    return apiClient(config);
  },
);

export default apiClient;
