import axios from "axios";
import { useClusterStore, ALL_CLUSTERS } from "@/store/clusterStore";

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL || "/api/v1",
  headers: { "Content-Type": "application/json" },
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
  (error) => {
    console.error("API Error:", error.response?.data || error.message);
    return Promise.reject(error);
  },
);

export default apiClient;
