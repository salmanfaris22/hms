package model

import (
	"encoding/json"
	"time"
)

type LogEntryDTO struct {
	ID         string          `json:"id"`
	OccurredAt time.Time       `json:"occurredAt"`
	ActorEmail string          `json:"actorEmail"`
	ActorKind  string          `json:"actorKind"`
	Action     string          `json:"action"`
	Resource   string          `json:"resource"`
	IP         string          `json:"ip"`
	UserAgent  string          `json:"userAgent"`
	Metadata   json.RawMessage `json:"metadata"`
}

type ListFilter struct {
	TenantID string
	Page     int
	PageSize int
	Search   string
	User     string
	Category string
	From     string
	To       string
}

type ListResult struct {
	Logs     []LogEntryDTO `json:"logs"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
	Users    []string      `json:"users"`
}

type ExportFilter struct {
	TenantID string
	From     string
	To       string
}

type ExportRow struct {
	OccurredAt time.Time
	ActorEmail string
	ActorKind  string
	Action     string
	Resource   string
	IP         string
}
