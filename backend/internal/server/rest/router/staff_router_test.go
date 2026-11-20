package router_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v3"

	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	"github.com/azhkadaes/vms/backend/internal/server/rest/router"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

// catalogs records the sessions op 08 was asked for and answers with a fixed result.
type catalogs struct {
	result   dto.StaffCatalog
	err      error
	sessions []dto.Session
}

func (c *catalogs) GetPublicCatalogs(context.Context) (dto.PublicCatalog, error) {
	return dto.PublicCatalog{}, errors.New("staff routes must not read the public catalog")
}

func (c *catalogs) GetStaffCatalogs(_ context.Context, session dto.Session) (dto.StaffCatalog, error) {
	c.sessions = append(c.sessions, session)
	return c.result, c.err
}

func staffCatalogApp(usecase *catalogs) *fiber.App {
	app := newApp()
	deps := &router.Dependencies{App: app, Cfg: authConfig(), StaffAuth: restmiddleware.StaffAuth(tokens, authority{}, &limiter{}), Workspace: restmiddleware.Workspace}
	router.RegisterStaffRoutes(deps, usecase)
	app.Use(func(c fiber.Ctx) error { return fiber.ErrNotFound })
	return app
}

var staffCatalog = dto.StaffCatalog{
	Officers: []dto.CatalogOfficer{{ID: "4", Name: "Official Example", VIP: true, Active: false, JobTitle: "Direktur Contoh", Directorate: "Direktorat Contoh", Deputy: "Kedeputian Contoh"}},
	Rooms:    []dto.CatalogRoom{{ID: "3", Name: "Ruang Contoh", Active: true}},
}

func TestStaffCatalogsServeOnlyAReadySessionAndPassItsLiveAuthority(t *testing.T) {
	for _, tc := range []struct {
		name, stage string
		want        reply
	}{
		{"unauthenticated", "", reply{401, "unauthenticated"}},
		{"role selection pending", "role selection", reply{403, "forbidden"}},
		{"password change pending", "password change", reply{403, "forbidden"}},
		{"ready", "ready", reply{200, ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usecase := &catalogs{result: staffCatalog}
			request := httptest.NewRequest("GET", "/v1/staff/catalogs", nil)
			if tc.stage != "" {
				request.Header.Set("Authorization", bearer(t, tc.stage))
			}
			if got := send(t, staffCatalogApp(usecase), request); got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
			if tc.want.status != 200 {
				if len(usecase.sessions) != 0 {
					t.Fatalf("blocked request reached the usecase: %+v", usecase.sessions)
				}
				return
			}
			if len(usecase.sessions) != 1 || !reflect.DeepEqual(usecase.sessions[0], stageSessions["ready"]) {
				t.Fatalf("usecase did not get the live session: %+v", usecase.sessions)
			}
		})
	}
}

func TestStaffCatalogsReturnTheCatalogUnchangedAndUncached(t *testing.T) {
	request := httptest.NewRequest("GET", "/v1/staff/catalogs", nil)
	request.Header.Set("Authorization", bearer(t, "ready"))
	response, err := staffCatalogApp(&catalogs{result: staffCatalog}).Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	var got dto.StaffCatalog
	if response.StatusCode != 200 || json.Unmarshal(raw, &got) != nil || !reflect.DeepEqual(got, staffCatalog) {
		t.Fatalf("status %d body %s", response.StatusCode, raw)
	}
	if response.Header.Get("Cache-Control") != "no-store" || response.Header.Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("headers %v", response.Header)
	}
}

func TestStaffCatalogsMapRefusalsAndRejectQueryStrings(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		err        error
		want       reply
		calls      int
	}{
		{"neither bookings nor people", "/v1/staff/catalogs", failure.New(403, "forbidden"), reply{403, "forbidden"}, 1},
		{"query string", "/v1/staff/catalogs?role=PA", nil, reply{400, "invalid_query_parameter"}, 0},
		{"empty query value", "/v1/staff/catalogs?officer=", nil, reply{400, "invalid_query_parameter"}, 0},
		{"trailing slash is another path", "/v1/staff/catalogs/", nil, reply{404, "not_found"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usecase := &catalogs{result: staffCatalog, err: tc.err}
			request := httptest.NewRequest("GET", tc.path, nil)
			request.Header.Set("Authorization", bearer(t, "ready"))
			if got := send(t, staffCatalogApp(usecase), request); got != tc.want || len(usecase.sessions) != tc.calls {
				t.Fatalf("got %+v after %d usecase calls, want %+v after %d", got, len(usecase.sessions), tc.want, tc.calls)
			}
		})
	}
}
