export interface AuditEntry {
  id: string;
  timestamp: string;
  user: string;
  action: string;
  resource: string;
  name: string;
  namespace: string;
  cluster: string;
  beforeJson: Record<string, unknown>;
  afterJson: Record<string, unknown>;
}

export interface AuditListResponse {
  entries: AuditEntry[];
  total: number;
}

export interface AuditDiffResponse {
  id: string;
  action: string;
  resource: string;
  name: string;
  namespace: string;
  beforeJson: Record<string, unknown>;
  afterJson: Record<string, unknown>;
}
