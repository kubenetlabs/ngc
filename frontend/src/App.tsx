import { Suspense, useEffect } from "react";
import { createBrowserRouter, RouterProvider } from "react-router-dom";
import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { routes } from "./routes";
import { fetchConfig } from "./api/config";
import { useSettingsStore } from "./store/settingsStore";
import { useActiveCluster } from "./hooks/useActiveCluster";
import { ErrorBoundary } from "./components/common/ErrorBoundary";

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
  const edition = useSettingsStore((s) => s.edition);
  const activeCluster = useActiveCluster();
  const { data } = useQuery({
    queryKey: ["config", activeCluster],
    queryFn: fetchConfig,
    staleTime: 60_000, // Re-check every 60s so transient failures self-correct
    refetchOnWindowFocus: true,
  });

  useEffect(() => {
    if (!data?.edition) return;

    const incoming = data.edition;

    // Never downgrade a known-good "enterprise" to "unknown" or "oss" —
    // transient API errors during pod restarts can cause false downgrades.
    // Only overwrite if: incoming is "enterprise" (always accept upgrade),
    // or the current value is "unknown" (no good value yet).
    if (incoming === "enterprise") {
      setEdition(incoming);
    } else if (edition === "unknown") {
      setEdition(incoming);
    }
    // If edition is already "enterprise" and incoming is "oss" or "unknown",
    // we keep the stored "enterprise" — it can only be downgraded by an
    // explicit "oss" response when the current stored value is "unknown".
  }, [data?.edition, edition, setEdition]);

  return null;
}

export default function App() {
  return (
    <ErrorBoundary>
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
    </ErrorBoundary>
  );
}
