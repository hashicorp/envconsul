package util

// StatsData is used for /stats endpoints
type StatsData struct {
	Stats     map[string]int64 `json:"stats"`
	RequestID string           `json:"request_id"`
}

// HealthData is used for /health endpoints
type HealthData struct {
	Health    bool   `json:"health"`
	RequestID string `json:"request_id"`
}
