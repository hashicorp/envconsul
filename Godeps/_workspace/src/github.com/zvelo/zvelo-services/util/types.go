package util

import "github.com/zvelo/zvelo-services/stats"

// StatsData is used for /stats endpoints
type StatsData struct {
	Stats     stats.Data `json:"stats"`
	RequestID string     `json:"request_id"`
}

// HealthData is used for /health endpoints
type HealthData struct {
	Health    bool   `json:"health"`
	RequestID string `json:"request_id"`
}
