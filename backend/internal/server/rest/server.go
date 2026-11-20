package restserver

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/config"
	"github.com/azhkadaes/vms/backend/internal/handler"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	"github.com/azhkadaes/vms/backend/internal/server/rest/router"
	uploadusecase "github.com/azhkadaes/vms/backend/internal/usecase/upload"
	databasehelper "github.com/azhkadaes/vms/backend/pkg/helper/database"
)

type Health interface {
	PingContext(ctx context.Context) error
}

type RestServer struct {
	app    *fiber.App
	cfg    config.Config
	db     *gorm.DB
	ctx    context.Context
	logger *slog.Logger
	tracer trace.Tracer
}

func NewRestServer(ctx context.Context, cfg *config.Config) *RestServer {
	return &RestServer{ctx: ctx, cfg: *cfg, logger: slog.Default(), tracer: otel.Tracer("vms/backend")}
}

func (s *RestServer) Start() error {
	connectCtx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	db, pool, err := databasehelper.NewGormDB(connectCtx, s.cfg.DatabaseURL)
	cancel()
	if err != nil {
		return err
	}
	defer pool.Close()
	s.db = db
	s.app = newApp(s.cfg, databasehelper.Readiness{Pool: pool}, s.logger, s.tracer)
	s.RegisterRoutes()
	stopped := make(chan error, 1)
	go func() { stopped <- s.app.Listen(s.cfg.ListenAddress) }()
	select {
	case err := <-stopped:
		if err != nil {
			return errors.New("HTTP listener failed")
		}
		return nil
	case <-s.ctx.Done():
		return s.Shutdown()
	}
}

func (s *RestServer) Shutdown() error {
	if s.app == nil {
		return nil
	}
	// Drain against a fresh context: the signal context is already canceled.
	drain, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.app.ShutdownWithContext(drain); err != nil {
		return errors.New("HTTP shutdown deadline exceeded")
	}
	return nil
}

func (s *RestServer) RegisterRoutes() {
	dependencies := router.NewDependencies(s.app, s.db, &s.cfg, s.tracer)
	router.PublicRouter(dependencies)
	if err := router.AuthRouter(dependencies); err != nil {
		s.app.Use("/v1/staff", func(c fiber.Ctx) error { return fiber.ErrServiceUnavailable })
		s.logger.Error("staff routes unavailable: password verifier initialization failed")
	}
	// After the fallback above, so a failed verifier also closes the staff business areas.
	router.StaffRouter(dependencies)
	router.UploadRouter(dependencies)
	router.RoleRouter(dependencies)
	s.app.Use(func(c fiber.Ctx) error { return fiber.ErrNotFound })
}

// uploadPaths take a multipart body up to the upload cap (D67) within the upload receive
// deadline (D69); every other route keeps the JSON body cap (D66) and the server read timeout.
var uploadPaths = []string{"/v1/public/attachments", "/v1/staff/attachments"}

const uploadReceiveTimeout = 60 * time.Second

// requestLimits runs once the request headers are read, before any body byte is.
func requestLimits(header *fasthttp.RequestHeader) fasthttp.RequestConfig {
	path, _, _ := strings.Cut(string(header.RequestURI()), "?")
	if !header.IsPost() || !slices.Contains(uploadPaths, path) {
		return fasthttp.RequestConfig{}
	}
	return fasthttp.RequestConfig{MaxRequestBodySize: handler.MultipartBodyLimit(uploadusecase.MaxFileBytes), ReadTimeout: uploadReceiveTimeout}
}

func newApp(cfg config.Config, health Health, logger *slog.Logger, tracer trace.Tracer) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:          "VMS",
		Immutable:        true,
		CaseSensitive:    true,
		StrictRouting:    true,
		BodyLimit:        cfg.JSONMaxBytes,
		ReadTimeout:      15 * time.Second,
		WriteTimeout:     20 * time.Second,
		IdleTimeout:      60 * time.Second,
		TrustProxy:       true,
		TrustProxyConfig: fiber.TrustProxyConfig{Proxies: cfg.TrustedProxies},
		ProxyHeader:      "X-Forwarded-For",
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return restmiddleware.Failure(c, err, logger)
		},
		// Multipart bodies stay raw until handler.Multipart bounds and parses them.
		DisablePreParseMultipartForm: true,
	})
	app.Server().HeaderReceived = requestLimits
	app.Use(recover.New())
	app.Use(restmiddleware.RequestContext(cfg.ExecutionTimeout, tracer))
	app.Get("/health/live", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	app.Get("/health/ready", func(c fiber.Ctx) error {
		if err := health.PingContext(c.Context()); err != nil {
			return fiber.ErrServiceUnavailable
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})
	router.SwaggerRouter(app, cfg)
	return app
}
