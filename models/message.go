package models

import "time"

// MessageRecord mirrors the Messages table in SQL Server.
// PascalCase JSON tags match the C# serialization convention expected by
// ManagementPanelAPI's BroadcastMessageEvent handler.
type MessageRecord struct {
	Id              int        `json:"Id"`
	ClientId        *string    `json:"ClientId,omitempty"`
	Message         string     `json:"Message"`
	IsActive        bool       `json:"IsActive"`
	StartUTC        time.Time  `json:"StartUTC"`
	ExpiresUTC      *time.Time `json:"ExpiresUTC,omitempty"`
	LastRunUTC      *time.Time `json:"LastRunUTC,omitempty"`
	IntervalMinutes int        `json:"IntervalMinutes"`
}
