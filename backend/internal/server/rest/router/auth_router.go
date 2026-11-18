package router

import (
	"github.com/azhkadaes/vms/backend/internal/handler"
	admissionrepository "github.com/azhkadaes/vms/backend/internal/repository/admission/implementation"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	admissionusecase "github.com/azhkadaes/vms/backend/internal/usecase/admission/implementation"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/gofiber/fiber/v3"
)

// AuthRouter registers the seven browser identity operations and private HTML initialization.
func AuthRouter(deps *Dependencies) error {
	if deps.Cfg.Auth == nil {
		deps.App.Use("/v1/staff", func(c fiber.Ctx) error { return fiber.ErrServiceUnavailable })
		return nil
	}
	cfg := *deps.Cfg.Auth
	verifier, err := authusecase.NewCredentialVerifier(cfg.BcryptCost, cfg.HashConcurrency)
	if err != nil {
		return err
	}
	repo := authrepository.NewSessionRepository(deps.DB)
	serializer := authserializer.NewAuthSerializer()
	reader := authusecase.NewAuthUsecase(repo, serializer)
	tokens := security.Tokens{Key: cfg.JWTKey, Issuer: cfg.Issuer, Audience: cfg.Audience}
	lifecycle := authusecase.NewSessionUsecase(repo, reader, verifier, tokens, deps.Cfg.RateKey, cfg.RecoveryKey)
	limiter := admissionusecase.NewRequestLimiter(admissionrepository.NewRequestLimitRepository(deps.DB), deps.Cfg.RateKey)
	h := handler.NewAuthHandler(lifecycle, cfg)
	deps.App.Get("/internal/staff/initialize", h.Initialize)
	group := deps.App.Group("/v1/staff")
	group.Post("/sessions", h.Login)
	group.Post("/session/refresh", h.Refresh)
	group.Delete("/session", h.Logout)
	group.Use(restmiddleware.BearerRequired(tokens, reader, limiter))
	group.Get("/session", h.Current)
	group.Post("/session/renew", h.Renew)
	group.Post("/session/role", h.Role)
	group.Post("/password", h.Password)
	return nil
}
