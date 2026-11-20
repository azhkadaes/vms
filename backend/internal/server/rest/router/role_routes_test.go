package router_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/config"
	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	"github.com/azhkadaes/vms/backend/internal/server/rest/router"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

// roles records what reached the role usecase, so refused requests can prove they reached nothing.
type roles struct {
	calls   []string
	session dto.Session
	proof   entity.SessionProof
	id      int32
	update  dto.RoleUpdate
	err     error
}

func definition(id string, version int32) dto.RoleDefinition {
	return dto.RoleDefinition{ID: id, Name: policy.RoleSuperAdmin, Description: "Administrator.", Permissions: []string{policy.PermBookings, policy.PermRoles},
		AssignablePermissions: []string{policy.PermBookings}, Editable: true, Version: version}
}

func (r *roles) GetRoles(_ context.Context, session dto.Session) ([]dto.RoleDefinition, error) {
	r.calls, r.session = append(r.calls, "GetRoles"), session
	if r.err != nil {
		return nil, r.err
	}
	return []dto.RoleDefinition{definition("4", 1)}, nil
}

func (r *roles) UpdateRole(_ context.Context, proof entity.SessionProof, id int32, update dto.RoleUpdate) (dto.RoleDefinition, error) {
	r.calls, r.proof, r.id, r.update = append(r.calls, "UpdateRole"), proof, id, update
	if r.err != nil {
		return dto.RoleDefinition{}, r.err
	}
	return definition("4", update.Version+1), nil
}

func roleApp() (*fiber.App, *roles) {
	app, usecase := newApp(), &roles{}
	deps := &router.Dependencies{App: app, Cfg: authConfig(), StaffAuth: restmiddleware.StaffAuth(tokens, authority{}, &limiter{}), Workspace: restmiddleware.Workspace}
	router.RegisterRoleRoutes(deps, usecase)
	app.Use(func(c fiber.Ctx) error { return fiber.ErrNotFound })
	return app, usecase
}

func TestRoleListPassesTheLiveSessionToTheUsecaseAndAnswersAnArray(t *testing.T) {
	app, usecase := roleApp()
	request := httptest.NewRequest("GET", "/v1/staff/roles", nil)
	request.Header.Set("Authorization", bearer(t, "ready"))
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	var body []dto.RoleDefinition
	if response.StatusCode != 200 || json.Unmarshal(raw, &body) != nil || len(body) != 1 || body[0].ID != "4" {
		t.Fatalf("got %d %s", response.StatusCode, raw)
	}
	if !slices.Equal(usecase.calls, []string{"GetRoles"}) || usecase.session.ID != "7" || usecase.session.Role != policy.RolePA {
		t.Fatalf("usecase got %v with %+v", usecase.calls, usecase.session)
	}
}

