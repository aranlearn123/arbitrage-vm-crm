package main

import (
	"context"
	"log"
	"strings"

	"arbitrage-vm-crm-backend/internal/config"
	"arbitrage-vm-crm-backend/internal/database"
	"arbitrage-vm-crm-backend/internal/handler"
	"arbitrage-vm-crm-backend/internal/repo"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	_ "arbitrage-vm-crm-backend/docs"

	swagger "github.com/swaggo/fiber-swagger"
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

	app := fiber.New(fiber.Config{
		AppName: "arbitrage-vm-crm-backend",
	})

	app.Use(recover.New())
	app.Use(fiberlogger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: strings.Join(cfg.CORSOrigins, ","),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	}))

	healthHandler := handler.NewHealthHandler(db)
	allocationRepo := repo.NewAllocationRepo(db)
	allocationHandler := handler.NewAllocationHandler(allocationRepo)
	fundingRepo := repo.NewFundingRepo(db)
	fundingHandler := handler.NewFundingHandler(fundingRepo)
	marketQualityRepo := repo.NewMarketQualityRepo(db)
	marketQualityHandler := handler.NewMarketQualityHandler(marketQualityRepo)

	api := app.Group("/api/v1")
	api.Get("/health", healthHandler.Check)
	api.Get("/allocations", allocationHandler.List)
	api.Get("/allocations/summary", allocationHandler.Summary)
	api.Get("/allocations/active", allocationHandler.Active)
	api.Get("/allocations/running", allocationHandler.Running)
	api.Get("/allocations/cancelled/reasons", allocationHandler.CancelledReasons)
	api.Get("/allocations/cancelled", allocationHandler.Cancelled)
	api.Get("/allocations/:id/timeline", allocationHandler.Timeline)
	api.Get("/allocations/:id", allocationHandler.Detail)
	api.Get("/funding/latest", fundingHandler.Latest)
	api.Get("/funding/history", fundingHandler.History)
	api.Get("/funding/spread", fundingHandler.Spread)
	api.Get("/funding/top-spreads", fundingHandler.TopSpreads)
	api.Get("/market-quality/latest", marketQualityHandler.Latest)
	api.Get("/market-quality/history", marketQualityHandler.History)
	api.Get("/market-quality/alerts", marketQualityHandler.Alerts)

	log.Printf("starting api server on :%s", cfg.AppPort)
	app.Get("/swagger/*", swagger.WrapHandler)
	if err := app.Listen(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}
