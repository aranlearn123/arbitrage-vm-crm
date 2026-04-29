package handler

import (
	"time"

	"arbitrage-vm-crm-backend/internal/response"

	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Check godoc
// @Summary Health check
// @Description Check API service status
// @Tags System
// @Produce json
// @Success 200 {object} response.Health
// @Router /health [get]
func (h *HealthHandler) Check(c *fiber.Ctx) error {
	return c.JSON(response.Health{
		Status: "ok",
		Time:   time.Now().UTC().Format(time.RFC3339),
	})
}
