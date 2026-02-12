import { useEffect, useRef, useCallback, useState } from "react";
import { useActiveCluster } from "./useActiveCluster";
import { ALL_CLUSTERS } from "@/store/clusterStore";

interface UseWebSocketOptions {
  url: string;
  onMessage?: (data: unknown) => void;
  reconnectInterval?: number;
  maxReconnectInterval?: number;
  enabled?: boolean;
}

export function useWebSocket({
  url,
  onMessage,
  reconnectInterval = 1000,
  maxReconnectInterval = 30000,
  enabled = true,
}: UseWebSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const attemptRef = useRef(0);
  const onMessageRef = useRef(onMessage);
  const [connected, setConnected] = useState(false);
  const activeCluster = useActiveCluster();

  // Keep onMessage ref current without triggering reconnects
  useEffect(() => {
    onMessageRef.current = onMessage;
  }, [onMessage]);

  useEffect(() => {
    if (!enabled) return;

    function connect() {
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      const separator = url.includes("?") ? "&" : "?";
      const clusterParam = activeCluster && activeCluster !== ALL_CLUSTERS
        ? `${separator}cluster=${activeCluster}`
        : "";
      const wsUrl = `${protocol}//${window.location.host}${url}${clusterParam}`;
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        attemptRef.current = 0;
        setConnected(true);
      };
      ws.onclose = () => {
        setConnected(false);
        // Exponential backoff with jitter, capped at maxReconnectInterval.
        const delay = Math.min(
          reconnectInterval * Math.pow(2, attemptRef.current) + Math.random() * 1000,
          maxReconnectInterval,
        );
        attemptRef.current++;
        reconnectTimerRef.current = setTimeout(connect, delay);
      };
      ws.onerror = () => {
        // Close will fire after error, triggering reconnect.
        ws.close();
      };
      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          onMessageRef.current?.(data);
        } catch {
          onMessageRef.current?.(event.data);
        }
      };

      wsRef.current = ws;
    }

    connect();

    return () => {
      clearTimeout(reconnectTimerRef.current);
      wsRef.current?.close();
    };
  }, [url, reconnectInterval, maxReconnectInterval, enabled, activeCluster]);

  const send = useCallback((data: unknown) => {
    wsRef.current?.send(JSON.stringify(data));
  }, []);

  return { connected, send };
}
