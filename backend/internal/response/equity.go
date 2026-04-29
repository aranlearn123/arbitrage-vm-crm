package response

type EquityCoin struct {
	Coin             string `json:"coin"`
	Equity           string `json:"equity"`
	WalletBalance    string `json:"wallet_balance"`
	AvailableBalance string `json:"available_balance"`
	Locked           string `json:"locked"`
	UnrealizedPnL    string `json:"unrealized_pnl"`
	USDValue         string `json:"usd_value"`
}

type EquitySnapshot struct {
	Exchange          string       `json:"exchange"`
	Time              string       `json:"time"`
	AccountEquity     string       `json:"account_equity"`
	WalletBalance     string       `json:"wallet_balance"`
	AvailableBalance  string       `json:"available_balance"`
	UnrealizedPnL     string       `json:"unrealized_pnl"`
	InitialMargin     string       `json:"initial_margin"`
	MaintenanceMargin string       `json:"maintenance_margin"`
	Coins             []EquityCoin `json:"coins"`
	Source            string       `json:"source"`
	Cached            bool         `json:"cached"`
}

type CombinedEquity struct {
	Time              string `json:"time"`
	AccountEquity     string `json:"account_equity"`
	WalletBalance     string `json:"wallet_balance"`
	AvailableBalance  string `json:"available_balance"`
	UnrealizedPnL     string `json:"unrealized_pnl"`
	InitialMargin     string `json:"initial_margin"`
	MaintenanceMargin string `json:"maintenance_margin"`
}

type EquityFetchError struct {
	Exchange string `json:"exchange"`
	Error    string `json:"error"`
}

type EquityLatest struct {
	Data            []EquitySnapshot   `json:"data"`
	Combined        CombinedEquity     `json:"combined"`
	Errors          []EquityFetchError `json:"errors,omitempty"`
	Count           int                `json:"count"`
	CacheTTLSeconds int                `json:"cache_ttl_seconds"`
}
