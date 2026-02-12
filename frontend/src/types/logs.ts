export interface LogEntry {
  timestamp: string;
  hostname: string;
  path: string;
  method: string;
  statusCode: number;
  latency: number;
  namespace: string;
  upstreamService: string;
}

export interface LogQueryRequest {
  namespace?: string;
  hostname?: string;
  statusMin?: number;
  statusMax?: number;
  since?: string;
  until?: string;
  search?: string;
  limit?: number;
}

export interface TopNEntry {
  key: string;
  count: number;
  percentage: number;
}
