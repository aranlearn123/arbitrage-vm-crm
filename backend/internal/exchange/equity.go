package exchange

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ExchangeBybit  = "bybit"
	ExchangeBitget = "bitget"
)

type Config struct {
	CacheTTL         time.Duration
	BybitDemo        bool
	BybitAPIKey      string
	BybitAPISecret   string
	BitgetDemo       bool
	BitgetAPIKey     string
	BitgetAPISecret  string
	BitgetPassphrase string
}

type EquityService struct {
	clients  map[string]walletClient
	cacheTTL time.Duration
	mu       sync.Mutex
	cache    map[string]cacheEntry
}

type WalletSnapshot struct {
	Exchange          string
	Time              time.Time
	AccountEquity     string
	WalletBalance     string
	AvailableBalance  string
	UnrealizedPnL     string
	InitialMargin     string
	MaintenanceMargin string
	Coins             []CoinBalance
	Source            string
	Cached            bool
}

type CoinBalance struct {
	Coin             string
	Equity           string
	WalletBalance    string
	AvailableBalance string
	Locked           string
	UnrealizedPnL    string
	USDValue         string
}

type FetchResult struct {
	Snapshots []WalletSnapshot
	Combined  CombinedEquity
	Errors    []FetchError
	CacheTTL  time.Duration
}

type CombinedEquity struct {
	Time              time.Time
	AccountEquity     string
	WalletBalance     string
	AvailableBalance  string
	UnrealizedPnL     string
	InitialMargin     string
	MaintenanceMargin string
}

type FetchError struct {
	Exchange string
	Message  string
}

type walletClient interface {
	Exchange() string
	Configured() bool
	FetchWallet(ctx context.Context, quotes []string) (WalletSnapshot, error)
}

type cacheEntry struct {
	expiresAt time.Time
	result    FetchResult
}

func NewEquityService(cfg Config) *EquityService {
	ttl := cfg.CacheTTL
	if ttl <= 0 {
		ttl = 15 * time.Second
	}

	clients := map[string]walletClient{
		ExchangeBybit: &bybitWalletClient{
			apiKey:    strings.TrimSpace(cfg.BybitAPIKey),
			apiSecret: strings.TrimSpace(cfg.BybitAPISecret),
			demo:      cfg.BybitDemo,
			client:    &http.Client{Timeout: 10 * time.Second},
		},
		ExchangeBitget: &bitgetWalletClient{
			apiKey:     strings.TrimSpace(cfg.BitgetAPIKey),
			apiSecret:  strings.TrimSpace(cfg.BitgetAPISecret),
			passphrase: strings.TrimSpace(cfg.BitgetPassphrase),
			demo:       cfg.BitgetDemo,
			client:     &http.Client{Timeout: 10 * time.Second},
		},
	}

	return &EquityService{
		clients:  clients,
		cacheTTL: ttl,
		cache:    make(map[string]cacheEntry),
	}
}

func (s *EquityService) Fetch(ctx context.Context, exchanges []string, quotes []string, refresh bool) (FetchResult, error) {
	if s == nil {
		return FetchResult{}, errors.New("equity service is not configured")
	}

	exchanges = normalizeExchanges(exchanges, s.clients)
	quotes = normalizeQuotes(quotes)
	if len(exchanges) == 0 {
		return FetchResult{}, errors.New("no configured exchange credentials")
	}

	key := cacheKey(exchanges, quotes)
	if !refresh {
		if cached, ok := s.getCache(key); ok {
			return cached, nil
		}
	}

	result := FetchResult{CacheTTL: s.cacheTTL}
	for _, name := range exchanges {
		client := s.clients[name]
		if client == nil {
			result.Errors = append(result.Errors, FetchError{Exchange: name, Message: "unsupported exchange"})
			continue
		}
		if !client.Configured() {
			result.Errors = append(result.Errors, FetchError{Exchange: name, Message: "exchange credential is not configured"})
			continue
		}

		snapshot, err := client.FetchWallet(ctx, quotes)
		if err != nil {
			result.Errors = append(result.Errors, FetchError{Exchange: name, Message: err.Error()})
			continue
		}
		result.Snapshots = append(result.Snapshots, snapshot)
	}

	result.Combined = combineSnapshots(result.Snapshots)
	s.setCache(key, result)
	return result, nil
}

