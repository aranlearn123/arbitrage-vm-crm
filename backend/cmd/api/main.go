package main

import (
	"log"
	"strings"

	"arbitrage-vm-crm-backend/internal/config"
	"arbitrage-vm-crm-backend/internal/handler"

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

	healthHandler := handler.NewHealthHandler()

	api := app.Group("/api/v1")
	api.Get("/health", healthHandler.Check)

	log.Printf("starting api server on :%s", cfg.AppPort)
	app.Get("/swagger/*", swagger.WrapHandler)
	if err := app.Listen(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}