func TestRoleUpdatePassesTheProofPathIDAndBodyToTheUsecase(t *testing.T) {
	app, usecase := roleApp()
	request := httptest.NewRequest("PATCH", "/v1/staff/roles/4", strings.NewReader(`{"permissions":["bookings","roles"],"version":3}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", bearer(t, "ready"))
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	var body dto.RoleDefinition
	if response.StatusCode != 200 || json.Unmarshal(raw, &body) != nil || body.Version != 4 {
		t.Fatalf("got %d %s", response.StatusCode, raw)
	}
	if !slices.Equal(usecase.calls, []string{"UpdateRole"}) || usecase.id != 4 || usecase.proof.SessionID != "ready" || usecase.proof.EmployeeID != 7 ||
		usecase.update.Version != 3 || !slices.Equal(usecase.update.Permissions, []string{"bookings", "roles"}) {
		t.Fatalf("usecase got %v id %d proof %+v update %+v", usecase.calls, usecase.id, usecase.proof, usecase.update)
	}
}

func TestRoleRoutesRefuseMalformedRequestsBeforeTheUsecase(t *testing.T) {
	body := `{"permissions":["bookings"],"version":1}`
	for _, tc := range []struct {
		name, method, path, body, contentType, session string
		want                                           reply
	}{
		{"list without a session", "GET", "/v1/staff/roles", "", "", "", reply{401, "unauthenticated"}},
		{"list while a role selection is pending", "GET", "/v1/staff/roles", "", "", "role selection", reply{403, "forbidden"}},
		{"list while a password change is pending", "GET", "/v1/staff/roles", "", "", "password change", reply{403, "forbidden"}},
		{"list with a query", "GET", "/v1/staff/roles?name=PA", "", "", "ready", reply{400, "invalid_query_parameter"}},
		{"update without a session", "PATCH", "/v1/staff/roles/4", body, "application/json", "", reply{401, "unauthenticated"}},
		{"update while a password change is pending", "PATCH", "/v1/staff/roles/4", body, "application/json", "password change", reply{403, "forbidden"}},
		{"update with a query", "PATCH", "/v1/staff/roles/4?version=1", body, "application/json", "ready", reply{400, "invalid_query_parameter"}},
		{"update a role named instead of numbered", "PATCH", "/v1/staff/roles/PA", body, "application/json", "ready", reply{400, "invalid_request"}},
		{"update role zero", "PATCH", "/v1/staff/roles/0", body, "application/json", "ready", reply{400, "invalid_request"}},
		{"update a padded role ID", "PATCH", "/v1/staff/roles/04", body, "application/json", "ready", reply{400, "invalid_request"}},
		{"update a role ID beyond INTEGER", "PATCH", "/v1/staff/roles/2147483648", body, "application/json", "ready", reply{400, "invalid_request"}},
		{"update with an unknown field", "PATCH", "/v1/staff/roles/4", `{"permissions":[],"version":1,"name":"Root"}`, "application/json", "ready", reply{400, "invalid_request"}},
		{"update with a fractional version", "PATCH", "/v1/staff/roles/4", `{"permissions":[],"version":1.5}`, "application/json", "ready", reply{400, "invalid_request"}},
		{"update with permissions as a string", "PATCH", "/v1/staff/roles/4", `{"permissions":"bookings","version":1}`, "application/json", "ready", reply{400, "invalid_request"}},
		{"update with a trailing value", "PATCH", "/v1/staff/roles/4", body + `{}`, "application/json", "ready", reply{400, "invalid_request"}},
		{"update with form data", "PATCH", "/v1/staff/roles/4", "version=1", "application/x-www-form-urlencoded", "ready", reply{415, "unsupported_media_type"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, usecase := roleApp()
			request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.contentType != "" {
				request.Header.Set("Content-Type", tc.contentType)
			}
			if tc.session != "" {
				request.Header.Set("Authorization", bearer(t, tc.session))
			}
			if got := send(t, app, request); got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
			if len(usecase.calls) != 0 {
				t.Fatalf("a refused request reached %v", usecase.calls)
			}
		})
	}
}

func TestRoleUpdateAnswersUsecaseRefusalsWithTheirStatusAndCode(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want reply
	}{
		{"forbidden", failure.New(403, "forbidden"), reply{403, "forbidden"}},
		{"unknown role", failure.ErrNotFound, reply{404, "not_found"}},
		{"stale version", failure.New(409, "version_conflict"), reply{409, "version_conflict"}},
		{"invalid permissions", failure.Validation().Add("permissions", "Pilih izin dari daftar yang tersedia.").Err(), reply{422, "validation_failed"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, usecase := roleApp()
			usecase.err = tc.err
			request := httptest.NewRequest("PATCH", "/v1/staff/roles/4", strings.NewReader(`{"permissions":["bookings"],"version":1}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", bearer(t, "ready"))
			if got := send(t, app, request); got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestRuntimeRoleRoutesAreMountedBehindStaffAuthentication(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("router-test")
	for _, tc := range []struct {
		name string
		cfg  *config.Config
		want reply
	}{
		{"not configured", &config.Config{}, reply{503, "service_unavailable"}},
		// Token verification fails before any database read, so no pool is needed here.
		{"configured", authConfig(), reply{401, "unauthenticated"}},
	} {
		for _, request := range []struct{ method, path string }{{"GET", "/v1/staff/roles"}, {"PATCH", "/v1/staff/roles/1"}} {
			t.Run(tc.name+"/"+request.method, func(t *testing.T) {
				app := newApp()
				router.RoleRouter(router.NewDependencies(app, nil, tc.cfg, tracer))
				app.Use(func(c fiber.Ctx) error { return fiber.ErrNotFound })
				if got := send(t, app, httptest.NewRequest(request.method, request.path, nil)); got != tc.want {
					t.Fatalf("got %+v, want %+v", got, tc.want)
				}
			})
		}
	}
}