func (s *EquityService) getCache(key string) (FetchResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.cache[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return FetchResult{}, false
	}

	result := entry.result
	for i := range result.Snapshots {
		result.Snapshots[i].Cached = true
	}
	return result, true
}

func (s *EquityService) setCache(key string, result FetchResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[key] = cacheEntry{
		expiresAt: time.Now().Add(s.cacheTTL),
		result:    result,
	}
}

func normalizeExchanges(values []string, clients map[string]walletClient) []string {
	if len(values) == 0 {
		out := make([]string, 0, len(clients))
		for name, client := range clients {
			if client != nil && client.Configured() {
				out = append(out, name)
			}
		}
		sort.Strings(out)
		return out
	}

	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		name := strings.ToLower(strings.TrimSpace(value))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func normalizeQuotes(values []string) []string {
	if len(values) == 0 {
		return []string{"USDT"}
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		quote := strings.ToUpper(strings.TrimSpace(value))
		if quote == "" || seen[quote] {
			continue
		}
		seen[quote] = true
		out = append(out, quote)
	}
	if len(out) == 0 {
		return []string{"USDT"}
	}
	return out
}

func cacheKey(exchanges []string, quotes []string) string {
	return strings.Join(exchanges, ",") + "|" + strings.Join(quotes, ",")
}

func combineSnapshots(rows []WalletSnapshot) CombinedEquity {
	out := CombinedEquity{Time: time.Now().UTC()}
	for _, row := range rows {
		if row.Time.After(out.Time) {
			out.Time = row.Time
		}
		out.AccountEquity = addDecimalStrings(out.AccountEquity, row.AccountEquity)
		out.WalletBalance = addDecimalStrings(out.WalletBalance, row.WalletBalance)
		out.AvailableBalance = addDecimalStrings(out.AvailableBalance, row.AvailableBalance)
		out.UnrealizedPnL = addDecimalStrings(out.UnrealizedPnL, row.UnrealizedPnL)
		out.InitialMargin = addDecimalStrings(out.InitialMargin, row.InitialMargin)
		out.MaintenanceMargin = addDecimalStrings(out.MaintenanceMargin, row.MaintenanceMargin)
	}
	return out
}

func addDecimalStrings(a string, b string) string {
	ar := parseRat(a)
	br := parseRat(b)
	ar.Add(ar, br)
	return formatRat(ar)
}

func parseRat(value string) *big.Rat {
	value = strings.TrimSpace(value)
	if value == "" {
		return new(big.Rat)
	}
	r := new(big.Rat)
	if _, ok := r.SetString(value); !ok {
		return new(big.Rat)
	}
	return r
}

func formatRat(value *big.Rat) string {
	if value == nil {
		return "0"
	}
	out := value.FloatString(18)
	out = strings.TrimRight(out, "0")
	out = strings.TrimRight(out, ".")
	if out == "" || out == "-0" {
		return "0"
	}
	return out
}

func httpGetJSON(ctx context.Context, client *http.Client, rawURL string, headers map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("exchange api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return err
	}
	return nil
}

type bybitWalletClient struct {
	apiKey    string
	apiSecret string
	demo      bool
	client    *http.Client
}

type bybitWalletResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []bybitWalletAccount `json:"list"`
	} `json:"result"`
}

type bybitWalletAccount struct {
	AccountType            string            `json:"accountType"`
	TotalEquity            string            `json:"totalEquity"`
	TotalWalletBalance     string            `json:"totalWalletBalance"`
	TotalAvailableBalance  string            `json:"totalAvailableBalance"`
	TotalPerpUPL           string            `json:"totalPerpUPL"`
	TotalInitialMargin     string            `json:"totalInitialMargin"`
	TotalMaintenanceMargin string            `json:"totalMaintenanceMargin"`
	Coin                   []bybitWalletCoin `json:"coin"`
}

