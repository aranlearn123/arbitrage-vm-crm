package response

type PnLEvent struct {
	EventTime  string `json:"event_time"`
	Exchange   string `json:"exchange"`
	Base       string `json:"base"`
	Quote      string `json:"quote"`
	Pair       string `json:"pair"`
	Component  string `json:"component"`
	Amount     string `json:"amount"`
	Currency   string `json:"currency"`
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
	CreatedAt  string `json:"created_at"`
}

type PnLEventList struct {
	Data  []PnLEvent `json:"data"`
	Count int        `json:"count"`
	Limit int        `json:"limit"`
}

type PnLSummary struct {
	Count            int64      `json:"count"`
	TotalAmount      string     `json:"total_amount"`
	FundingAmount    string     `json:"funding_amount"`
	TradingFeeAmount string     `json:"trading_fee_amount"`
	TradingPnLAmount string     `json:"trading_pnl_amount"`
	ByComponent      []PnLGroup `json:"by_component,omitempty"`
	ByExchange       []PnLGroup `json:"by_exchange,omitempty"`
	ByPair           []PnLGroup `json:"by_pair,omitempty"`
}

type PnLGroup struct {
	Exchange         string `json:"exchange,omitempty"`
	Base             string `json:"base,omitempty"`
	Quote            string `json:"quote,omitempty"`
	Pair             string `json:"pair,omitempty"`
	Component        string `json:"component,omitempty"`
	Count            int64  `json:"count"`
	TotalAmount      string `json:"total_amount"`
	FundingAmount    string `json:"funding_amount,omitempty"`
	TradingFeeAmount string `json:"trading_fee_amount,omitempty"`
	TradingPnLAmount string `json:"trading_pnl_amount,omitempty"`
	FirstEventTime   string `json:"first_event_time,omitempty"`
	LastEventTime    string `json:"last_event_time,omitempty"`
}

type PnLGroupList struct {
	Data  []PnLGroup `json:"data"`
	Count int        `json:"count"`
	Limit int        `json:"limit"`
}
