package models

import "time"

// AuditLog representa la entidad AUDIT_LOG para trazabilidad/accounting.
type AuditLog struct {
	IDLog      int64     `json:"idLog"`
	IDUser     *int      `json:"idUser,omitempty"`
	Role       string    `json:"role"`
	Action     string    `json:"action"`
	Method     string    `json:"method"`
	Endpoint   string    `json:"endpoint"`
	IPAddress  string    `json:"ipAddress"`
	StatusCode int       `json:"statusCode"`
	LatencyMs  int       `json:"latencyMs"`
	Details    *string   `json:"details,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}