type bybitWalletCoin struct {
	Coin          string `json:"coin"`
	Equity        string `json:"equity"`
	UsdValue      string `json:"usdValue"`
	WalletBalance string `json:"walletBalance"`
	UnrealisedPnL string `json:"unrealisedPnl"`
	Locked        string `json:"locked"`
}

func (c *bybitWalletClient) Exchange() string { return ExchangeBybit }

func (c *bybitWalletClient) Configured() bool {
	return c != nil && c.apiKey != "" && c.apiSecret != ""
}

func (c *bybitWalletClient) FetchWallet(ctx context.Context, quotes []string) (WalletSnapshot, error) {
	endpoint := "https://api.bybit.com"
	if c.demo {
		endpoint = "https://api-demo.bybit.com"
	}

	query := url.Values{}
	query.Set("accountType", "UNIFIED")
	if len(quotes) > 0 {
		query.Set("coin", strings.Join(quotes, ","))
	}
	queryString := query.Encode()
	path := "/v5/account/wallet-balance"
	timestamp := time.Now().UnixMilli()
	recvWindow := int64(5000)

	headers := map[string]string{
		"X-BAPI-API-KEY":     c.apiKey,
		"X-BAPI-TIMESTAMP":   strconv.FormatInt(timestamp, 10),
		"X-BAPI-RECV-WINDOW": strconv.FormatInt(recvWindow, 10),
		"X-BAPI-SIGN":        bybitSignQuery(timestamp, c.apiKey, recvWindow, queryString, c.apiSecret),
	}

	var res bybitWalletResponse
	if err := httpGetJSON(ctx, c.client, endpoint+path+"?"+queryString, headers, &res); err != nil {
		return WalletSnapshot{}, err
	}
	if res.RetCode != 0 {
		return WalletSnapshot{}, fmt.Errorf("bybit: %s (retCode=%d)", res.RetMsg, res.RetCode)
	}
	if len(res.Result.List) == 0 {
		return WalletSnapshot{Exchange: ExchangeBybit, Time: time.Now().UTC(), Source: "exchange_api"}, nil
	}

	acct := res.Result.List[0]
	out := WalletSnapshot{
		Exchange:          ExchangeBybit,
		Time:              time.Now().UTC(),
		AccountEquity:     acct.TotalEquity,
		WalletBalance:     acct.TotalWalletBalance,
		AvailableBalance:  acct.TotalAvailableBalance,
		UnrealizedPnL:     acct.TotalPerpUPL,
		InitialMargin:     acct.TotalInitialMargin,
		MaintenanceMargin: acct.TotalMaintenanceMargin,
		Source:            "exchange_api",
	}
	for _, coin := range acct.Coin {
		out.Coins = append(out.Coins, CoinBalance{
			Coin:          coin.Coin,
			Equity:        coin.Equity,
			WalletBalance: coin.WalletBalance,
			Locked:        coin.Locked,
			UnrealizedPnL: coin.UnrealisedPnL,
			USDValue:      coin.UsdValue,
		})
	}
	return out, nil
}

