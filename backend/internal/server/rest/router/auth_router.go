package router

import (
	"github.com/azhkadaes/vms/backend/internal/handler"
	"github.com/azhkadaes/vms/backend/internal/policy"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	authcontract "github.com/azhkadaes/vms/backend/internal/usecase/auth"
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
	deps.CredentialVerifier = verifier
	repo := authrepository.NewSessionRepository(deps.DB)
	reader := authusecase.NewAuthUsecase(repo, authserializer.NewAuthSerializer())
	tokens := security.Tokens{Key: cfg.JWTKey, Issuer: cfg.Issuer, Audience: cfg.Audience}
	lifecycle := authusecase.NewSessionUsecase(repo, reader, verifier, tokens, deps.Cfg.RateKey, cfg.RecoveryKey)
	RegisterAuthRoutes(deps, lifecycle)
	return nil
}

// RegisterAuthRoutes shares the session routes between runtime and injected usecases. Login,
// refresh and logout are cookie-bound. The Bearer operations take deps.StaffAuth per route and
// then the A-18 gate for their operation, so nothing here protects or blocks another staff route.
func RegisterAuthRoutes(deps *Dependencies, lifecycle authcontract.SessionUsecase) {
	h := handler.NewAuthHandler(lifecycle, *deps.Cfg.Auth)
	deps.App.Get("/internal/staff/initialize", h.Initialize)
	group := deps.App.Group("/v1/staff")
	group.Post("/sessions", h.Login)
	group.Post("/session/refresh", h.Refresh)
	group.Delete("/session", h.Logout)
	group.Get("/session", deps.StaffAuth, restmiddleware.SessionOperation(policy.OpGetSession), h.Current)
	group.Post("/session/renew", deps.StaffAuth, restmiddleware.SessionOperation(policy.OpRenewSession), h.Renew)
	group.Post("/session/role", deps.StaffAuth, restmiddleware.SessionOperation(policy.OpSelectRole), h.Role)
	group.Post("/password", deps.StaffAuth, restmiddleware.SessionOperation(policy.OpChangePassword), h.Password)
}
