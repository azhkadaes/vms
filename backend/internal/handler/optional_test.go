package handler_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/internal/handler"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type editProbe struct {
	RoomID dto.Optional[string] `json:"roomId"`
	Note   dto.Optional[string] `json:"coordinationNote"`
	Seats  dto.Optional[int]    `json:"seats"`
}

func state[T any](o dto.Optional[T]) string {
	switch {
	case !o.Set:
		return "omitted"
	case o.Null:
		return "null"
	}
	return "value"
}

func optionalApp() *fiber.App {
	return transportApp(fiber.Config{}, func(app *fiber.App) {
		app.Post("/v1/probe", func(c fiber.Ctx) error {
			var req editProbe
			if err := handler.DecodeJSON(c, &req); err != nil {
				return err
			}
			return c.JSON(fiber.Map{"room": state(req.RoomID), "roomValue": req.RoomID.Value, "note": state(req.Note), "noteValue": req.Note.Value, "seats": state(req.Seats), "seatsValue": req.Seats.Value})
		})
	})
}

func post(t *testing.T, body string) reply {
	t.Helper()
	request := httptest.NewRequest("POST", "/v1/probe", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return do(t, optionalApp(), request)
}

func TestOptionalDistinguishesOmittedNullAndValueUnderStrictDecoding(t *testing.T) {
	for _, tc := range []struct {
		body, room, roomValue, note, noteValue, seats string
		seatsValue                                    float64
	}{
		{`{}`, "omitted", "", "omitted", "", "omitted", 0},
		{`{"roomId":null}`, "null", "", "omitted", "", "omitted", 0},
		{`{"roomId":"12","coordinationNote":""}`, "value", "12", "value", "", "omitted", 0},
		{`{"coordinationNote":"Bawa KTP","seats":0}`, "omitted", "", "value", "Bawa KTP", "value", 0},
		{`{"roomId":"12","roomId":null,"seats":3}`, "null", "", "omitted", "", "value", 3},
	} {
		t.Run(tc.body, func(t *testing.T) {
			got := post(t, tc.body)
			if got.status != 200 {
				t.Fatalf("got %d %s", got.status, got.raw)
			}
			if got.body["room"] != tc.room || got.body["roomValue"] != tc.roomValue || got.body["note"] != tc.note || got.body["noteValue"] != tc.noteValue ||
				got.body["seats"] != tc.seats || got.body["seatsValue"] != tc.seatsValue {
				t.Fatalf("got %s", got.raw)
			}
		})
	}
}

func TestOptionalStillRejectsWrongTypesAndUnknownFields(t *testing.T) {
	for _, body := range []string{`{"roomId":12}`, `{"seats":"3"}`, `{"roomId":null,"room":"A"}`, `{"roomId":"12"} {}`} {
		expectError(t, post(t, body), 400, "invalid_request")
	}
}
