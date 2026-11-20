package handler

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/azhkadaes/vms/backend/config"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/gofiber/fiber/v3"
)

const refreshCookie = "__Host-vms-refresh"
const csrfCookie = "__Host-vms-csrf"
const anonymousCookie = "__Host-vms-anonymous"

type authHandler struct {
	usecase authusecase.SessionUsecase
	cfg     config.Auth
	csrf    security.CSRF
}

func NewAuthHandler(usecase authusecase.SessionUsecase, cfg config.Auth) *authHandler {
	return &authHandler{usecase: usecase, cfg: cfg, csrf: security.CSRF{Key: cfg.CSRFKey}}
}
func (h *authHandler) cookie(c fiber.Ctx, name, value string, expiry time.Time, httpOnly bool) {
	c.Cookie(&fiber.Cookie{Name: name, Value: value, Path: "/", Secure: true, HTTPOnly: httpOnly, SameSite: "Lax", Expires: expiry, MaxAge: max(0, int(time.Until(expiry).Seconds()))})
}
func (h *authHandler) clear(c fiber.Ctx, name string, httpOnly bool) {
	c.Cookie(&fiber.Cookie{Name: name, Path: "/", Secure: true, HTTPOnly: httpOnly, SameSite: "Lax", Expires: time.Unix(1, 0), MaxAge: -1})
}
func (h *authHandler) clearSession(c fiber.Ctx) {
	h.clear(c, refreshCookie, true)
	h.clear(c, csrfCookie, false)
	h.clear(c, anonymousCookie, true)
}
func (h *authHandler) origin(c fiber.Ctx) error {
	if c.Get("Origin") != h.cfg.Origin {
		return failure.New(403, "forbidden")
	}
	if site := c.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" {
		return failure.New(403, "forbidden")
	}
	return nil
}
func (h *authHandler) anonymous(c fiber.Ctx) (string, time.Time, error) {
	raw := c.Cookies(anonymousCookie)
	parts := strings.Split(raw, ".")
	if len(parts) != 3 || len(parts[0]) != 43 || !security.Equal(parts[2], security.Digest(h.cfg.CSRFKey, "anonymous", parts[0]+"."+parts[1])) {
		return "", time.Time{}, failure.New(403, "csrf_expired")
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	expiry := time.Unix(exp, 0)
	if err != nil || !time.Now().Before(expiry) || time.Until(expiry) > 30*time.Minute {
		return "", time.Time{}, failure.New(403, "csrf_expired")
	}
	return raw, expiry, nil
}
func (h *authHandler) lifecycle(c fiber.Ctx, login bool) (string, error) {
	if err := NoQuery(c); err != nil {
		return "", err
	}
	if err := h.origin(c); err != nil {
		return "", err
	}
	var binding string
	if login {
		raw, _, err := h.anonymous(c)
		if err != nil {
			return "", err
		}
		binding = "anonymous:" + raw
	} else {
		id, _, err := h.usecase.Binding(c.Context(), c.Cookies(refreshCookie), true)
		if err != nil {
			return "", err
		}
		binding = "session:" + id
	}
	if h.csrf.Verify(c.Cookies(csrfCookie), c.Get("X-CSRF-Token"), binding, time.Now()) != nil {
		return "", failure.New(403, "forbidden")
	}
	return binding, nil
}
func (h *authHandler) respond(c fiber.Ctx, result authusecase.CredentialResult, replaceAnonymous bool) error {
	if result.Refresh != "" {
		h.cookie(c, refreshCookie, result.Refresh, result.AbsoluteExpiresAt, true)
	}
	if replaceAnonymous {
		token, err := h.csrf.Sign("session:"+result.SessionID, time.Now(), result.AbsoluteExpiresAt)
		if err != nil {
			return err
		}
		h.cookie(c, csrfCookie, token, result.AbsoluteExpiresAt, false)
		h.clear(c, anonymousCookie, true)
	}
	return c.JSON(result.Result)
}
func (h *authHandler) Login(c fiber.Ctx) error {
	if _, err := h.lifecycle(c, true); err != nil {
		return err
	}
	var req dto.Login
	if err := DecodeJSON(c, &req); err != nil {
		return err
	}
	result, err := h.usecase.Login(c.Context(), req, c.IP())
	if err != nil {
		return err
	}
	return h.respond(c, result, true)
}
func (h *authHandler) Current(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	return c.JSON(restmiddleware.Session(c))
}
func (h *authHandler) Refresh(c fiber.Ctx) error {
	if _, err := h.lifecycle(c, false); err != nil {
		return err
	}
	var req dto.Refresh
	if err := DecodeJSON(c, &req); err != nil {
		return err
	}
	result, err := h.usecase.Refresh(c.Context(), c.Cookies(refreshCookie), req.RefreshRequestID)
	if err != nil {
		if errors.Is(err, authusecase.ErrUnauthenticated) {
			h.clearSession(c)
		}
		return err
	}
	return h.respond(c, result, false)
}
func (h *authHandler) Logout(c fiber.Ctx) error {
	if _, err := h.lifecycle(c, false); err != nil {
		if errors.Is(err, authusecase.ErrUnauthenticated) {
			h.clearSession(c)
		}
		return err
	}
	err := h.usecase.Logout(c.Context(), c.Cookies(refreshCookie))
	h.clearSession(c)
	if err != nil {
		return err
	}
	return c.SendStatus(204)
}
func (h *authHandler) Renew(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	var req struct{}
	if err := DecodeJSON(c, &req); err != nil {
		return err
	}
	result, err := h.usecase.Renew(c.Context(), restmiddleware.Proof(c))
	if err != nil {
		return err
	}
	return c.JSON(result)
}
func (h *authHandler) Role(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	var req dto.RoleSelect
	if err := DecodeJSON(c, &req); err != nil {
		return err
	}
	result, err := h.usecase.SwitchRole(c.Context(), restmiddleware.Proof(c), req.Role)
	if err != nil {
		return err
	}
	return h.respond(c, result, false)
}
func (h *authHandler) Password(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	var req dto.PasswordChange
	if err := DecodeJSON(c, &req); err != nil {
		return err
	}
	result, err := h.usecase.ChangePassword(c.Context(), restmiddleware.Proof(c), req)
	if err != nil {
		return err
	}
	return h.respond(c, result, false)
}

// Initialize is an internal gateway subrequest; it never returns access tokens or HTML.
func (h *authHandler) Initialize(c fiber.Ctx) error {
	peer := net.ParseIP(c.RequestCtx().RemoteIP().String())
	allowed := false
	for _, network := range h.cfg.GatewayNetworks {
		if network.Contains(peer) {
			allowed = true
		}
	}
	if !allowed || !security.Equal(c.Get("X-VMS-Gateway-Key"), h.cfg.GatewayKey) {
		return fiber.ErrNotFound
	}
	if err := NoQuery(c); err != nil {
		return err
	}
	method, target := c.Get("X-VMS-Original-Method"), c.Get("X-VMS-Original-URI")
	if method != "GET" || !(target == "/admin" || strings.HasPrefix(target, "/admin/")) {
		return fiber.ErrForbidden
	}
	mode, dest := c.Get("X-VMS-Fetch-Mode"), c.Get("X-VMS-Fetch-Dest")
	if (mode != "" && mode != "navigate") || (dest != "" && dest != "document") {
		return fiber.ErrForbidden
	}
	c.Set("Cache-Control", "private, no-store")
	if secret := c.Cookies(refreshCookie); secret != "" {
		id, expiry, err := h.usecase.Binding(c.Context(), secret, false)
		if err == nil {
			binding := "session:" + id
			existing := c.Cookies(csrfCookie)
			if h.csrf.Verify(existing, existing, binding, time.Now()) == nil {
				return c.SendStatus(204)
			}
			token, err := h.csrf.Sign(binding, time.Now(), expiry)
			if err != nil {
				return err
			}
			h.cookie(c, csrfCookie, token, expiry, false)
			return c.SendStatus(204)
		}
		if !errors.Is(err, authusecase.ErrUnauthenticated) {
			return err
		}
		h.clearSession(c)
	}
	anon, expiry, err := h.anonymous(c)
	if err != nil {
		nonce, err := security.Random()
		if err != nil {
			return err
		}
		expiry = time.Now().Add(30 * time.Minute)
		body := nonce + "." + strconv.FormatInt(expiry.Unix(), 10)
		anon = body + "." + security.Digest(h.cfg.CSRFKey, "anonymous", body)
		h.cookie(c, anonymousCookie, anon, expiry, true)
	}
	current := c.Cookies(csrfCookie)
	if h.csrf.Verify(current, current, "anonymous:"+anon, time.Now()) == nil {
		return c.SendStatus(204)
	}
	token, err := h.csrf.Sign("anonymous:"+anon, time.Now(), expiry)
	if err != nil {
		return err
	}
	h.cookie(c, csrfCookie, token, expiry, false)
	return c.SendStatus(204)
}
