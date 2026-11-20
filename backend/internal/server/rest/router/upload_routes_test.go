package router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/internal/entity"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	"github.com/azhkadaes/vms/backend/internal/server/rest/router"
	uploadusecase "github.com/azhkadaes/vms/backend/internal/usecase/upload"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

const uploadReference = "0f8fad5b-d9cb-469f-a165-70867728950e"

// uploads records what reached the upload usecase.
type uploads struct {
	calls   []string
	ip      string
	proof   entity.SessionProof
	got     entity.UploadRequest
	replay  bool
	failure error
}

func (u *uploads) issue() (uploadusecase.Issued, error) {
	if u.failure != nil {
		return uploadusecase.Issued{}, u.failure
	}
	return uploadusecase.Issued{Created: !u.replay, Result: dto.UploadResult{Token: "vmsu1." + strings.Repeat("b", 43), ExpiresAt: time.Date(2030, 1, 2, 1, 0, 0, 0, time.UTC), ServerTime: 1893542400000}}, nil
}
func (u *uploads) UploadPublic(_ context.Context, ip string, upload entity.UploadRequest) (uploadusecase.Issued, error) {
	u.calls, u.ip, u.got = append(u.calls, "UploadPublic"), ip, upload
	return u.issue()
}
func (u *uploads) UploadStaff(_ context.Context, proof entity.SessionProof, ip string, upload entity.UploadRequest) (uploadusecase.Issued, error) {
	u.calls, u.ip, u.proof, u.got = append(u.calls, "UploadStaff"), ip, proof, upload
	return u.issue()
}

type publicCatalog struct{}

func (publicCatalog) GetPublicCatalogs(context.Context) (dto.PublicCatalog, error) {
	return dto.PublicCatalog{Officers: []dto.CatalogOfficer{}, Rooms: []dto.CatalogRoom{}}, nil
}
func (publicCatalog) GetStaffCatalogs(context.Context, dto.Session) (dto.StaffCatalog, error) {
	return dto.StaffCatalog{Officers: []dto.CatalogOfficer{}, Rooms: []dto.CatalogRoom{}}, nil
}

type publicAdmission struct {
	calls   int
	refused bool
}

func (a *publicAdmission) PublicRequest(context.Context, string) (entity.RequestAdmission, error) {
	a.calls++
	return entity.RequestAdmission{Allowed: !a.refused, RetryAfter: 7}, nil
}

func uploadApp(u *uploads, admission *publicAdmission) *fiber.App {
	app := newApp()
	deps := &router.Dependencies{App: app, Cfg: authConfig(), StaffAuth: restmiddleware.StaffAuth(tokens, authority{}, &limiter{}), Workspace: restmiddleware.Workspace}
	router.RegisterPublicRoutes(app, publicCatalog{}, admission, u)
	router.RegisterStaffUploadRoutes(deps, u)
	app.Use(func(c fiber.Ctx) error { return fiber.ErrNotFound })
	return app
}

