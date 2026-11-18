package handler_test

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/internal/handler"
)

// queryApp echoes what a report-style handler would read from an allow-listed query.
func queryApp() *fiber.App {
	return transportApp(fiber.Config{}, func(app *fiber.App) {
		app.Get("/v1/probe", func(c fiber.Ctx) error {
			query, err := handler.Query(c, "status", "page", "pageSize", "q")
			if err != nil {
				return err
			}
			page, pageSet, err := query.Int("page", 1, 10000)
			if err != nil {
				return err
			}
			size, sizeSet, err := query.Int("pageSize", 1, 100)
			if err != nil {
				return err
			}
			term, err := query.String("q")
			if err != nil {
				return err
			}
			return c.JSON(fiber.Map{"keys": query.Keys(), "status": query.Strings("status"), "statusSet": query.Has("status"),
				"page": page, "pageSet": pageSet, "pageSize": size, "pageSizeSet": sizeSet, "q": term})
		})
	})
}

func get(t *testing.T, target string) reply {
	t.Helper()
	return do(t, queryApp(), httptest.NewRequest("GET", target, nil))
}

func TestQueryRejectsKeysOutsideTheAllowList(t *testing.T) {
	for _, target := range []string{"/v1/probe?role=PA", "/v1/probe?status=approved&Status=approved", "/v1/probe?page=1&page%5B%5D=2"} {
		expectError(t, get(t, target), 400, "invalid_query_parameter")
	}
}

func TestQueryKeepsRepeatedValuesAndKeysInReceivedOrder(t *testing.T) {
	got := get(t, "/v1/probe?pageSize=25&status=checked_in&page=2&status=approved&status=pending")
	if got.status != 200 {
		t.Fatalf("got %d %s", got.status, got.raw)
	}
	if !reflect.DeepEqual(got.body["status"], []any{"checked_in", "approved", "pending"}) {
		t.Fatalf("status order %v", got.body["status"])
	}
	if !reflect.DeepEqual(got.body["keys"], []any{"pageSize", "status", "page"}) {
		t.Fatalf("key order %v", got.body["keys"])
	}
	if got.body["page"] != float64(2) || got.body["pageSize"] != float64(25) {
		t.Fatalf("ints %s", got.raw)
	}
}

func TestQueryTreatsEmptyValuesAsUnset(t *testing.T) {
	got := get(t, "/v1/probe?q=&page=&status=&pageSize")
	if got.status != 200 || got.body["q"] != "" || got.body["pageSet"] != false || got.body["pageSizeSet"] != false || got.body["statusSet"] != false {
		t.Fatalf("got %d %s", got.status, got.raw)
	}
	if !reflect.DeepEqual(got.body["keys"], []any{}) || !reflect.DeepEqual(got.body["status"], []any{}) {
		t.Fatalf("empty values reported as set: %s", got.raw)
	}
	got = get(t, "/v1/probe?status=&status=approved")
	if !reflect.DeepEqual(got.body["status"], []any{"approved"}) {
		t.Fatalf("empty repeat kept: %s", got.raw)
	}
}

func TestQueryEnforcesIntegerBoundsAndCanonicalForm(t *testing.T) {
	for _, value := range []string{"0", "101", "-1", "abc", "1e2", "+5", "05", "2.0", "99999999999999999999"} {
		expectError(t, get(t, "/v1/probe?pageSize="+value), 400, "invalid_query_parameter")
	}
	for value, want := range map[string]float64{"1": 1, "100": 100} {
		got := get(t, "/v1/probe?pageSize="+value)
		if got.status != 200 || got.body["pageSize"] != want {
			t.Fatalf("pageSize=%s: %d %s", value, got.status, got.raw)
		}
	}
}

func TestQueryDecodesTextAndRejectsAmbiguousOrMalformedValues(t *testing.T) {
	got := get(t, "/v1/probe?q=Budi+Santoso%20caf%C3%A9")
	if got.status != 200 || got.body["q"] != "Budi Santoso café" {
		t.Fatalf("got %d %s", got.status, got.raw)
	}
	for _, target := range []string{"/v1/probe?q=a&q=b", "/v1/probe?q=%zz", "/v1/probe?q=%FF", "/v1/probe?q=a;b=c"} {
		expectError(t, get(t, target), 400, "invalid_query_parameter")
	}
}
