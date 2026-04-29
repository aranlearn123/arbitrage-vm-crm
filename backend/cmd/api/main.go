package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"arbitrage-vm-crm-backend/internal/config"
	"arbitrage-vm-crm-backend/internal/database"
	"arbitrage-vm-crm-backend/internal/exchange"
	"arbitrage-vm-crm-backend/internal/handler"
	"arbitrage-vm-crm-backend/internal/repo"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	_ "arbitrage-vm-crm-backend/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Arbitrage VM CRM API
// @version 0.1.0
// @description CRM monitoring API for arbitrage trading system.
// @BasePath /api/v1
func main() {
	cfg := config.Load()

	db, err := database.Open(context.Background(), database.Config{
		DatabaseURL: cfg.DatabaseURL,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := database.Close(db); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	allowedOrigins := cfg.CORSOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}))

	healthHandler := handler.NewHealthHandler(db)
	systemRepo := repo.NewSystemRepo(db)
	systemHandler := handler.NewSystemHandler(db, systemRepo, cfg)
	allocationRepo := repo.NewAllocationRepo(db)
	allocationHandler := handler.NewAllocationHandler(allocationRepo)
	fundingRepo := repo.NewFundingRepo(db)
	fundingHandler := handler.NewFundingHandler(fundingRepo)
	marketQualityRepo := repo.NewMarketQualityRepo(db)
	marketQualityHandler := handler.NewMarketQualityHandler(marketQualityRepo)
	pnlRepo := repo.NewPnLRepo(db)
	pnlHandler := handler.NewPnLHandler(pnlRepo)
	equityService := exchange.NewEquityService(exchange.Config{
		CacheTTL:         time.Duration(cfg.EquityCacheTTLSeconds) * time.Second,
		BybitDemo:        cfg.BybitDemo,
		BybitAPIKey:      cfg.BybitCredential.APIKey,
		BybitAPISecret:   cfg.BybitCredential.APISecret,
		BitgetDemo:       cfg.BitgetDemo,
		BitgetAPIKey:     cfg.BitgetCredential.APIKey,
		BitgetAPISecret:  cfg.BitgetCredential.APISecret,
		BitgetPassphrase: cfg.BitgetCredential.Passphrase,
	})
	equityHandler := handler.NewEquityHandler(equityService)

	api := chi.NewRouter()
	api.Get("/health", handler.Wrap(healthHandler.Check))
	api.Get("/system/status", handler.Wrap(systemHandler.Status))
	api.Get("/system/exchanges", handler.Wrap(systemHandler.Exchanges))
	api.Get("/allocations", handler.Wrap(allocationHandler.List))
	api.Get("/allocations/summary", handler.Wrap(allocationHandler.Summary))
	api.Get("/allocations/active", handler.Wrap(allocationHandler.Active))
	api.Get("/allocations/running", handler.Wrap(allocationHandler.Running))
	api.Get("/allocations/cancelled/reasons", handler.Wrap(allocationHandler.CancelledReasons))
	api.Get("/allocations/cancelled", handler.Wrap(allocationHandler.Cancelled))
	api.Get("/allocations/{id}/timeline", handler.Wrap(allocationHandler.Timeline))
	api.Get("/allocations/{id}", handler.Wrap(allocationHandler.Detail))
	api.Get("/funding/latest", handler.Wrap(fundingHandler.Latest))
	api.Get("/funding/history", handler.Wrap(fundingHandler.History))
	api.Get("/funding/spread", handler.Wrap(fundingHandler.Spread))
	api.Get("/funding/top-spreads", handler.Wrap(fundingHandler.TopSpreads))
	api.Get("/market-quality/latest", handler.Wrap(marketQualityHandler.Latest))
	api.Get("/market-quality/history", handler.Wrap(marketQualityHandler.History))
	api.Get("/market-quality/alerts", handler.Wrap(marketQualityHandler.Alerts))
	api.Get("/pnl/events", handler.Wrap(pnlHandler.Events))
	api.Get("/pnl/summary", handler.Wrap(pnlHandler.Summary))
	api.Get("/pnl/by-pair", handler.Wrap(pnlHandler.ByPair))
	api.Get("/pnl/by-exchange", handler.Wrap(pnlHandler.ByExchange))
	api.Get("/pnl/by-component", handler.Wrap(pnlHandler.ByComponent))
	api.Get("/equity/latest", handler.Wrap(equityHandler.Latest))
	api.Get("/equity/live", handler.Wrap(equityHandler.Latest))

	router.Mount("/api/v1", api)
	router.Get("/swagger/*", httpSwagger.WrapHandler)
	log.Printf("starting api server on :%s", cfg.AppPort)
	if err := http.ListenAndServe(":"+cfg.AppPort, router); err != nil {
		log.Fatal(err)
	}
}
