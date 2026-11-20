package router

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/config"
	"github.com/azhkadaes/vms/backend/internal/infrastructure/storage"
	storageimplementation "github.com/azhkadaes/vms/backend/internal/infrastructure/storage/implementation"
	admissionrepository "github.com/azhkadaes/vms/backend/internal/repository/admission/implementation"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	uploadrepository "github.com/azhkadaes/vms/backend/internal/repository/upload/implementation"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	uploadserializer "github.com/azhkadaes/vms/backend/internal/serializer/upload/implementation"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	admissionusecase "github.com/azhkadaes/vms/backend/internal/usecase/admission/implementation"
	authcontract "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	uploadcontract "github.com/azhkadaes/vms/backend/internal/usecase/upload"
	uploadusecase "github.com/azhkadaes/vms/backend/internal/usecase/upload/implementation"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
)

// Dependencies carries what routers wire from. Every staff business router mounts
// deps.App.Group("/v1/staff/<area>", deps.StaffAuth, deps.Workspace), so its protection
// does not depend on the order in which routers register.
type Dependencies struct {
	App    *fiber.App
	DB     *gorm.DB
	Cfg    *config.Config
	Tracer trace.Tracer
	// StaffAuth is Bearer verification plus live session authority plus the per-employee limit.
	StaffAuth fiber.Handler
	// Workspace blocks sessions that still need a role selected or a password changed.
	Workspace fiber.Handler
	// Uploads serves both upload routes; it answers 503 unless UploadsEnabled.
	Uploads uploadcontract.UploadUsecase
	// BookingFiles shares the same private storage used by uploads.
	BookingFiles storage.Storage
	// UploadsEnabled is true once private storage and the token key are configured and usable.
	UploadsEnabled bool
	// CredentialVerifier shares the bcrypt work limit between authentication and employee creation.
	CredentialVerifier authcontract.CredentialVerifier
}

func NewDependencies(app *fiber.App, db *gorm.DB, cfg *config.Config, tracer trace.Tracer) *Dependencies {
	deps := &Dependencies{App: app, DB: db, Cfg: cfg, Tracer: tracer, StaffAuth: staffAuth(db, cfg), Workspace: restmiddleware.Workspace}
	deps.Uploads, deps.BookingFiles, deps.UploadsEnabled = uploads(db, cfg, tracer)
	return deps
}

// uploads fails closed: without configuration, or when the storage directory is unusable,
// both upload routes answer 503 and the catalogs report attachments as unavailable.
func uploads(db *gorm.DB, cfg *config.Config, tracer trace.Tracer) (uploadcontract.UploadUsecase, storage.Storage, bool) {
	if cfg.Uploads == nil {
		return uploadusecase.NewDisabledUploadUsecase(), nil, false
	}
	files, err := storageimplementation.NewLocalStorage(cfg.Uploads.Dir)
	if err != nil {
		slog.Default().Error("uploads unavailable: private storage initialization failed", "cause", err.Error())
		return uploadusecase.NewDisabledUploadUsecase(), nil, false
	}
	validator := authusecase.NewSessionValidator(authserializer.NewAuthSerializer())
	keys := uploadusecase.Keys{Token: cfg.Uploads.TokenKey, Version: cfg.Uploads.KeyVersion, Rate: cfg.RateKey}
	return uploadusecase.NewUploadUsecase(uploadrepository.NewUploadRepository(db), files, uploadserializer.NewUploadSerializer(), validator, keys, tracer), files, true
}

// staffAuth fails closed with 503 when staff authentication is not configured.
func staffAuth(db *gorm.DB, cfg *config.Config) fiber.Handler {
	if cfg.Auth == nil {
		return func(fiber.Ctx) error { return failure.New(503, "service_unavailable") }
	}
	tokens := security.Tokens{Key: cfg.Auth.JWTKey, Issuer: cfg.Auth.Issuer, Audience: cfg.Auth.Audience}
	reader := authusecase.NewAuthUsecase(authrepository.NewAuthRepository(db), authserializer.NewAuthSerializer())
	limiter := admissionusecase.NewRequestLimiter(admissionrepository.NewRequestLimitRepository(db), cfg.RateKey)
	return restmiddleware.StaffAuth(tokens, reader, limiter)
}
