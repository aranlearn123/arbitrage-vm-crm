package response

type Health struct {
	Status string       `json:"status"`
	Time   string       `json:"time"`
	Checks HealthChecks `json:"checks"`
}

type HealthChecks struct {
	API      HealthComponent `json:"api"`
	Database HealthComponent `json:"database"`
}

type HealthComponent struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}
