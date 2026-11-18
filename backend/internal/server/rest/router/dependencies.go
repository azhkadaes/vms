package router

import (
	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/config"
)

type Dependencies struct {
	App    *fiber.App
	DB     *gorm.DB
	Cfg    *config.Config
	Tracer trace.Tracer
}

func NewDependencies(app *fiber.App, db *gorm.DB, cfg *config.Config, tracer trace.Tracer) *Dependencies {
	return &Dependencies{
		App:    app,
		DB:     db,
		Cfg:    cfg,
		Tracer: tracer,
	}
}