func uploadForm(t *testing.T) (string, *bytes.Buffer) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	file, err := writer.CreateFormFile("file", "nota dinas.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(file, "%PDF-1.7 synthetic"); err != nil {
		t.Fatal(err)
	}
	for _, field := range [][2]string{{"reference", uploadReference}, {"purpose", "document"}} {
		if err := writer.WriteField(field[0], field[1]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return writer.FormDataContentType(), body
}

type uploadReply struct {
	status int
	code   string
	body   map[string]any
	retry  string
}

func sendUpload(t *testing.T, app *fiber.App, request *http.Request) uploadReply {
	t.Helper()
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	out := uploadReply{status: response.StatusCode, retry: response.Header.Get("Retry-After")}
	if err := json.Unmarshal(raw, &out.body); err != nil {
		t.Fatalf("not JSON: %s", raw)
	}
	out.code, _ = out.body["code"].(string)
	return out
}

func uploadRequest(t *testing.T, path, stage string) *http.Request {
	t.Helper()
	contentType, body := uploadForm(t)
	request := httptest.NewRequest("POST", path, body)
	request.Header.Set("Content-Type", contentType)
	if stage != "" {
		request.Header.Set("Authorization", bearer(t, stage))
	}
	return request
}

func TestUploadsAnswer201WhenCreatedAnd200WhenReplayed(t *testing.T) {
	for _, path := range []string{"/v1/public/attachments", "/v1/staff/attachments"} {
		for _, replay := range []bool{false, true} {
			u := &uploads{replay: replay}
			got := sendUpload(t, uploadApp(u, &publicAdmission{}), uploadRequest(t, path, "ready"))
			want := 201
			if replay {
				want = 200
			}
			if got.status != want || got.body["token"] != "vmsu1."+strings.Repeat("b", 43) || got.body["expiresAt"] != "2030-01-02T01:00:00Z" || got.body["serverTime"] != float64(1893542400000) || len(got.body) != 3 {
				t.Fatalf("%s replay=%v: %+v", path, replay, got)
			}
			if u.got.Reference != uploadReference || u.got.Purpose != "document" || u.got.Filename != "nota dinas.pdf" || string(u.got.Content) != "%PDF-1.7 synthetic" || u.ip == "" {
				t.Fatalf("usecase received %+v from %q", u.got, u.ip)
			}
		}
	}
}

func TestTheStaffUploadRouteNeedsAReadySessionAndPassesItsProof(t *testing.T) {
	for _, tc := range []struct {
		stage string
		want  int
		code  string
	}{
		{"", 401, "unauthenticated"},
		{"unknown", 401, "unauthenticated"},
		{"role selection", 403, "forbidden"},
		{"password change", 403, "forbidden"},
		{"ready", 201, ""},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			u := &uploads{}
			got := sendUpload(t, uploadApp(u, &publicAdmission{}), uploadRequest(t, "/v1/staff/attachments", tc.stage))
			if got.status != tc.want || got.code != tc.code {
				t.Fatalf("got %+v", got)
			}
			if tc.want != 201 {
				if len(u.calls) != 0 {
					t.Fatalf("a refused request reached the usecase: %v", u.calls)
				}
				return
			}
			if len(u.calls) != 1 || u.calls[0] != "UploadStaff" || u.proof.SessionID != "ready" || u.proof.EmployeeID != 7 {
				t.Fatalf("calls %v proof %+v", u.calls, u.proof)
			}
		})
	}
}

func TestThePublicUploadRouteSitsBehindThePublicCeiling(t *testing.T) {
	u, admission := &uploads{}, &publicAdmission{refused: true}
	got := sendUpload(t, uploadApp(u, admission), uploadRequest(t, "/v1/public/attachments", ""))
	if got.status != 429 || got.code != "rate_limited" || got.retry != "7" || len(u.calls) != 0 || admission.calls != 1 {
		t.Fatalf("got %+v, usecase calls %v, admissions %d", got, u.calls, admission.calls)
	}
	admission.refused = false
	if got := sendUpload(t, uploadApp(u, admission), uploadRequest(t, "/v1/public/attachments", "")); got.status != 201 {
		t.Fatalf("admitted upload: %+v", got)
	}
}

func TestUploadRoutesRejectQueriesAndNonMultipartBodiesBeforeTheUsecase(t *testing.T) {
	for _, path := range []string{"/v1/public/attachments", "/v1/staff/attachments"} {
		u := &uploads{}
		app := uploadApp(u, &publicAdmission{})
		query := uploadRequest(t, path+"?purpose=document", "ready")
		if got := sendUpload(t, app, query); got.status != 400 || got.code != "invalid_query_parameter" {
			t.Fatalf("%s with a query: %+v", path, got)
		}
		raw := httptest.NewRequest("POST", path, strings.NewReader("%PDF-1.7 synthetic"))
		raw.Header.Set("Content-Type", "application/pdf")
		raw.Header.Set("Authorization", bearer(t, "ready"))
		if got := sendUpload(t, app, raw); got.status != 415 || got.code != "unsupported_media_type" {
			t.Fatalf("%s with a raw PDF body: %+v", path, got)
		}
		if len(u.calls) != 0 {
			t.Fatalf("rejected requests reached the usecase: %v", u.calls)
		}
	}
}

func TestUploadUsecaseFailuresKeepTheirStatusAndRetryAfter(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{failure.New(409, "idempotency_conflict"), 409, "idempotency_conflict"},
		{failure.Validation().Add("file", "Gunakan berkas PDF yang valid.").Err(), 422, "validation_failed"},
		{&failure.Error{Status: 429, Code: "rate_limited", RetryAfter: 1800}, 429, "rate_limited"},
		{failure.New(503, "service_unavailable"), 503, "service_unavailable"},
	} {
		got := sendUpload(t, uploadApp(&uploads{failure: tc.err}, &publicAdmission{}), uploadRequest(t, "/v1/public/attachments", ""))
		if got.status != tc.status || got.code != tc.code {
			t.Fatalf("got %+v, want %d %s", got, tc.status, tc.code)
		}
		if tc.status == 429 && got.retry != "1800" {
			t.Fatalf("Retry-After %q", got.retry)
		}
	}
}
