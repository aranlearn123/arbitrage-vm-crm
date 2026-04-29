package response

type SystemStatus struct {
	Status      string                 `json:"status"`
	Time        string                 `json:"time"`
	APIVersion  string                 `json:"api_version"`
	Database    HealthComponent        `json:"database"`
	Timescale   HealthComponent        `json:"timescale"`
	MarketData  SystemMarketDataStatus `json:"market_data"`
	Allocations SystemAllocationStatus `json:"allocations"`
	Exchanges   []SystemExchange       `json:"exchanges"`
}

type SystemMarketDataStatus struct {
	FundingLastAt       string `json:"funding_last_at,omitempty"`
	OpenInterestLastAt  string `json:"open_interest_last_at,omitempty"`
	MarketQualityLastAt string `json:"market_quality_last_at,omitempty"`
}

type SystemAllocationStatus struct {
	Total         int64  `json:"total"`
	Running       int64  `json:"running"`
	LastUpdatedAt string `json:"last_updated_at,omitempty"`
}

type SystemExchange struct {
	Exchange             string   `json:"exchange"`
	Enabled              bool     `json:"enabled"`
	Supported            bool     `json:"supported"`
	CredentialConfigured bool     `json:"credential_configured"`
	Demo                 bool     `json:"demo"`
	Status               string   `json:"status"`
	LastMarketDataAt     string   `json:"last_market_data_at,omitempty"`
	LastFundingAt        string   `json:"last_funding_at,omitempty"`
	LastOpenInterestAt   string   `json:"last_open_interest_at,omitempty"`
	LastMarketQualityAt  string   `json:"last_market_quality_at,omitempty"`
	LastAccountEventAt   string   `json:"last_account_event_at,omitempty"`
	LastWalletSnapshotAt string   `json:"last_wallet_snapshot_at,omitempty"`
	Notes                []string `json:"notes,omitempty"`
}

type SystemExchangeList struct {
	Data  []SystemExchange `json:"data"`
	Count int              `json:"count"`
}
