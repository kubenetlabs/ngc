import { Suspense, useEffect } from "react";
import { createBrowserRouter, RouterProvider } from "react-router-dom";
import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { routes } from "./routes";
import { fetchConfig } from "./api/config";
import { useSettingsStore } from "./store/settingsStore";
import { useActiveCluster } from "./hooks/useActiveCluster";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
});

const router = createBrowserRouter(routes);

function EditionDetector() {
  const setEdition = useSettingsStore((s) => s.setEdition);
  const activeCluster = useActiveCluster();
  const { data } = useQuery({
    queryKey: ["config", activeCluster],
    queryFn: fetchConfig,
    staleTime: Infinity,
  });

  useEffect(() => {
    if (data?.edition) {
      setEdition(data.edition);
    }
  }, [data?.edition, setEdition]);

  return null;
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <EditionDetector />
      <Suspense
        fallback={
          <div className="flex h-screen items-center justify-center">
            <div className="text-muted-foreground">Loading...</div>
          </div>
        }
      >
        <RouterProvider router={router} />
      </Suspense>
    </QueryClientProvider>
  );
}
