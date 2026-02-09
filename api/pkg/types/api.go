package types

import "time"

// APIResponse is the standard envelope for all API responses.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError describes an error returned by the API.
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// PaginatedResponse wraps a list response with pagination metadata.
type PaginatedResponse struct {
	Items      interface{}     `json:"items"`
	Pagination PaginationMeta  `json:"pagination"`
}

// PaginationMeta contains pagination information.
type PaginationMeta struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
}

// PaginationParams are query parameters for paginated list requests.
type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

// TimeRange represents a time range for queries.
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// NamespacedName identifies a Kubernetes resource.
type NamespacedName struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// AuditEntry records a change to a resource.
type AuditEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	User      string    `json:"user"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Diff      string    `json:"diff,omitempty"`
}

// MetricsSummary contains aggregated metrics.
type MetricsSummary struct {
	RequestsPerSecond float64            `json:"requestsPerSecond"`
	ErrorRate         float64            `json:"errorRate"`
	P50Latency        float64            `json:"p50Latency"`
	P99Latency        float64            `json:"p99Latency"`
	ByStatusCode      map[int]int64      `json:"byStatusCode,omitempty"`
	Labels            map[string]string  `json:"labels,omitempty"`
}
