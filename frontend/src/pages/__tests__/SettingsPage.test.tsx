import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, beforeEach, vi } from "vitest";
import SettingsPage from "../SettingsPage";

// Mock the alerts API so useQuery doesn't make real HTTP requests
vi.mock("@/api/alerts", () => ({
  fetchAlertRules: vi.fn().mockResolvedValue([]),
  createAlertRule: vi.fn(),
  deleteAlertRule: vi.fn(),
  toggleAlertRule: vi.fn(),
}));

// Mock useActiveCluster to return a stable value
vi.mock("@/hooks/useActiveCluster", () => ({
  useActiveCluster: () => "test-cluster",
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>{children}</MemoryRouter>
      </QueryClientProvider>
    );
  };
}

describe("SettingsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the settings page heading", () => {
    render(<SettingsPage />, { wrapper: createWrapper() });
    expect(screen.getByText("Settings")).toBeInTheDocument();
  });

  it("shows the description text", () => {
    render(<SettingsPage />, { wrapper: createWrapper() });
    expect(
      screen.getByText(
        "User preferences, alert rules, and system configuration.",
      ),
    ).toBeInTheDocument();
  });

  it("renders all three tab buttons", () => {
    render(<SettingsPage />, { wrapper: createWrapper() });
    // "Alert Rules" appears both as a tab button and as an h2 heading,
    // so use getAllByText and verify at least one match exists for each tab.
    expect(screen.getAllByText("Alert Rules").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Notifications")).toBeInTheDocument();
    expect(screen.getByText("Preferences")).toBeInTheDocument();
  });

  it("shows Alert Rules tab content by default", () => {
    render(<SettingsPage />, { wrapper: createWrapper() });
    // The "Create Alert Rule" button is visible in the Alert Rules tab
    expect(
      screen.getByRole("button", { name: "Create Alert Rule" }),
    ).toBeInTheDocument();
  });

  it("shows Preferences tab with theme toggle and namespace input", async () => {
    const { default: userEvent } = await import("@testing-library/user-event");
    const user = userEvent.setup();
    render(<SettingsPage />, { wrapper: createWrapper() });

    await user.click(screen.getByText("Preferences"));

    expect(screen.getByText("Theme")).toBeInTheDocument();
    expect(screen.getByText("Default Namespace")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Switch to .+ Mode/ }),
    ).toBeInTheDocument();
    expect(screen.getByPlaceholderText("default")).toBeInTheDocument();
  });
});
