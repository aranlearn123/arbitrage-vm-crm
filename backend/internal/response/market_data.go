package response

type FundingRate struct {
	Time        string `json:"time"`
	Exchange    string `json:"exchange"`
	Base        string `json:"base"`
	Quote       string `json:"quote"`
	Pair        string `json:"pair"`
	FundingRate string `json:"funding_rate"`
}

type FundingRateList struct {
	Data  []FundingRate `json:"data"`
	Count int           `json:"count"`
	Limit int           `json:"limit"`
}

type FundingSpread struct {
	Time         string `json:"time"`
	Base         string `json:"base"`
	Quote        string `json:"quote"`
	Pair         string `json:"pair"`
	BybitRate    string `json:"bybit_rate"`
	BitgetRate   string `json:"bitget_rate"`
	Spread       string `json:"spread"`
	SpreadBps    string `json:"spread_bps"`
	AbsSpreadBps string `json:"abs_spread_bps"`
	Direction    string `json:"direction_hint"`
	BybitTime    string `json:"bybit_time"`
	BitgetTime   string `json:"bitget_time"`
}

type FundingSpreadDetail struct {
	Data FundingSpread `json:"data"`
}

type FundingSpreadList struct {
	Data  []FundingSpread `json:"data"`
	Count int             `json:"count"`
	Limit int             `json:"limit"`
}

type MarketQualityMetric struct {
	Time                 string `json:"time"`
	Exchange             string `json:"exchange"`
	Base                 string `json:"base"`
	Quote                string `json:"quote"`
	Pair                 string `json:"pair"`
	Samples              int    `json:"samples"`
	SpreadBpsP50         string `json:"spread_bps_p50"`
	MidSpeedBpsPerSecP95 string `json:"mid_speed_bps_per_sec_p95"`
	DepthStabilityRatio  string `json:"depth_stability_ratio"`
}

type MarketQualityList struct {
	Data  []MarketQualityMetric `json:"data"`
	Count int                   `json:"count"`
	Limit int                   `json:"limit"`
}

type MarketQualityAlert struct {
	MarketQualityMetric
	Reasons []string `json:"reasons"`
}

type MarketQualityAlertList struct {
	Data  []MarketQualityAlert `json:"data"`
	Count int                  `json:"count"`
	Limit int                  `json:"limit"`
}