func bybitSignQuery(timestamp int64, key string, recvWindow int64, queryString string, secret string) string {
	payload := strconv.FormatInt(timestamp, 10) + key + strconv.FormatInt(recvWindow, 10) + queryString
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

type bitgetWalletClient struct {
	apiKey     string
	apiSecret  string
	passphrase string
	demo       bool
	client     *http.Client
}

type bitgetWalletResponse struct {
	Code string                `json:"code"`
	Msg  string                `json:"msg"`
	Data []bitgetAccountRecord `json:"data"`
}

type bitgetAccountRecord struct {
	MarginCoin           string `json:"marginCoin"`
	Locked               string `json:"locked"`
	Available            string `json:"available"`
	CrossedMaxAvailable  string `json:"crossedMaxAvailable"`
	AccountEquity        string `json:"accountEquity"`
	UsdtEquity           string `json:"usdtEquity"`
	CrossedUnrealizedPL  string `json:"crossedUnrealizedPL"`
	IsolatedUnrealizedPL string `json:"isolatedUnrealizedPL"`
	CrossedMargin        string `json:"crossedMargin"`
	IsolatedMargin       string `json:"isolatedMargin"`
}

func (c *bitgetWalletClient) Exchange() string { return ExchangeBitget }

func (c *bitgetWalletClient) Configured() bool {
	return c != nil && c.apiKey != "" && c.apiSecret != "" && c.passphrase != ""
}

func (c *bitgetWalletClient) FetchWallet(ctx context.Context, quotes []string) (WalletSnapshot, error) {
	out := WalletSnapshot{
		Exchange: ExchangeBitget,
		Time:     time.Now().UTC(),
		Source:   "exchange_api",
	}

	for _, quote := range normalizeQuotes(quotes) {
		rows, err := c.fetchAccounts(ctx, quote)
		if err != nil {
			return WalletSnapshot{}, err
		}
		for _, acct := range rows {
			if acct.MarginCoin != "" && !strings.EqualFold(acct.MarginCoin, quote) {
				continue
			}
			available := acct.Available
			locked := acct.Locked
			unrealized := addDecimalStrings(acct.CrossedUnrealizedPL, acct.IsolatedUnrealizedPL)
			initialMargin := addDecimalStrings(acct.CrossedMargin, acct.IsolatedMargin)
			walletBalance := addDecimalStrings(available, locked)

			out.Coins = append(out.Coins, CoinBalance{
				Coin:             acct.MarginCoin,
				Equity:           acct.AccountEquity,
				WalletBalance:    walletBalance,
				AvailableBalance: available,
				Locked:           locked,
				UnrealizedPnL:    unrealized,
				USDValue:         acct.UsdtEquity,
			})
			out.AccountEquity = addDecimalStrings(out.AccountEquity, acct.UsdtEquity)
			out.WalletBalance = addDecimalStrings(out.WalletBalance, walletBalance)
			out.AvailableBalance = addDecimalStrings(out.AvailableBalance, acct.CrossedMaxAvailable)
			out.UnrealizedPnL = addDecimalStrings(out.UnrealizedPnL, unrealized)
			out.InitialMargin = addDecimalStrings(out.InitialMargin, initialMargin)
		}
	}
	return out, nil
}

func (c *bitgetWalletClient) fetchAccounts(ctx context.Context, quote string) ([]bitgetAccountRecord, error) {
	path := "/api/v2/mix/account/accounts"
	query := url.Values{}
	query.Set("productType", bitgetProductType(quote))
	queryString := query.Encode()
	timestamp := time.Now().UnixMilli()
	headers := map[string]string{
		"ACCESS-KEY":        c.apiKey,
		"ACCESS-TIMESTAMP":  strconv.FormatInt(timestamp, 10),
		"ACCESS-SIGN":       bitgetSignQuery(c.apiSecret, timestamp, http.MethodGet, path, queryString),
		"ACCESS-PASSPHRASE": c.passphrase,
		"LOCAL":             "en-US",
		"Content-Type":      "application/json",
	}
	if c.demo {
		headers["paptrading"] = "1"
	}

	var res bitgetWalletResponse
	if err := httpGetJSON(ctx, c.client, "https://api.bitget.com"+path+"?"+queryString, headers, &res); err != nil {
		return nil, err
	}
	if res.Code != "00000" {
		return nil, fmt.Errorf("bitget: %s (code=%s)", res.Msg, res.Code)
	}
	return res.Data, nil
}

func bitgetProductType(quote string) string {
	if strings.EqualFold(quote, "USDC") {
		return "USDC-FUTURES"
	}
	return "USDT-FUTURES"
}

func bitgetSignQuery(secret string, timestamp int64, method string, requestPath string, queryString string) string {
	payload := strconv.FormatInt(timestamp, 10) + method + requestPath
	if queryString != "" {
		payload += "?" + queryString
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
