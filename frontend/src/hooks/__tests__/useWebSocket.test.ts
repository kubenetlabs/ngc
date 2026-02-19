import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { useWebSocket } from "../useWebSocket";

// Mock useActiveCluster
vi.mock("@/hooks/useActiveCluster", () => ({
  useActiveCluster: vi.fn(() => ""),
}));

// Mock clusterStore's ALL_CLUSTERS constant
vi.mock("@/store/clusterStore", () => ({
  ALL_CLUSTERS: "all",
}));

// --- Mock WebSocket ---
type WSEventHandler = ((event: unknown) => void) | null;

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  url: string;
  onopen: WSEventHandler = null;
  onclose: WSEventHandler = null;
  onerror: WSEventHandler = null;
  onmessage: WSEventHandler = null;
  readyState = 0; // CONNECTING
  closeCalled = false;

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  close() {
    this.closeCalled = true;
  }

  send = vi.fn();

  // Helper to simulate server events
  simulateOpen() {
    this.readyState = 1;
    this.onopen?.({});
  }

  simulateMessage(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) });
  }

  simulateClose() {
    this.readyState = 3;
    this.onclose?.({});
  }
}

describe("useWebSocket", () => {
  beforeEach(() => {
    MockWebSocket.instances = [];
    vi.stubGlobal("WebSocket", MockWebSocket);
    // Mock location for URL construction
    Object.defineProperty(window, "location", {
      value: { protocol: "https:", host: "localhost:3000" },
      writable: true,
    });
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("connects to correct URL with wss: for https:", () => {
    renderHook(() =>
      useWebSocket({ url: "/api/v1/ws/test" }),
    );

    expect(MockWebSocket.instances).toHaveLength(1);
    expect(MockWebSocket.instances[0].url).toBe(
      "wss://localhost:3000/api/v1/ws/test",
    );
  });

  it("connects with ws: for http:", () => {
    Object.defineProperty(window, "location", {
      value: { protocol: "http:", host: "localhost:3000" },
      writable: true,
    });

    renderHook(() =>
      useWebSocket({ url: "/api/v1/ws/test" }),
    );

    expect(MockWebSocket.instances[0].url).toBe(
      "ws://localhost:3000/api/v1/ws/test",
    );
  });

  it("appends cluster param when active cluster is set", async () => {
    const { useActiveCluster } = await import("@/hooks/useActiveCluster");
    vi.mocked(useActiveCluster).mockReturnValue("prod-cluster");

    renderHook(() =>
      useWebSocket({ url: "/api/v1/ws/test" }),
    );

    expect(MockWebSocket.instances[0].url).toContain("?cluster=prod-cluster");
  });

  it("does not append cluster param for ALL_CLUSTERS", async () => {
    const { useActiveCluster } = await import("@/hooks/useActiveCluster");
    vi.mocked(useActiveCluster).mockReturnValue("all");

    renderHook(() =>
      useWebSocket({ url: "/api/v1/ws/test" }),
    );

    expect(MockWebSocket.instances[0].url).not.toContain("cluster=");
  });

  it("sets connected to true on open", () => {
    const { result } = renderHook(() =>
      useWebSocket({ url: "/api/v1/ws/test" }),
    );

    expect(result.current.connected).toBe(false);

    act(() => {
      MockWebSocket.instances[0].simulateOpen();
    });

    expect(result.current.connected).toBe(true);
  });

  it("calls onMessage with parsed JSON data", () => {
    const onMessage = vi.fn();
    renderHook(() =>
      useWebSocket({ url: "/api/v1/ws/test", onMessage }),
    );

    act(() => {
      MockWebSocket.instances[0].simulateOpen();
      MockWebSocket.instances[0].simulateMessage({ type: "update", value: 42 });
    });

    expect(onMessage).toHaveBeenCalledWith({ type: "update", value: 42 });
  });

  it("reconnects with backoff on close", () => {
    renderHook(() =>
      useWebSocket({
        url: "/api/v1/ws/test",
        reconnectInterval: 1000,
      }),
    );

    expect(MockWebSocket.instances).toHaveLength(1);

    act(() => {
      MockWebSocket.instances[0].simulateClose();
    });

    // Advance past reconnect delay (1000ms * 2^0 + up to 1000ms jitter)
    act(() => {
      vi.advanceTimersByTime(2100);
    });

    expect(MockWebSocket.instances).toHaveLength(2);
  });

  it("does not connect when enabled is false", () => {
    renderHook(() =>
      useWebSocket({ url: "/api/v1/ws/test", enabled: false }),
    );

    expect(MockWebSocket.instances).toHaveLength(0);
  });

  it("cleans up on unmount", () => {
    const { unmount } = renderHook(() =>
      useWebSocket({ url: "/api/v1/ws/test" }),
    );

    const ws = MockWebSocket.instances[0];
    expect(ws.closeCalled).toBe(false);

    unmount();
    expect(ws.closeCalled).toBe(true);
  });

  it("sends JSON data via send function", () => {
    const { result } = renderHook(() =>
      useWebSocket({ url: "/api/v1/ws/test" }),
    );

    act(() => {
      MockWebSocket.instances[0].simulateOpen();
      result.current.send({ action: "subscribe", topic: "metrics" });
    });

    expect(MockWebSocket.instances[0].send).toHaveBeenCalledWith(
      JSON.stringify({ action: "subscribe", topic: "metrics" }),
    );
  });
});
