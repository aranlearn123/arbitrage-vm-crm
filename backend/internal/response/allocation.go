package response

type Allocation struct {
	ID        int64   `json:"id"`
	Base      string  `json:"base"`
	Quote     string  `json:"quote"`
	Pair      string  `json:"pair"`
	Direction string  `json:"direction"`
	Rank      int     `json:"rank"`
	Score     string  `json:"score"`
	Role      string  `json:"role"`
	Status    string  `json:"status"`
	BudgetUSD string  `json:"budget_usd"`
	WorkerPID *string `json:"worker_pid,omitempty"`
	Note      *string `json:"note,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type AllocationSummary struct {
	Total             int64                  `json:"total"`
	Active            int64                  `json:"active"`
	Cancelled         int64                  `json:"cancelled"`
	ActiveBudgetUSD   string                 `json:"active_budget_usd"`
	ByStatus          map[string]int64       `json:"by_status"`
	CancelledByReason []CancelledReasonCount `json:"cancelled_by_reason,omitempty"`
}

type CancelledReasonCount struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

type AllocationList struct {
	Data  []Allocation `json:"data"`
	Count int          `json:"count"`
	Limit int          `json:"limit"`
}

type CancelledReasonList struct {
	Data  []CancelledReasonCount `json:"data"`
	Count int                    `json:"count"`
	Limit int                    `json:"limit"`
}

type Error struct {
	Error string `json:"error"`
}
